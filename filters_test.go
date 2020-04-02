package script

import (
	"bytes"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
)

func TestBasename(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		want         string
	}{
		{"\n", ".\n"},
		{"/", "/\n"},
		{"/root", "root\n"},
		{"/tmp/example.php", "example.php\n"},
		{"./src/filters", "filters\n"},
		{"/var/tmp/example.php", "example.php\n"},
		{"/tmp/script-21345.txt\n/tmp/script-5371253.txt", "script-21345.txt\nscript-5371253.txt\n"},
		{"C:/Program Files", "Program Files\n"},
		{"C:/Program Files/", "Program Files\n"},
	}
	for _, tc := range testCases {
		got, err := Echo(tc.testFileName).Basename().String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q: want %q, got %q", tc.testFileName, tc.want, got)
		}
	}
}

func TestColumn(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/column.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := File("testdata/column.input.txt").Column(3).Bytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestConcat(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/concat.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Echo("testdata/test.txt\ntestdata/doesntexist.txt\ntestdata/hello.txt").Concat().Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestDirname(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		want         string
	}{
		{"/", "/\n"},
		{"\n", ".\n"},
		{"/root", "/\n"},
		{"/tmp/example.php", "/tmp\n"},
		{"/var/tmp/example.php", "/var/tmp\n"},
		{"/var/tmp", "/var\n"},
		{"/var/tmp/", "/var\n"},
		{"./src/filters", "./src\n"},
		{"./src/filters/", "./src\n"},
		{"/tmp/script-21345.txt\n/tmp/script-5371253.txt", "/tmp\n/tmp\n"},
		{"C:/Program Files/PHP", "C:/Program Files\n"},
		{"C:/Program Files/PHP/", "C:/Program Files\n"},
		{"C:/Program Files", "C:\n"},

		// NOTE:
		// there are no tests for Windows-style directory separators,
		// because these are not supported by filepath at this time
		//
		// the following test data currently does not work with the
		// Golang filepath library:
		//
		// {"C:\\Program Files\\PHP", "C:\\Program Files\n"},
		// {"C:\\Program Files\\PHP\\", "C:\\Program Files\n"},
	}
	for _, tc := range testCases {
		got, err := Echo(tc.testFileName).Dirname().String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q: want %q, got %q", tc.testFileName, tc.want, got)
		}
	}
}

func TestEachLine(t *testing.T) {
	t.Parallel()
	p := Echo("Hello\nGoodbye")
	q := p.EachLine(func(line string, out *strings.Builder) {
		out.WriteString(line + " world\n")
	})
	want := "Hello world\nGoodbye world\n"
	got, err := q.String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExecFilter(t *testing.T) {
	t.Parallel()
	want := "hello world"
	p := File("testdata/hello.txt")
	q := p.Exec("cat")
	got, err := q.String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
	// This should fail because p is now closed.
	_, err = p.String()
	if err == nil {
		t.Error("input not closed after reading")
	}
	p = Echo("hello world")
	p.SetError(errors.New("oh no"))
	// This should be a no-op because the pipe has error status.
	out, _ := p.Exec("cat").String()
	if out != "" {
		t.Error("want exec on erroneous pipe to be a no-op, but it wasn't")
	}
}

func TestExecForEach(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		Input       []string
		Command     string
		ErrExpected bool
		WantOutput  string
	}{
		{
			Command:     "bash -c 'echo {{.}}'",
			Input:       []string{"a", "b", "c"},
			ErrExpected: false,
			WantOutput:  "a\nb\nc\n",
		},
		{
			Command:     "bash -c 'echo {{if not .}}DEFAULT{{else}}{{.}}{{end}}'",
			Input:       []string{"a", "", "c"},
			ErrExpected: false,
			WantOutput:  "a\nDEFAULT\nc\n",
		},
		{
			Command:     "bash -c 'echo {{bogus template syntax}}'",
			Input:       []string{"a", "", "c"},
			ErrExpected: true,
			WantOutput:  "",
		},
		{
			Command:     "bogus {{.}}",
			Input:       []string{"a", "b", "c"},
			ErrExpected: true,
			WantOutput:  "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Command, func(t *testing.T) {
			p := Slice(tc.Input).ExecForEach(tc.Command)
			if tc.ErrExpected != (p.Error() != nil) {
				t.Fatalf("unexpected error value: %v", p.Error())
			}
			p.SetError(nil) // else p.String() would be a no-op
			output, err := p.String()
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !strings.Contains(output, tc.WantOutput) {
				t.Fatalf("want output %q to contain %q", output, tc.WantOutput)
			}
		})
	}
}

