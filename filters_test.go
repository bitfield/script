package script

import (
	"errors"
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
			t.Fatalf("panic: %s on %s pipe", action, kind)
		}
	}()
	// also tests methods that wrap EachLine, such as Match*/Reject*
	action = "EachLine()"
	q := p.EachLine(func(string, *strings.Builder) {})
	if q != p {
		t.Fatalf("no-op expected from %s on %s pipe", action, kind)
	}
	action = "Exec()"
	q = p.Exec("bogus")
	if q != p {
		t.Fatalf("no-op expected from %s on %s pipe", action, kind)
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
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("%q in %q: want %d, got %d", tc.match, tc.testFileName, tc.want, got)
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
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("%q in %q: want %d, got %d", tc.match, tc.testFileName, tc.want, got)
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
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("%q in %q: want %d, got %d", tc.reject, tc.testFileName, tc.want, got)
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
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("%q in %q: want %d, got %d", tc.reject, tc.testFileName, tc.want, got)
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
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestExecFilter(t *testing.T) {
	t.Parallel()
	want := "hello world"
	p := File("testdata/hello.txt")
	q := p.Exec("cat")
	got, err := q.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	// This should fail because p is now closed.
	_, err = p.String()
	if err == nil {
		t.Fatal("input reader not closed")
	}
	p = Echo("hello world")
	p.SetError(errors.New("oh no"))
	// This should be a no-op because the pipe has error status.
	out, _ := p.Exec("cat").String()
	if out != "" {
		t.Fatal("expected exec on erroneous pipe to be a no-op, but it wasn't")
	}
}
