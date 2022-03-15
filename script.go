package script

import (
	"bufio"
	"bytes"
	"container/ring"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
		return ReadAutoCloser{ioutil.NopCloser(r)}
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
	mu     *sync.Mutex
	err    error
	stdout io.Writer
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

// Close closes the pipe's associated reader. This is always safe to do, because
// pipes created from a non-closable source will have an `ioutil.NopCloser` to
// call.
func (p *Pipe) Close() error {
	if p == nil {
		return nil
	}
	return p.Reader.Close()
}

// Error returns the last error returned by any pipe operation, or nil otherwise.
func (p *Pipe) Error() error {
	if p == nil {
		return nil
	}
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
	if p == nil {
		return 0, io.EOF
	}
	return p.Reader.Read(b)
}

// SetError sets the pipe's error status to the specified error.
func (p *Pipe) SetError(err error) {
	if p == nil {
		return
	}
	if p.mu == nil { // uninitialised pipe
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err != nil {
		p.Close()
	}
	p.err = err
}

// WithReader takes an io.Reader, and associates the pipe with that reader. If
// necessary, the reader will be automatically closed once it has been
// completely read.
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	if p == nil {
		return nil
	}
	p.Reader = NewReadAutoCloser(r)
	return p
}

// WithStdout takes an io.Writer, and associates the pipe's standard output with
// that reader, instead of the default os.Stdout. This is primarily useful for
// testing.
func (p *Pipe) WithStdout(w io.Writer) *Pipe {
	if p == nil {
		return nil
	}
	p.stdout = w
	return p
}

// WithError sets the pipe's error status to the specified error and returns the
// modified pipe.
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

// Echo returns a pipe containing the supplied string.
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}

// Exec runs an external command and returns a pipe containing its combined
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

// File returns a *Pipe that reads from the specified file. This is useful for
// starting pipelines. If there is an error opening the file, the pipe's error
// status will be set.
func File(name string) *Pipe {
	p := NewPipe()
	f, err := os.Open(name)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}

// FindFiles takes a directory path and returns a pipe listing all the files in
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

// IfExists tests whether the specified file exists, and returns a pipe whose
// error status reflects the result. If the file doesn't exist, the pipe's error
// status will be set, and if the file does exist, the pipe will have no error
// status. This can be used to do some operation only if a given file exists:
//
// IfExists("/foo/bar").Exec("/usr/bin/something")
func IfExists(filename string) *Pipe {
	_, err := os.Stat(filename)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe()
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
	files, err := ioutil.ReadDir(path)
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

// Slice returns a pipe containing each element of the supplied slice of
// strings, one per line.
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// Stdin returns a pipe that reads from the program's standard input.
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
	if p == nil || p.Error() != nil {
		return p
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		out.WriteString(filepath.Base(line))
		out.WriteRune('\n')
	})
}

// Column reads from the pipe, and returns a new pipe containing only the Nth
// column of each line in the input, where '1' means the first column, and
// columns are delimited by whitespace. Specifically, whatever Unicode defines
// as whitespace ('WSpace=yes'). If there is an error reading the pipe, the
// pipe's error status is also set.
//
// Lines containing less than N columns will be ignored.
func (p *Pipe) Column(col int) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		columns := strings.Fields(line)
		if col > 0 && col <= len(columns) {
			out.WriteString(columns[col-1])
			out.WriteRune('\n')
		}
	})
}

// Concat reads a list of filenames from the pipe, one per line, and returns a
// pipe that reads all those files in sequence. If there are any errors (for
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
	if p == nil || p.Error() != nil {
		return p
	}
	var readers []io.Reader
	scanner := bufio.NewScanner(p.Reader)
	for scanner.Scan() {
		input, err := os.Open(scanner.Text())
		if err != nil {
			continue // skip errors
		}
		readers = append(readers, NewReadAutoCloser(input))
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	return p.WithReader(io.MultiReader(readers...))
}

// Dirname reads a list of pathnames from the pipe, one per line, and returns a
// pipe that contains only the parent directories of each pathname. For example,
// `/usr/local/bin/foo` would become just `/usr/local/bin`. This is the
// complementary operation to Basename.
//
// If a line is empty, Dirname will produce a '.'. Trailing slashes are removed,
// unless Dirname returns the root folder. The behaviour of Dirname is the same
// as filepath.Dir (not by coincidence).
func (p *Pipe) Dirname() *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		// filepath.Dir() does not handle trailing slashes correctly
		if len(line) > 1 && strings.HasSuffix(line, "/") {
			line = line[0 : len(line)-1]
		}
		dirname := filepath.Dir(line)
		// filepath.Dir() does not preserve a leading './'
		if len(dirname) > 1 && strings.HasPrefix(line, "./") {
			dirname = "./" + dirname
		}
		out.WriteString(dirname)
		out.WriteRune('\n')
	})
}

