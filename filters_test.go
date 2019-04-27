package script

import (
	"regexp"
	"strings"
	"testing"
)

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
