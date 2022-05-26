package script

import (
	"bufio"
	"container/ring"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// ReadAutoCloser represents a pipe source that will be automatically closed
// once it has been fully read.
type ReadAutoCloser struct {
	r io.ReadCloser
}

// Read reads up to len(buf) bytes from the data source into buf. It returns the
// number of bytes read and any error encountered. At end of file, Read returns
// 0, io.EOF. In the EOF case, the data source will be closed.
func (a ReadAutoCloser) Read(buf []byte) (n int, err error) {
	if a.r == nil {
		return 0, io.EOF
	}
	n, err = a.r.Read(buf)
	if err == io.EOF {
		a.Close()
	}
	return n, err
}

// Close closes the data source associated with a, and returns the result of
// that close operation.
func (a ReadAutoCloser) Close() error {
	if a.r == nil {
		return nil
	}
	return a.r.(io.Closer).Close()
}

// NewReadAutoCloser returns an ReadAutoCloser wrapping the supplied Reader. If
// the Reader is not a Closer, it will be wrapped in a NopCloser to make it
// closable.
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

// Pipe represents a pipe object with an associated ReadAutoCloser.
type Pipe struct {
	Reader ReadAutoCloser
	stdout io.Writer

	// because pipe stages are concurrent, protect 'err'
	mu  *sync.Mutex
	err error
}

// NewPipe returns a pointer to a new empty pipe.
func NewPipe() *Pipe {
	return &Pipe{
		Reader: ReadAutoCloser{},
		mu:     &sync.Mutex{},
		err:    nil,
		stdout: os.Stdout,
	}
}

// Close closes the pipe's associated reader. This is a no-op if the reader is
// not also a Closer.
func (p *Pipe) Close() error {
	return p.Reader.Close()
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

var exitStatusPattern = regexp.MustCompile(`exit status (\d+)$`)

// ExitStatus returns the integer exit status of a previous command, if the
// pipe's error status is set, and if the error matches the pattern "exit status
// %d". Otherwise, it returns zero.
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

// Read reads up to len(b) bytes from the data source into b. It returns the
// number of bytes read and any error encountered. At end of file, or on a nil
// pipe, Read returns 0, io.EOF.
//
// Unlike most sinks, Read does not necessarily read the whole contents of the
// pipe. It will read as many bytes as it takes to fill the slice.
func (p *Pipe) Read(b []byte) (int, error) {
	return p.Reader.Read(b)
}

// SetError sets the specified error on the pipe.
func (p *Pipe) SetError(err error) {
	if p.mu == nil { // uninitialised pipe
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}

// WithReader sets the pipe's input to the specified reader. If necessary, the
// reader will be automatically closed once it has been completely read.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	p.Reader = NewReadAutoCloser(r)
	return p
}

// WithStdout sets the pipe's standard output to the specified reader, instead
// of the default os.Stdout.
func (p *Pipe) WithStdout(w io.Writer) *Pipe {
	p.stdout = w
	return p
}

// WithError sets the specified error on the pipe and returns the modified pipe.
func (p *Pipe) WithError(err error) *Pipe {
	p.SetError(err)
	return p
}

// Args creates a pipe containing the program's command-line arguments, one per
// line.
func Args() *Pipe {
	var s strings.Builder
	for _, a := range os.Args[1:] {
		s.WriteString(a + "\n")
	}
	return Echo(s.String())
}

// Echo creates a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec runs an external command and creates a pipe containing its combined
// output (stdout and stderr).
//
// If the command had a non-zero exit status, the pipe's error status will also
// be set to the string "exit status X", where X is the integer exit status.
//
// For convenience, you can get this value directly as an integer by calling
// ExitStatus on the pipe.
//
// Even in the event of a non-zero exit status, the command's output will still
// be available in the pipe. This is often helpful for debugging. However,
// because String is a no-op if the pipe's error status is set, if you want
// output you will need to reset the error status before calling String.
//
// Note that Exec can also be used as a filter, in which case the given command
// will read from the pipe as its standard input.
func Exec(s string) *Pipe {
	return NewPipe().Exec(s)
}

// File creates a pipe that reads from the file at the specified path.
func File(name string) *Pipe {
	p := NewPipe()
	f, err := os.Open(name)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}

// FindFiles takes a directory path and creates a pipe listing all the files in
// the directory and its subdirectories recursively, one per line, like Unix
// `find -type f`. If the path doesn't exist or can't be read, the pipe's error
// status will be set.
//
// Each line of the output consists of a slash-separated pathname, starting with
// the initial directory. For example, if the starting directory is "test", and
// it contains 1.txt and 2.txt:
//
// test/1.txt
// test/2.txt
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

// IfExists tests whether the specified file exists, and creates a pipe whose
// error status reflects the result. If the file doesn't exist, the pipe's error
// status will be set, and if the file does exist, the pipe will have no error
// status. This can be used to do some operation only if a given file exists:
//
// IfExists("/foo/bar").Exec("/usr/bin/something")
func IfExists(filename string) *Pipe {
	p := NewPipe()
	_, err := os.Stat(filename)
	if err != nil {
		return p.WithError(err)
	}
	return p
}

// ListFiles creates a pipe containing the files and directories matching the
// supplied path, one per line. The path can be the name of a directory
// (`/path/to/dir`), the name of a file (`/path/to/file`), or a glob (wildcard
// expression) conforming to the syntax accepted by filepath.Match (for example
// `/path/to/*`).
//
// ListFiles does not recurse into subdirectories (use FindFiles for this).
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

// Slice creates a pipe containing each element of the supplied slice of
// strings, one per line.
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// Stdin creates a pipe that reads from os.Stdin.
func Stdin() *Pipe {
	return NewPipe().WithReader(os.Stdin)
}

// Basename reads a list of filepaths from the pipe, one per line, and removes
// any leading directory components from each line. So, for example,
// `/usr/local/bin/foo` would become just `foo`. This is the complementary
// operation to Dirname.
//
// If a line is empty, Basename will produce '.'. Trailing slashes are removed.
// The behaviour of Basename is the same as filepath.Base (not by coincidence).
func (p *Pipe) Basename() *Pipe {
	return p.FilterLine(filepath.Base)
}

// Column produces only the Nth column of each line of input, where '1' is the
// first column, and columns are delimited by whitespace. Specifically, whatever
// Unicode defines as whitespace ('WSpace=yes').
//
// Lines containing less than N columns will be dropped altogether.
func (p *Pipe) Column(col int) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		columns := strings.Fields(line)
		if col > 0 && col <= len(columns) {
			fmt.Fprintln(w, columns[col-1])
		}
	})
}

