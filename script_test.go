package script

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestWithReader(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := NewPipe().WithReader(strings.NewReader(want))
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
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
		// Reading an erroneous pipe should not panic.
		if r := recover(); r != nil {
			t.Fatalf("panic reading erroneous pipe: %v", r)
		}
	}()
	_, err := p.String()
	if err != p.Error() {
		t.Fatal(err)
	}
	_, err = p.CountLines()
	if err != p.Error() {
		t.Fatal(err)
	}
	e := errors.New("fake error")
	p.SetError(e)
	if p.Error() != e {
		t.Fatalf("setting pipe error: want %v, got %v", e, p.Error())
	}

}

func TestString(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	p := File("testdata/test.txt")
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	_, err = p.String() // result should be empty
	if p.Error() == nil {
		t.Fatalf("reading closed pipe: want error, got nil")
	}
	if err != p.Error() {
		t.Fatalf("returned %v but pipe error status was %v", err, p.Error())
	}
	_, err = ioutil.ReadAll(p.Reader)
	if err == nil {
		t.Fatal("failed to close file after reading")
	}

}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got, err := CountLines("testdata/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got, err = CountLines("testdata/empty.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("counting empty file: want %d, got %d", want, got)
	}
	want = 3
	p := File("testdata/test.txt")
	got, err = p.CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("counting lines from pipe: want %d, got %d", want, got)
	}
	res, err := ioutil.ReadAll(p.Reader)
	if err == nil {
		fmt.Println(res)
		t.Fatal("failed to close file after reading")
	}
	_, err = p.CountLines() // result should be zero
	if p.Error() == nil {
		t.Fatalf("reading closed pipe: want error, got nil")
	}
	if err != p.Error() {
		t.Fatalf("returned %v but pipe error status was %v", err, p.Error())
	}
}
