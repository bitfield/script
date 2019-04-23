package script

import (
	"io/ioutil"
	"testing"
)

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

func TestEcho(t *testing.T) {
	want := "Hello, world."
	p := Echo(want)
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
