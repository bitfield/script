package script

import (
	"bufio"
	"container/ring"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"bitbucket.org/creachadair/shell"
	"github.com/itchyny/gojq"
)

// Pipe represents a pipe object with an associated [ReadAutoCloser].
type Pipe struct {
	// Reader is the underlying reader.
	Reader     ReadAutoCloser
	stdout     io.Writer
	httpClient *http.Client

	// because pipe stages are concurrent, protect 'err'
	mu  *sync.Mutex
	err error
}

// Args creates a pipe containing the program's command-line arguments from
// [os.Args], excluding the program name, one per line.
func Args() *Pipe {
	var s strings.Builder
	for _, a := range os.Args[1:] {
		s.WriteString(a + "\n")
	}
	return Echo(s.String())
}

// Do creates a pipe that makes the HTTP request req and produces the response.
// See [Pipe.Do] for how the HTTP response status is interpreted.
func Do(req *http.Request) *Pipe {
	return NewPipe().Do(req)
}

// Echo creates a pipe containing the string s.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec creates a pipe that runs cmdLine as an external command and produces
// its combined output (interleaving standard output and standard error). See
// [Pipe.Exec] for error handling details.
//
// Use [Pipe.Exec] to send the contents of an existing pipe to the command's
// standard input.
func Exec(cmdLine string) *Pipe {
	return NewPipe().Exec(cmdLine)
}

// File creates a pipe that reads from the file path.
func File(path string) *Pipe {
	p := NewPipe()
	f, err := os.Open(path)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}

// FindFiles creates a pipe listing all the files in the directory path and its
// subdirectories recursively, one per line, like Unix find(1). If path doesn't
// exist or can't be read, the pipe's error status will be set.
//
// Each line of the output consists of a slash-separated path, starting with
// the initial directory. For example, if the directory looks like this:
//
//	test/
//	        1.txt
//	        2.txt
//
// the pipe's output will be:
//
//	test/1.txt
//	test/2.txt
func FindFiles(path string) *Pipe {
	var fileNames []string
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileNames = append(fileNames, path)
		}
		return nil
	}
	if err := filepath.Walk(path, walkFn); err != nil {
		return NewPipe().WithError(err)
	}
	return Slice(fileNames)
}

// Get creates a pipe that makes an HTTP GET request to URL, and produces the
// response. See [Pipe.Do] for how the HTTP response status is interpreted.
func Get(URL string) *Pipe {
	return NewPipe().Get(URL)
}

// IfExists tests whether path exists, and creates a pipe whose error status
// reflects the result. If the file doesn't exist, the pipe's error status will
// be set, and if the file does exist, the pipe will have no error status. This
// can be used to do some operation only if a given file exists:
//
//	IfExists("/foo/bar").Exec("/usr/bin/something")
func IfExists(path string) *Pipe {
	p := NewPipe()
	_, err := os.Stat(path)
	if err != nil {
		return p.WithError(err)
	}
	return p
}

// ListFiles creates a pipe containing the files or directories specified by
// path, one per line. path can be a glob expression, as for [filepath.Match].
// For example:
//
//	ListFiles("/data/*").Stdout()
//
// ListFiles does not recurse into subdirectories; use [FindFiles] instead.
func ListFiles(path string) *Pipe {
	if strings.ContainsAny(path, "[]^*?\\{}!") {
		fileNames, err := filepath.Glob(path)
		if err != nil {
			return NewPipe().WithError(err)
		}
		return Slice(fileNames)
	}
	files, err := os.ReadDir(path)
	if err != nil {
		// Check for the case where the path matches exactly one file
		s, err := os.Stat(path)
		if err != nil {
			return NewPipe().WithError(err)
		}
		if !s.IsDir() {
			return Echo(path)
		}
		return NewPipe().WithError(err)
	}
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = filepath.Join(path, f.Name())
	}
	return Slice(fileNames)
}

// NewPipe creates a new pipe with an empty reader (use [Pipe.WithReader] to
// attach another reader to it).
func NewPipe() *Pipe {
	return &Pipe{
		Reader:     ReadAutoCloser{},
		mu:         &sync.Mutex{},
		err:        nil,
		stdout:     os.Stdout,
		httpClient: http.DefaultClient,
	}
}

