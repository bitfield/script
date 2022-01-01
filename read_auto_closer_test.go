package script_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bitfield/script"
)

func TestReadAutoCloser(t *testing.T) {
	t.Parallel()
	wantFile, err := os.Open("testdata/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer wantFile.Close()
	want, err := ioutil.ReadAll(wantFile)
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.Open("testdata/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	acr := script.NewReadAutoCloser(input)
	got, err := ioutil.ReadAll(acr)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
	_, err = ioutil.ReadAll(acr)
	if err == nil {
		t.Error("input not closed after reading")
	}
}
