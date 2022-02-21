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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"bitbucket.org/creachadair/shell"
)

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