// Post creates a pipe that makes an HTTP POST request to URL, with an empty
// body, and produces the response. See [Pipe.Do] for how the HTTP response
// status is interpreted.
func Post(URL string) *Pipe {
	return NewPipe().Post(URL)
}

// Slice creates a pipe containing each element of s, one per line.
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// Stdin creates a pipe that reads from [os.Stdin].
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

// AppendFile appends the contents of the pipe to the file path, creating it if
// necessary, and returns the number of bytes successfully written, or an
// error.
func (p *Pipe) AppendFile(path string) (int64, error) {
	return p.writeOrAppendFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

var exitStatusPattern = regexp.MustCompile(`exit status (\d+)$`)

// Basename reads paths from the pipe, one per line, and removes any leading
// directory components from each. So, for example, /usr/local/bin/foo would
// become just foo. This is the complementary operation to [Pipe.Dirname].
//
// If any line is empty, Basename will transform it to a single dot. Trailing
// slashes are removed. The behaviour of Basename is the same as
// [filepath.Base] (not by coincidence).
func (p *Pipe) Basename() *Pipe {
	return p.FilterLine(filepath.Base)
}

// Bytes returns the contents of the pipe as a []byte, or an error.
func (p *Pipe) Bytes() ([]byte, error) {
	if p.Error() != nil {
		return nil, p.Error()
	}
	data, err := io.ReadAll(p)
	if err != nil {
		p.SetError(err)
	}
	return data, nil
}

// Close closes the pipe's associated reader. This is a no-op if the reader is
// not an [io.Closer].
func (p *Pipe) Close() error {
	return p.Reader.Close()
}

// Column produces column col of each line of input, where the first column is
// column 1, and columns are delimited by Unicode whitespace. Lines containing
// fewer than col columns will be skipped.
func (p *Pipe) Column(col int) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		columns := strings.Fields(line)
		if col > 0 && col <= len(columns) {
			fmt.Fprintln(w, columns[col-1])
		}
	})
}

// Concat reads paths from the pipe, one per line, and produces the contents of
// all the corresponding files in sequence. If there are any errors (for
// example, non-existent files), these will be ignored, execution will
// continue, and the pipe's error status will not be set.
//
// This makes it convenient to write programs that take a list of paths on the
// command line. For example:
//
//	script.Args().Concat().Stdout()
//
// The list of paths could also come from a file:
//
//	script.File("filelist.txt").Concat()
//
// Or from the output of a command:
//
//	script.Exec("ls /var/app/config/").Concat().Stdout()
//
// Each input file will be closed once it has been fully read. If any of the
// files can't be opened or read, Concat will simply skip these and carry on,
// without setting the pipe's error status. This mimics the behaviour of Unix
// cat(1).
func (p *Pipe) Concat() *Pipe {
	var readers []io.Reader
	p.FilterScan(func(line string, w io.Writer) {
		input, err := os.Open(line)
		if err != nil {
			return // skip errors
		}
		readers = append(readers, NewReadAutoCloser(input))
	}).Wait()
	return p.WithReader(io.MultiReader(readers...))
}

// CountLines returns the number of lines of input, or an error.
func (p *Pipe) CountLines() (int, error) {
	lines := 0
	p.FilterScan(func(line string, w io.Writer) {
		lines++
	}).Wait()
	return lines, p.Error()
}

// Dirname reads paths from the pipe, one per line, and produces only the
// parent directories of each path. For example, /usr/local/bin/foo would
// become just /usr/local/bin. This is the complementary operation to
// [Pipe.Basename].
//
// If a line is empty, Dirname will transform it to a single dot. Trailing
// slashes are removed, unless Dirname returns the root folder. Otherwise, the
// behaviour of Dirname is the same as [filepath.Dir] (not by coincidence).
func (p *Pipe) Dirname() *Pipe {
	return p.FilterLine(func(line string) string {
		// filepath.Dir() does not handle trailing slashes correctly
		if len(line) > 1 && strings.HasSuffix(line, "/") {
			line = line[:len(line)-1]
		}
		dirname := filepath.Dir(line)
		// filepath.Dir() does not preserve a leading './'
		if strings.HasPrefix(line, "./") {
			return "./" + dirname
		}
		return dirname
	})
}