// EachLine calls the specified function for each line of input, passing it the
// line as a string, and a *strings.Builder to write its output to. The return
// value from EachLine is a pipe containing the contents of the strings.Builder.
func (p *Pipe) EachLine(process func(string, *strings.Builder)) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	scanner := bufio.NewScanner(p.Reader)
	output := strings.Builder{}
	for scanner.Scan() {
		process(scanner.Text(), &output)
		if p.Error() != nil {
			return p
		}
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	return Echo(output.String())
}

// Echo returns a pipe containing the supplied string.
func (p *Pipe) Echo(s string) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	return p.WithReader(strings.NewReader(s))
}

// Exec runs an external command, sending it the contents of the pipe as input,
// and returns a pipe containing the command's combined output (`stdout` and
// `stderr`). The effect of this is to use the external command as a filter on
// the pipe.
//
// If the command had a non-zero exit status, the pipe's error status will also
// be set to the string "exit status X", where X is the integer exit status.
func (p *Pipe) Exec(cmdLine string) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	q := NewPipe()
	args, ok := shell.Split(cmdLine) // strings.Fields doesn't handle quotes
	if !ok {
		return p.WithError(fmt.Errorf("unbalanced quotes or backslashes in [%s]", cmdLine))
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = p.Reader
	output, err := cmd.CombinedOutput()
	if err != nil {
		q.SetError(err)
	}
	return q.WithReader(bytes.NewReader(output))
}

// ExecForEach runs the supplied command once for each line of input, and
// returns a pipe containing the output. The command string is interpreted as a
// Go template, so `{{.}}` will be replaced with the input value, for example.
// If any command resulted in a non-zero exit status, the pipe's error status
// will also be set to the string "exit status X", where X is the integer exit
// status.
func (p *Pipe) ExecForEach(cmdTpl string) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	tpl, err := template.New("").Parse(cmdTpl)
	if err != nil {
		return p.WithError(err)
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		cmdLine := strings.Builder{}
		err := tpl.Execute(&cmdLine, line)
		if err != nil {
			p.SetError(err)
			return
		}
		cmdOutput, err := Exec(cmdLine.String()).String()
		if err != nil {
			p.SetError(err)
			return
		}
		out.WriteString(cmdOutput)
	})
}

type FilterFunc func(io.Reader, io.Writer) error

func (p *Pipe) Filter(filter FilterFunc) *Pipe {
	pr, pw := io.Pipe()
	q := NewPipe().WithReader(pr)
	go func() {
		err := filter(p.Reader, pw)
		q.SetError(err)
		pw.Close()
	}()
	return q
}

// First reads from the pipe, and returns a new pipe containing only the first N
// lines. If there is an error reading the pipe, the pipe's error status is also
// set.
func (p *Pipe) First(lines int) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	defer p.Close()
	if lines <= 0 {
		return NewPipe()
	}
	scanner := bufio.NewScanner(p.Reader)
	output := strings.Builder{}
	for i := 0; i < lines; i++ {
		if !scanner.Scan() {
			break
		}
		output.WriteString(scanner.Text())
		output.WriteRune('\n')
	}
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	return Echo(output.String())
}

// Freq reads from the pipe, and returns a new pipe containing only unique lines
// from the input, prefixed with a frequency count, in descending numerical
// order (most frequent lines first). Lines with equal frequency will be sorted
// alphabetically. If there is an error reading the pipe, the pipe's error
// status is also set.
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
	if p == nil || p.Error() != nil {
		return p
	}
	freq := map[string]int{}
	p.EachLine(func(line string, out *strings.Builder) {
		freq[line]++
	})
	type frequency struct {
		line  string
		count int
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
		if freqs[i].count == freqs[j].count {
			return freqs[i].line < freqs[j].line
		}
		return freqs[i].count > freqs[j].count
	})
	fieldWidth := len(strconv.Itoa(maxCount))
	var output strings.Builder
	for _, item := range freqs {
		output.WriteString(fmt.Sprintf("%*d %s", fieldWidth, item.count, item.line))
		output.WriteRune('\n')
	}
	return Echo(output.String())
}

