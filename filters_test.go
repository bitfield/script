package script

import (
	"bytes"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
)

// doFiltersOnPipe calls every kind of filter method on the supplied pipe and
// tries to trigger a panic.
func doFiltersOnPipe(t *testing.T, p *Pipe, kind string) {
	var action string
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %s on %s pipe", action, kind)
		}
	}()
	// also tests methods that wrap EachLine, such as Match*/Reject*
	action = "EachLine()"
	output, err := p.EachLine(func(string, *strings.Builder) {}).String()
	if err != nil {
		t.Error(err)
	}
	if output != "" {
		t.Errorf("want zero output from %s on %s pipe, but got %q", action, kind, output)
	}
	action = "Exec()"
	output, err = p.Exec("bogus").String()
	if err != nil && kind == "nil" {
		t.Errorf("%s on %s pipe: %v", action, kind, err)
	}
	if err == nil && kind == "zero" {
		t.Errorf("want error from %s on %s pipe, but got nil", action, kind)
	}
	if output != "" {
		t.Errorf("want zero output from %s on %s pipe, but got %q", action, kind, output)
	}
}
func TestNilPipeFilters(t *testing.T) {
	t.Parallel()
	doFiltersOnPipe(t, nil, "nil")
}

func TestZeroPipeFilters(t *testing.T) {
	t.Parallel()
	doFiltersOnPipe(t, &Pipe{}, "zero")
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
	testCases := []struct {
		testFileName string
		old          string
		new          string
		want         string
	}{
		{"testdata/hello.txt", "hello", "bye", "bye world\n"},
		{"testdata/hello.txt", "Does not exist in input", "Will not appear in output", "hello world\n"},
		{"testdata/hello.txt", "hello world", "string with newline\n", "string with newline\n\n"},
		{"testdata/hello.txt", "hello", "Ж9", "Ж9 world\n"},
	}
	for _, tc := range testCases {
		got, err := File(tc.testFileName).Replace(tc.old, tc.new).String()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("%s with %s in %s, wanted %s, got %s", tc.old, tc.new, tc.testFileName, tc.want, got)
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

func TestConcat(t *testing.T) {
	t.Parallel()
	want, err := ioutil.ReadFile("testdata/concat.golden.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Echo("testdata/test.txt\ntestdata/hello.txt").Concat().Bytes()
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
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