// Do performs the HTTP request req using the pipe's configured HTTP client, as
// set by [Pipe.WithHTTPClient], or [http.DefaultClient] otherwise. The
// response body is streamed concurrently to the pipe's output. If the response
// status is anything other than HTTP 200-299, the pipe's error status is set.
func (p *Pipe) Do(req *http.Request) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		// Any HTTP 2xx status code is considered okay
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("unexpected HTTP response status: %s", resp.Status)
		}
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			return err
		}
		return nil
	})
}

// EachLine calls the function process on each line of input, passing it the
// line as a string, and a [*strings.Builder] to write its output to.
//
// Deprecated: use [Pipe.FilterLine] or [Pipe.FilterScan] instead, which run
// concurrently and don't do unnecessary reads on the input.
func (p *Pipe) EachLine(process func(string, *strings.Builder)) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		output := strings.Builder{}
		for scanner.Scan() {
			process(scanner.Text(), &output)
		}
		fmt.Fprint(w, output.String())
		return scanner.Err()
	})
}

// Echo sets the pipe's reader to one that produces the string s, detaching any
// existing reader without draining or closing it.
func (p *Pipe) Echo(s string) *Pipe {
	if p.Error() != nil {
		return p
	}
	return p.WithReader(NewReadAutoCloser(strings.NewReader(s)))
}

// Error returns any error present on the pipe, or nil otherwise.
func (p *Pipe) Error() error {
	if p.mu == nil { // uninitialised pipe
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// Exec runs cmdLine as an external command, sending it the contents of the
// pipe as input, and produces the command's combined output. The effect of
// this is to filter the contents of the pipe through the external command.
//
// # Error handling
//
// If the command had a non-zero exit status, the pipe's error status will also
// be set to the string “exit status X”, where X is the integer exit status.
// Even in the event of a non-zero exit status, the command's output will still
// be available in the pipe. This is often helpful for debugging. However,
// because [Pipe.String] is a no-op if the pipe's error status is set, if you
// want output you will need to reset the error status before calling
// [Pipe.String].
func (p *Pipe) Exec(cmdLine string) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		args, ok := shell.Split(cmdLine) // strings.Fields doesn't handle quotes
		if !ok {
			return fmt.Errorf("unbalanced quotes or backslashes in [%s]", cmdLine)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = r
		cmd.Stdout = w
		cmd.Stderr = w
		err := cmd.Start()
		if err != nil {
			fmt.Fprintln(w, err)
			return err
		}
		return cmd.Wait()
	})
}

// ExecForEach renders cmdLine as a Go template for each line of input, running
// the resulting command, and produces the combined output of all these
// commands in sequence. See [Pipe.Exec] for error handling details.
//
// This is mostly useful for substituting data into commands using Go template
// syntax. For example:
//
//	ListFiles("*").ExecForEach("touch {{.}}").Wait()
func (p *Pipe) ExecForEach(cmdLine string) *Pipe {
	tpl, err := template.New("").Parse(cmdLine)
	if err != nil {
		return p.WithError(err)
	}
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			cmdLine := strings.Builder{}
			err := tpl.Execute(&cmdLine, scanner.Text())
			if err != nil {
				return err
			}
			// strings.Fields doesn't handle quotes
			args, ok := shell.Split(cmdLine.String())
			if !ok {
				return fmt.Errorf("unbalanced quotes or backslashes in [%s]", cmdLine.String())
			}
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = w
			cmd.Stderr = w
			err = cmd.Start()
			if err != nil {
				fmt.Fprintln(w, err)
				continue
			}
			err = cmd.Wait()
			if err != nil {
				fmt.Fprintln(w, err)
				continue
			}
		}
		return scanner.Err()
	})
}

// ExitStatus returns the integer exit status of a previous command (for
// example run by [Pipe.Exec]). This will be zero unless the pipe's error
// status is set and the error matches the pattern “exit status %d”.
func (p *Pipe) ExitStatus() int {
	if p.Error() == nil {
		return 0
	}
	match := exitStatusPattern.FindStringSubmatch(p.Error().Error())
	if len(match) < 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		// This seems unlikely, but...
		return 0
	}
	return status
}