// Join reads the contents of the pipe, line by line, and joins them into a
// single space-separated string. It returns a pipe containing this string. Any
// terminating newline is preserved.
func (p *Pipe) Join() *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	result, err := p.String()
	if err != nil {
		return p
	}
	var terminator string
	if strings.HasSuffix(result, "\n") {
		terminator = "\n"
		result = result[:len(result)-1]
	}
	output := strings.ReplaceAll(result, "\n", " ")
	return Echo(output + terminator)
}

// Last reads from the pipe, and returns a new pipe containing only the last N
// lines. If there is an error reading the pipe, the pipe's error status is also
// set.
func (p *Pipe) Last(lines int) *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}
	defer p.Close()
	if lines <= 0 {
		return NewPipe()
	}
	scanner := bufio.NewScanner(p.Reader)
	input := ring.New(lines)
	for scanner.Scan() {
		input.Value = scanner.Text()
		input = input.Next()
	}
	output := strings.Builder{}
	input.Do(func(p interface{}) {
		value, ok := p.(string)
		if ok {
			output.WriteString(value)
			output.WriteRune('\n')
		}
	})
	err := scanner.Err()
	if err != nil {
		p.SetError(err)
	}
	return Echo(output.String())
}

// Match reads from the pipe, and returns a new pipe containing only lines that
// contain the specified string. If there is an error reading the pipe, the
// pipe's error status is also set.
func (p *Pipe) Match(s string) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if strings.Contains(line, s) {
			out.WriteString(line)
			out.WriteRune('\n')
		}
	})
}

// MatchRegexp reads from the pipe, and returns a new pipe containing only lines
// that match the specified compiled regular expression. If there is an error
// reading the pipe, the pipe's error status is also set.
func (p *Pipe) MatchRegexp(re *regexp.Regexp) *Pipe {
	if re == nil { // to prevent SIGSEGV
		return p.WithError(errors.New("nil regular expression"))
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		if re.MatchString(line) {
			out.WriteString(line)
			out.WriteRune('\n')
		}
	})
}

// Reject reads from the pipe, and returns a new pipe containing only lines
// that do not contain the specified string. If there is an error reading the
// pipe, the pipe's error status is also set.
func (p *Pipe) Reject(s string) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		if !strings.Contains(line, s) {
			out.WriteString(line)
			out.WriteRune('\n')
		}
	})
}

// RejectRegexp reads from the pipe, and returns a new pipe containing only
// lines that don't match the specified compiled regular expression. If there
// is an error reading the pipe, the pipe's error status is also set.
func (p *Pipe) RejectRegexp(re *regexp.Regexp) *Pipe {
	if re == nil { // to prevent SIGSEGV
		return p.WithError(errors.New("nil regular expression"))
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		if !re.MatchString(line) {
			out.WriteString(line)
			out.WriteRune('\n')
		}
	})
}

// Replace filters its input by replacing all occurrences of the string `search`
// with the string `replace`. If there is an error reading the pipe, the pipe's
// error status is also set.
func (p *Pipe) Replace(search, replace string) *Pipe {
	return p.EachLine(func(line string, out *strings.Builder) {
		out.WriteString(strings.ReplaceAll(line, search, replace))
		out.WriteRune('\n')
	})
}

// ReplaceRegexp filters its input by replacing all matches of the compiled
// regular expression `re` with the replacement string `replace`. Inside
// `replace`, $ signs are interpreted as in regexp.Expand, so for instance "$1"
// represents the text of the first submatch. If there is an error reading the
// pipe, the pipe's error status is also set.
func (p *Pipe) ReplaceRegexp(re *regexp.Regexp, replace string) *Pipe {
	if re == nil { // to prevent SIGSEGV
		return p.WithError(errors.New("nil regular expression"))
	}
	return p.EachLine(func(line string, out *strings.Builder) {
		out.WriteString(re.ReplaceAllString(line, replace))
		out.WriteRune('\n')
	})
}