// Concat reads a list of file paths from the pipe, one per line, and produces
// the contents of all those files in sequence. If there are any errors (for
// example, non-existent files), these will be ignored, execution will continue,
// and the pipe's error status will not be set.
//
// This makes it convenient to write programs that take a list of input files on
// the command line. For example:
//
// script.Args().Concat().Stdout()
//
// The list of files could also come from a file:
//
// script.File("filelist.txt").Concat()
//
// ...or from the output of a command:
//
// script.Exec("ls /var/app/config/").Concat().Stdout()
//
// Each input file will be closed once it has been fully read. If any of the
// files can't be opened or read, `Concat` will simply skip these and carry on,
// without setting the pipe's error status. This mimics the behaviour of Unix
// `cat`.
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

// Dirname reads a list of pathnames from the pipe, one per line, and produces
// only the parent directories of each pathname. For example,
// `/usr/local/bin/foo` would become just `/usr/local/bin`. This is the
// complementary operation to Basename.
//
// If a line is empty, Dirname will produce a '.'. Trailing slashes are removed,
// unless Dirname returns the root folder. Otherwise, the behaviour of Dirname
// is the same as filepath.Dir (not by coincidence).
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

// EachLine calls the specified function for each line of input, passing it the
// line as a string, and a *strings.Builder to write its output to.
//
// Deprecated: use FilterLine or FilterScan instead, which run concurrently and
// don't do unnecessary reads on the input.
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

