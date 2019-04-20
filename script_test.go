package script

import (
	"io/ioutil"
	"testing"
)

func TestEcho(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	got := Echo(want).String()
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
func TestCat(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	got := Cat("testdata/test.txt").Cat().String()
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got := CountLines("testdata/test.txt").Int()
	if got != want {
		t.Fatalf("failed counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got = CountLines("testdata/empty.txt").Int()
	if got != want {
		t.Fatalf("failed counting empty file: want %d, got %d", want, got)
	}
	want = 3
	got = Cat("testdata/test.txt").CountLines().Int()
	if got != want {
		t.Fatalf("failed counting lines from a non-empty pipe: want %d, got %d", want, got)
	}
	want = 1
	got = Cat("testdata/test.txt").CountLines().CountLines().Int()
	if got != want {
		t.Fatalf("failed to count lines in its own output: want %d, got %d", want, got)
	}
}