// Filter sends the contents of the pipe to the function filter and produces
// the result. filter takes an [io.Reader] to read its input from and an
// [io.Writer] to write its output to, and returns an error, which will be set
// on the pipe.
//
// filter runs concurrently, so its goroutine will not exit until the pipe has
// been fully read. Use [Pipe.Wait] to wait for all concurrent filters to
// complete.
func (p *Pipe) Filter(filter func(io.Reader, io.Writer) error) *Pipe {
	if p.Error() != nil {
		return p
	}
	pr, pw := io.Pipe()
	origReader := p.Reader
	p = p.WithReader(pr)
	go func() {
		defer pw.Close()
		err := filter(origReader, pw)
		if err != nil {
			p.SetError(err)
		}
	}()
	return p
}

// FilterLine sends the contents of the pipe to the function filter, a line at
// a time, and produces the result. filter takes each line as a string and
// returns a string as its output. See [Pipe.Filter] for concurrency handling.
func (p *Pipe) FilterLine(filter func(string) string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		fmt.Fprintln(w, filter(line))
	})
}

// FilterScan sends the contents of the pipe to the function filter, a line at
// a time, and produces the result. filter takes each line as a string and an
// [io.Writer] to write its output to. See [Pipe.Filter] for concurrency
// handling.
func (p *Pipe) FilterScan(filter func(string, io.Writer)) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			filter(scanner.Text(), w)
		}
		return scanner.Err()
	})
}

// First produces only the first n lines of the pipe's contents, or all the
// lines if there are less than n. If n is zero or negative, there is no output
// at all.
func (p *Pipe) First(n int) *Pipe {
	if p.Error() != nil {
		return p
	}
	if n <= 0 {
		return NewPipe()
	}
	i := 0
	return p.FilterScan(func(line string, w io.Writer) {
		if i >= n {
			return
		}
		fmt.Fprintln(w, line)
		i++
	})
}

// Freq produces only the unique lines from the pipe's contents, each prefixed
// with a frequency count, in descending numerical order (most frequent lines
// first). Lines with equal frequency will be sorted alphabetically.
//
// For example, we could take a common shell pipeline like this:
//
//	sort input.txt |uniq -c |sort -rn
//
// and replace it with:
//
//	File("input.txt").Freq().Stdout()
//
// Or to get only the ten most common lines:
//
//	File("input.txt").Freq().First(10).Stdout()
//
// Like Unix uniq(1), Freq right-justifies its count values in a column for
// readability, padding with spaces if necessary.
func (p *Pipe) Freq() *Pipe {
	freq := map[string]int{}
	type frequency struct {
		line  string
		count int
	}
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			freq[scanner.Text()]++
		}
		freqs := make([]frequency, 0, len(freq))
		var maxCount int
		for line, count := range freq {
			freqs = append(freqs, frequency{line, count})
			if count > maxCount {
				maxCount = count
			}
		}
		sort.Slice(freqs, func(i, j int) bool {
			x, y := freqs[i].count, freqs[j].count
			if x == y {
				return freqs[i].line < freqs[j].line
			}
			return x > y
		})
		fieldWidth := len(strconv.Itoa(maxCount))
		for _, item := range freqs {
			fmt.Fprintf(w, "%*d %s\n", fieldWidth, item.count, item.line)
		}
		return nil
	})
}

// Get makes an HTTP GET request to URL, sending the contents of the pipe as
// the request body, and produces the server's response. See [Pipe.Do] for how
// the HTTP response status is interpreted.
func (p *Pipe) Get(URL string) *Pipe {
	req, err := http.NewRequest(http.MethodGet, URL, p.Reader)
	if err != nil {
		return p.WithError(err)
	}
	return p.Do(req)
}

// Join joins all the lines in the pipe's contents into a single
// space-separated string, which will always end with a newline.
func (p *Pipe) Join() *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		var line string
		first := true
		for scanner.Scan() {
			if !first {
				fmt.Fprint(w, " ")
			}
			line = scanner.Text()
			fmt.Fprint(w, line)
			first = false
		}
		fmt.Fprintln(w)
		return scanner.Err()
	})
}

