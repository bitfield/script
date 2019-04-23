package script

import (
	"io/ioutil"
	"testing"
)

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
