package script

import (
	"errors"
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