// SHA256Sums reads a list of file paths from the pipe, one per line, and
// returns a pipe that contains the SHA-256 checksum of each pathname, in hex.
// If there are any errors (for example, non-existent files), the pipe's error
// status will be set to the first error encountered, but execution will
// continue.
func (p *Pipe) SHA256Sums() *Pipe {
	if p == nil || p.Error() != nil {
		return p
	}

	return p.EachLine(func(line string, out *strings.Builder) {
		f, err := os.Open(line)
		if err != nil {
			p.SetError(err)
			return
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			p.SetError(err)
			return
		}

		out.WriteString(hex.EncodeToString(h.Sum(nil)[:]))
		out.WriteRune('\n')
	})
}

// AppendFile appends the contents of the Pipe to the specified file, and closes
// the pipe after reading. If the file does not exist, it is created.
//
// AppendFile returns the number of bytes successfully written, or an error. If
// there is an error reading or writing, the pipe's error status is also set.
func (p *Pipe) AppendFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
}

// Bytes returns the contents of the Pipe as a slice of byte, or an error. If
// there is an error reading, the pipe's error status is also set.
func (p *Pipe) Bytes() ([]byte, error) {
	if p == nil || p.Error() != nil {
		return []byte{}, p.Error()
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return []byte{}, err
	}
	return res, nil
}

// CountLines counts lines from the pipe's reader, and returns the integer
// result, or an error. If there is an error reading the pipe, the pipe's error
// status is also set.
func (p *Pipe) CountLines() (int, error) {
	if p == nil || p.Error() != nil {
		return 0, p.Error()
	}
	var lines int
	p.EachLine(func(line string, out *strings.Builder) {
		lines++
	})
	return lines, p.Error()
}

// SHA256Sum calculates the SHA-256 of the file from the pipe's reader, and
// returns the hex-encoded string result, or an error. If there is an error
// reading the pipe, the pipe's error status is also set.
func (p *Pipe) SHA256Sum() (string, error) {
	if p == nil || p.Error() != nil {
		return "", p.Error()
	}

	h := sha256.New()
	if _, err := io.Copy(h, p.Reader); err != nil {
		p.SetError(err)
		return "", p.Error()
	}

	encodedCheckSum := hex.EncodeToString(h.Sum(nil)[:])
	return encodedCheckSum, nil
}

// Slice returns the contents of the pipe as a slice of strings, one element per
// line, or an error. If there is an error reading the pipe, the pipe's error
// status is also set.
//
// An empty pipe will produce an empty slice. A pipe containing a single empty
// line (that is, a single `\n` character) will produce a slice of one element
// that is the empty string.
func (p *Pipe) Slice() ([]string, error) {
	if p == nil || p.Error() != nil {
		return nil, p.Error()
	}
	result := []string{}
	p.EachLine(func(line string, out *strings.Builder) {
		result = append(result, line)
	})
	return result, p.Error()
}

// Stdout writes the contents of the pipe to its configured standard output. It
// returns the number of bytes successfully written, plus a non-nil error if the
// write failed or if there was an error reading from the pipe. If the pipe has
// error status, Stdout returns zero plus the existing error.
func (p *Pipe) Stdout() (int, error) {
	if p == nil || p.Error() != nil || p.stdout == nil {
		return 0, p.Error()
	}
	n64, err := io.Copy(p.stdout, p.Reader)
	if err != nil {
		return 0, err
	}
	n := int(n64)
	if int64(n) != n64 {
		return 0, fmt.Errorf("length %d overflows int", n64)
	}
	return n, nil
}

// String returns the contents of the Pipe as a string, or an error, and closes
// the pipe after reading. If there is an error reading, the pipe's error status
// is also set.
//
// Note that String consumes the complete output of the pipe, which closes the
// input reader automatically. Therefore, calling String (or any other sink
// method) again on the same pipe will return an error.
func (p *Pipe) String() (string, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return string(data), nil
}

// WriteFile writes the contents of the Pipe to the specified file, and closes
// the pipe after reading. If the file already exists, it is truncated and the
// new data will replace the old. It returns the number of bytes successfully
// written, or an error. If there is an error reading or writing, the pipe's
// error status is also set.
func (p *Pipe) WriteFile(fileName string) (int64, error) {
	return p.writeOrAppendFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
}

func (p *Pipe) writeOrAppendFile(fileName string, mode int) (int64, error) {
	if p == nil || p.Error() != nil {
		return 0, p.Error()
	}
	out, err := os.OpenFile(fileName, mode, 0666)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	defer out.Close()
	wrote, err := io.Copy(out, p.Reader)
	if err != nil {
		p.SetError(err)
		return 0, err
	}
	return wrote, nil
}