// Echo produces the supplied string.
func (p *Pipe) Echo(s string) *Pipe {
	if p.Error() != nil {
		return p
	}
	return p.WithReader(NewReadAutoCloser(strings.NewReader(s)))
}

// Exec runs an external command, sending it the contents of the pipe as input,
// and produces the command's combined output (`stdout` and `stderr`). The
// effect of this is to filter the contents of the pipe through the external
// command.
//
// If the command had a non-zero exit status, the pipe's error status will also
// be set to the string "exit status X", where X is the integer exit status.
func (p *Pipe) Exec(command string) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		args, ok := shell.Split(command) // strings.Fields doesn't handle quotes
		if !ok {
			return fmt.Errorf("unbalanced quotes or backslashes in [%s]", command)
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

// ExecForEach runs the supplied command once for each line of input, and
// produces its combined output. The command string is interpreted as a Go
// template, so `{{.}}` will be replaced with the input value, for example.
//
// If any command resulted in a non-zero exit status, the pipe's error status
// will also be set to the string "exit status X", where X is the integer exit
// status.
func (p *Pipe) ExecForEach(command string) *Pipe {
	if p.Error() != nil {
		return p
	}
	tpl, err := template.New("").Parse(command)
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
			args, ok := shell.Split(cmdLine.String()) // strings.Fields doesn't handle quotes
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
			cmd.Wait()
		}
		return scanner.Err()
	})
}

// Filter filters the contents of the pipe through the supplied function, which
// takes an io.Reader (the filter input) and an io.Writer (the filter output),
// and returns an error, which will be set on the pipe.
//
// The filter function runs concurrently, so its goroutine will not complete
// until the pipe has been fully read. If you just need to make sure all
// concurrent filters have completed, call Wait on the end of the pipe.
func (p *Pipe) Filter(filter func(io.Reader, io.Writer) error) *Pipe {
	pr, pw := io.Pipe()
	q := NewPipe().WithReader(pr)
	go func() {
		defer pw.Close()
		err := filter(p, pw)
		q.SetError(err)
	}()
	return q
}

// FilterLine filters the contents of the pipe, a line at a time, through the
// supplied function, which takes the line as a string and returns a string (the
// filter output). The filter function runs concurrently.
func (p *Pipe) FilterLine(filter func(string) string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		fmt.Fprintln(w, filter(line))
	})
}

// FilterScan filters the contents of the pipe, a line at a time, through the
// supplied function, which takes the line as a string and an io.Writer (the
// filtero output). The filter function runs concurrently.
func (p *Pipe) FilterScan(filter func(string, io.Writer)) *Pipe {
	return p.Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			filter(scanner.Text(), w)
		}
		return scanner.Err()
	})
}

// First produces only the first N lines of input, or the whole input if there
// are less than N lines. If N is zero or negative, there is no output at all.
func (p *Pipe) First(n int) *Pipe {
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

// Freq produces only unique lines from the input, prefixed with a frequency
// count, in descending numerical order (most frequent lines first). Lines with
// equal frequency will be sorted alphabetically.
//
// This is a common pattern in shell scripts to find the most
// frequently-occurring lines in a file:
//
// sort testdata/freq.input.txt |uniq -c |sort -rn
//
// Freq's behaviour is like the combination of Unix `sort`, `uniq -c`, and `sort
// -rn` used here. You can use Freq in combination with First to get, for
// example, the ten most common lines in a file:
//
// script.Stdin().Freq().First(10).Stdout()
//
// Like `uniq -c`, Freq left-pads its count values if necessary to make them
// easier to read:
//
// 10 apple
//  4 banana
//  2 orange
//  1 kumquat
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

// Join produces its input as a single space-separated string, which will always
// end with a newline.
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

// JQ takes a query in the 'jq' language and applies it to the input (presumed
// to be JSON), producing the result. An invalid query will set the appropriate
// error on the pipe.
//
// The exact dialect of JQ supported is that provided by
// github.com/itchyny/gojq, whose documentation explains the differences between
// it and 'standard' JQ.
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

// Last produces only the last N lines of input, or the whole input if there are
// less than N lines. If N is zero or negative, there is no output at all.
func (p *Pipe) Last(n int) *Pipe {
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

// Match produces only lines that contain the specified string.
func (p *Pipe) Match(s string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if strings.Contains(line, s) {
			fmt.Fprintln(w, line)
		}
	})
}

