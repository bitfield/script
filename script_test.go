package script

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestFile(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	p := File("testdata/test.txt")
	gotRaw, err := ioutil.ReadAll(p.reader)
	if err != nil {
		t.Fatal("failed to read file")
	}
	got := string(gotRaw)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
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
	_, err := ioutil.ReadAll(p.reader)
	if err == nil {
		t.Fatal("failed to close file after reading")
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got := CountLines("testdata/test.txt")
	if got != want {
		t.Fatalf("failed counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got = CountLines("testdata/empty.txt")
	if got != want {
		t.Fatalf("failed counting empty file: want %d, got %d", want, got)
	}
	want = 3
	p := File("testdata/test.txt")
	got = p.CountLines()
	if got != want {
		t.Fatalf("failed counting lines from a non-empty pipe: want %d, got %d", want, got)
	}
	res, err := ioutil.ReadAll(p.reader)
	if err == nil {
		fmt.Println(res)
		t.Fatal("failed to close file after reading")
	}
}
