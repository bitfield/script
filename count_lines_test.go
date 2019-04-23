package script

import (
	"fmt"
	"io/ioutil"
	"testing"
)

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