func TestFirst(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/first10.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	input := File("testdata/first.input.txt")
	got, err := input.First(10).Bytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
	_, err = ioutil.ReadAll(input.Reader)
	if err == nil {
		t.Error("input not closed after reading")
	}
	input = File("testdata/first.input.txt")
	gotZero, err := input.First(0).CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if gotZero != 0 {
		t.Errorf("want 0 lines, got %d lines", gotZero)
	}
	_, err = ioutil.ReadAll(input.Reader)
	if err == nil {
		t.Error("input not closed after reading")
	}
	want, err = File("testdata/first.input.txt").Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err = File("testdata/first.input.txt").First(100).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestFreq(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/freq.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := File("testdata/freq.input.txt").Freq().Bytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()
	input := "hello\nfrom\nthe\njoin\ntest\n"
	want := "hello from the join test\n"
	got, err := Echo(input).Join().String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
	input = "hello\nworld"
	want = "hello world"
	got, err = Echo(input).Join().String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestLast(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/last10.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	input := File("testdata/first.input.txt")
	got, err := input.Last(10).Bytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
	_, err = ioutil.ReadAll(input.Reader)
	if err == nil {
		t.Error("input not closed after reading")
	}
	input = File("testdata/first.input.txt")
	gotZero, err := input.Last(0).CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if gotZero != 0 {
		t.Errorf("want 0 lines, got %d lines", gotZero)
	}
	_, err = ioutil.ReadAll(input.Reader)
	if err == nil {
		t.Error("input not closed after reading")
	}
	want, err = File("testdata/first.input.txt").Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got, err = File("testdata/first.input.txt").Last(100).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		match        string
		want         int
	}{
		{"testdata/test.txt", "line", 2},
		{"testdata/test.txt", "another", 1},
		{"testdata/test.txt", "rhymenocerous", 0},
		{"testdata/empty.txt", "line", 0},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).Match(tc.match).CountLines()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q in %q: want %d, got %d", tc.match, tc.testFileName, tc.want, got)
		}
	}
}

func TestMatchRegexp(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		match        string
		want         int
	}{
		{"testdata/test.txt", `Hello|file`, 3},
		{"testdata/test.txt", `an.ther`, 1},
		{"testdata/test.txt", `r[a-z]+s`, 0},
		{"testdata/empty.txt", `bogus$`, 0},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).MatchRegexp(regexp.MustCompile(tc.match)).CountLines()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q in %q: want %d, got %d", tc.match, tc.testFileName, tc.want, got)
		}
	}
}

func TestReplace(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		search       string
		replace      string
		want         string
	}{
		{"testdata/hello.txt", "hello", "bye", "bye world\n"},
		{"testdata/hello.txt", "Does not exist in input", "Will not appear in output", "hello world\n"},
		{"testdata/hello.txt", " world", " string with newline\n", "hello string with newline\n\n"},
		{"testdata/hello.txt", "hello", "Ж9", "Ж9 world\n"},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).Replace(tc.search, tc.replace).String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%s with %s in %s, want %s, got %s", tc.search, tc.replace, tc.testFileName, tc.want, got)
		}
	}
}

func TestReplaceRegexp(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		regexp       string
		replace      string
		want         string
	}{
		{"testdata/hello.txt", "hel+o", "bye", "bye world\n"},
		{"testdata/hello.txt", "Does not .* in input", "Will not appear in output", "hello world\n"},
		{"testdata/hello.txt", "^([a-z]+) ([a-z]+)", "$1 cruel $2", "hello cruel world\n"},
		{"testdata/hello.txt", "hello{1}", "Ж9", "Ж9 world\n"},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).ReplaceRegexp(regexp.MustCompile(tc.regexp), tc.replace).String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%s with %s in %s, want %s, got %s", tc.regexp, tc.replace, tc.testFileName, tc.want, got)
		}
	}
}

func TestReject(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		reject       string
		want         int
	}{
		{"testdata/test.txt", "line", 1},
		{"testdata/test.txt", "another", 2},
		{"testdata/test.txt", "rhymenocerous", 3},
		{"testdata/empty.txt", "line", 0},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).Reject(tc.reject).CountLines()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q in %q: want %d, got %d", tc.reject, tc.testFileName, tc.want, got)
		}
	}
}

func TestRejectRegexp(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		reject       string
		want         int
	}{
		{"testdata/test.txt", `Hello|line`, 0},
		{"testdata/test.txt", `another`, 2},
		{"testdata/test.txt", `r.*s`, 2},
		{"testdata/empty.txt", "wontmatch", 0},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).RejectRegexp(regexp.MustCompile(tc.reject)).CountLines()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q in %q: want %d, got %d", tc.reject, tc.testFileName, tc.want, got)
		}
	}
}

func TestSHA256Sums(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		want         string
	}{
		// To get the checksum run: openssl dgst -sha256 <file_name>
		{"testdata/sha256Sum.input.txt", "1870478d23b0b4db37735d917f4f0ff9393dd3e52d8b0efa852ab85536ddad8e\n"},
		{"testdata/hello.txt", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\n"},
		{"testdata/multiple_files", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n"},
	}
	for _, tc := range testCases {
		got, err := ListFiles(tc.testFileName).SHA256Sums().String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%q: want %q, got %q", tc.testFileName, tc.want, got)
		}
	}
}