// JQ executes query on the pipe's contents (presumed to be JSON), producing
// the result. An invalid query will set the appropriate error on the pipe.
//
// The exact dialect of JQ supported is that provided by
// [github.com/itchyny/gojq], whose documentation explains the differences
// between it and standard JQ.
func (p *Pipe) JQ(query string) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		q, err := gojq.Parse(query)
		if err != nil {
			return err
		}
		var input interface{}
		err = json.NewDecoder(r).Decode(&input)
		if err != nil {
			return err
		}
		iter := q.Run(input)
		for {
			v, ok := iter.Next()
			if !ok {
				return nil
			}
			if err, ok := v.(error); ok {
				return err
			}
			result, err := gojq.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Fprintln(w, string(result))
		}
	})
}

// Last produces only the last n lines of the pipe's contents, or all the lines
// if there are less than n. If n is zero or negative, there is no output at
// all.
func (p *Pipe) Last(n int) *Pipe {
	if p.Error() != nil {
		return p
	}
	if n <= 0 {
		return NewPipe()
	}
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		input := ring.New(n)
		for scanner.Scan() {
			input.Value = scanner.Text()
			input = input.Next()
		}
		input.Do(func(p interface{}) {
			if p != nil {
				fmt.Fprintln(w, p)
			}
		})
		return scanner.Err()
	})
}

// Match produces only the input lines that contain the string s.
func (p *Pipe) Match(s string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if strings.Contains(line, s) {
			fmt.Fprintln(w, line)
		}
	})
}

// MatchRegexp produces only the input lines that match the compiled regexp re.
func (p *Pipe) MatchRegexp(re *regexp.Regexp) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if re.MatchString(line) {
			fmt.Fprintln(w, line)
		}
	})
}

// Post makes an HTTP POST request to URL, using the contents of the pipe as
// the request body, and produces the server's response. See [Pipe.Do] for how
// the HTTP response status is interpreted.
func (p *Pipe) Post(URL string) *Pipe {
	req, err := http.NewRequest(http.MethodPost, URL, p.Reader)
	if err != nil {
		return p.WithError(err)
	}
	return p.Do(req)
}

// Read reads up to len(b) bytes from the pipe into b. It returns the number of
// bytes read and any error encountered. At end of file, or on a nil pipe, Read
// returns 0, [io.EOF].
func (p *Pipe) Read(b []byte) (int, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	return p.Reader.Read(b)
}

// Reject produces only lines that do not contain the string s.
func (p *Pipe) Reject(s string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if !strings.Contains(line, s) {
			fmt.Fprintln(w, line)
		}
	})
}

// RejectRegexp produces only lines that don't match the compiled regexp re.
func (p *Pipe) RejectRegexp(re *regexp.Regexp) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if !re.MatchString(line) {
			fmt.Fprintln(w, line)
		}
	})
}

// Replace replaces all occurrences of the string search with the string
// replace.
func (p *Pipe) Replace(search, replace string) *Pipe {
	return p.FilterLine(func(line string) string {
		return strings.ReplaceAll(line, search, replace)
	})
}

// ReplaceRegexp replaces all matches of the compiled regexp re with the string
// re. $x variables in the replace string are interpreted as by
// [regexp.Expand]; for example, $1 represents the text of the first submatch.
func (p *Pipe) ReplaceRegexp(re *regexp.Regexp, replace string) *Pipe {
	return p.FilterLine(func(line string) string {
		return re.ReplaceAllString(line, replace)
	})
}

// SetError sets the error err on the pipe.
func (p *Pipe) SetError(err error) {
	if p.mu == nil { // uninitialised pipe
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}

// SHA256Sum returns the hex-encoded SHA-256 hash of the entire contents of the
// pipe, or an error.
func (p *Pipe) SHA256Sum() (string, error) {
	if p.Error() != nil {
		return "", p.Error()
	}
	hasher := sha256.New()
	_, err := io.Copy(hasher, p)
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), p.Error()
}

// SHA256Sums reads paths from the pipe, one per line, and produces the
// hex-encoded SHA-256 hash of each corresponding file, one per line. Any files
// that cannot be opened or read will be ignored.
func (p *Pipe) SHA256Sums() *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		f, err := os.Open(line)
		if err != nil {
			return // skip unopenable files
		}
		defer f.Close()
		h := sha256.New()
		_, err = io.Copy(h, f)
		if err != nil {
			return // skip unreadable files
		}
		fmt.Fprintln(w, hex.EncodeToString(h.Sum(nil)))
	})
}

