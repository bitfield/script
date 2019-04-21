package script

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestWithReader(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := NewPipe().WithReader(strings.NewReader(want))
	got := p.String()
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
func TestFile(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	p := File("testdata/test.txt")
	gotRaw, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		t.Fatal("failed to read file")
	}
	got := string(gotRaw)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestError(t *testing.T) {
	t.Parallel()
	p := File("testdata/nonexistent.txt")
	if p.Error() == nil {
		t.Fatalf("reading nonexistent file: pipe error status should be non-nil")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reading pipe with non-nil error status should succeed, but got: %v", r)
		}
	}()
	_ = p.String()     // not interested in result
	_ = p.CountLines() // not interested in result
}

func TestString(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	p := File("testdata/test.txt")
	got := p.String()
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	_ = p.String() // result should be empty
	if p.Error() == nil {
		t.Fatalf("reading closed pipe: want error, got nil")
	}
	_, err := ioutil.ReadAll(p.Reader)
	if err == nil {
		t.Fatal("failed to close file after reading")
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got := CountLines("testdata/test.txt")
	if got != want {
		t.Fatalf("counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got = CountLines("testdata/empty.txt")
	if got != want {
		t.Fatalf("counting empty file: want %d, got %d", want, got)
	}
	want = 3
	p := File("testdata/test.txt")
	got = p.CountLines()
	if got != want {
		t.Fatalf("counting lines from pipe: want %d, got %d", want, got)
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err == nil {
		fmt.Println(res)
		t.Fatal("failed to close file after reading")
	}
	_ = p.CountLines() // result should be zero
	if p.Error() == nil {
		t.Fatalf("reading closed pipe: want error, got nil")
	}
}