// MatchRegexp produces only lines that match the specified compiled regular
// expression.
func (p *Pipe) MatchRegexp(re *regexp.Regexp) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if re.MatchString(line) {
			fmt.Fprintln(w, line)
		}
	})
}

// Reject produces only lines that do not contain the specified string.
func (p *Pipe) Reject(s string) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if !strings.Contains(line, s) {
			fmt.Fprintln(w, line)
		}
	})
}

// RejectRegexp produces only lines that don't match the specified compiled
// regular expression.
func (p *Pipe) RejectRegexp(re *regexp.Regexp) *Pipe {
	return p.FilterScan(func(line string, w io.Writer) {
		if !re.MatchString(line) {
			fmt.Fprintln(w, line)
		}
	})
}

// Replace replaces all occurrences of the 'search' string with the 'replace'
// string.
func (p *Pipe) Replace(search, replace string) *Pipe {
	return p.FilterLine(func(line string) string {
		return strings.ReplaceAll(line, search, replace)
	})
}

// ReplaceRegexp replaces all matches of the specified compiled regular
// expression with the 'replace' string. '$' characters in the replace string
// are interpreted as in regexp.Expand; for example, "$1" represents the text of
// the first submatch.
func (p *Pipe) ReplaceRegexp(re *regexp.Regexp, replace string) *Pipe {
	return p.FilterLine(func(line string) string {
		return re.ReplaceAllString(line, replace)
	})
}

// SHA256Sums reads a list of file paths from the pipe, one per line, and
// produces the hex-encoded SHA-256 hash of each file. Any files that cannot be
// opened or read will be ignored.
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

// AppendFile appends the contents of the pipe to the specified file, and
// returns the number of bytes successfully written, or an error. If the file
// does not exist, it is created.
func (p *Pipe) AppendFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

// Bytes returns the contents of the pipe as a []]byte, or an error.
func (p *Pipe) Bytes() ([]byte, error) {
	res, err := io.ReadAll(p)
	if err != nil {
		p.SetError(err)
	}
	return res, err
}

// CountLines returns the number of lines of input, or an error.
func (p *Pipe) CountLines() (int, error) {
	lines := 0
	p.FilterScan(func(line string, w io.Writer) {
		lines++
	}).Wait()
	return lines, p.Error()
}

// SHA256Sum returns the hex-encoded SHA-256 hash of its input, or an error.
func (p *Pipe) SHA256Sum() (string, error) {
	hasher := sha256.New()
	_, err := io.Copy(hasher, p)
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), p.Error()
}

// Slice returns the input as a slice of strings, one element per line, or an
// error.
//
// An empty pipe will produce an empty slice. A pipe containing a single empty
// line (that is, a single `\n` character) will produce a slice containing the
// empty string.
func (p *Pipe) Slice() ([]string, error) {
	result := []string{}
	p.FilterScan(func(line string, w io.Writer) {
		result = append(result, line)
	}).Wait()
	return result, p.Error()
}

// Stdout writes the input to the pipe's configured standard output, and returns
// the number of bytes successfully written, or an error.
func (p *Pipe) Stdout() (int, error) {
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

// String returns the input as a string, or an error.
func (p *Pipe) String() (string, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
	}
	return string(data), p.Error()
}

// Wait reads the input to completion and discards it. This is mostly useful for
// waiting until all concurrent filter stages have finished.
func (p *Pipe) Wait() {
	_, err := io.Copy(io.Discard, p)
	if err != nil {
		p.SetError(err)
	}
}

// WriteFile writes the input to the specified file, and returns the number of
// bytes successfully written, or an error. If the file already exists, it is
// truncated and the new data will replace the old.
func (p *Pipe) WriteFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
}

func (p *Pipe) writeOrAppendFile(fileName string, mode int) (int64, error) {
	out, err := os.OpenFile(fileName, mode, 0666)
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