// Slice returns the pipe's contents as a slice of strings, one element per
// line, or an error.
//
// An empty pipe will produce an empty slice. A pipe containing a single empty
// line (that is, a single \n character) will produce a slice containing the
// empty string as its single element.
func (p *Pipe) Slice() ([]string, error) {
	result := []string{}
	p.FilterScan(func(line string, w io.Writer) {
		result = append(result, line)
	}).Wait()
	return result, p.Error()
}

// Stdout copies the pipe's contents to its configured standard output (using
// [Pipe.WithStdout]), or to [os.Stdout] otherwise, and returns the number of
// bytes successfully written, together with any error.
func (p *Pipe) Stdout() (int, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	n64, err := io.Copy(p.stdout, p)
	if err != nil {
		return 0, err
	}
	n := int(n64)
	if int64(n) != n64 {
		return 0, fmt.Errorf("length %d overflows int", n64)
	}
	return n, p.Error()
}

// String returns the pipe's contents as a string, together with any error.
func (p *Pipe) String() (string, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
	}
	return string(data), p.Error()
}

// Wait reads the pipe to completion and discards the result. This is mostly
// useful for waiting until concurrent filters have completed (see
// [Pipe.Filter]).
func (p *Pipe) Wait() {
	_, err := io.Copy(io.Discard, p)
	if err != nil {
		p.SetError(err)
	}
}

// WithError sets the error err on the pipe.
func (p *Pipe) WithError(err error) *Pipe {
	p.SetError(err)
	return p
}

// WithHTTPClient sets the HTTP client c for use with subsequent requests via
// [Pipe.Do], [Pipe.Get], or [Pipe.Post]. For example, to make a request using
// a client with a timeout:
//
//	NewPipe().WithHTTPClient(&http.Client{
//	        Timeout: 10 * time.Second,
//	}).Get("https://example.com").Stdout()
func (p *Pipe) WithHTTPClient(c *http.Client) *Pipe {
	p.httpClient = c
	return p
}

// WithReader sets the pipe's input reader to r. Once r has been completely
// read, it will be closed if necessary.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	p.Reader = NewReadAutoCloser(r)
	return p
}

// WithStdout sets the pipe's standard output to the writer w, instead of the
// default [os.Stdout].
func (p *Pipe) WithStdout(w io.Writer) *Pipe {
	p.stdout = w
	return p
}

// WriteFile writes the pipe's contents to the file path, truncating it if it
// exists, and returns the number of bytes successfully written, or an error.
func (p *Pipe) WriteFile(path string) (int64, error) {
	return p.writeOrAppendFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
}

func (p *Pipe) writeOrAppendFile(path string, mode int) (int64, error) {
	if p.Error() != nil {
		return 0, p.Error()
	}
	out, err := os.OpenFile(path, mode, 0666)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	defer out.Close()
	wrote, err := io.Copy(out, p)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	return wrote, nil
}

// ReadAutoCloser wraps an [io.ReadCloser] so that it will be automatically
// closed once it has been fully read.
type ReadAutoCloser struct {
	r io.ReadCloser
}

// NewReadAutoCloser returns a [ReadAutoCloser] wrapping the reader r.
func NewReadAutoCloser(r io.Reader) ReadAutoCloser {
	if _, ok := r.(io.Closer); !ok {
		return ReadAutoCloser{io.NopCloser(r)}
	}
	rc, ok := r.(io.ReadCloser)
	if !ok {
		// This can never happen, but just in case it does...
		panic("internal error: type assertion to io.ReadCloser failed")
	}
	return ReadAutoCloser{rc}
}

// Close closes ra's reader, returning any resulting error.
func (ra ReadAutoCloser) Close() error {
	if ra.r == nil {
		return nil
	}
	return ra.r.(io.Closer).Close()
}

// Read reads up to len(b) bytes from ra's reader into b. It returns the number
// of bytes read and any error encountered. At end of file, Read returns 0,
// [io.EOF]. If end-of-file is reached, the reader will be closed.
func (ra ReadAutoCloser) Read(b []byte) (n int, err error) {
	if ra.r == nil {
		return 0, io.EOF
	}
	n, err = ra.r.Read(b)
	if err == io.EOF {
		ra.Close()
	}
	return n, err
}
