package script

import (
	"io/ioutil"
	"os"
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
		t.Fatalf("expected error status after read from closed pipe, got nil")
	}
	if err != p.Error() {
		t.Fatalf("returned %v but pipe error status was %v", err, p.Error())
	}
	_, err = p.String()
	if err == nil {
		t.Fatal("input reader not closed")
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got, err := File("testdata/test.txt").CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got, err = File("testdata/empty.txt").CountLines()
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
	_, err = ioutil.ReadAll(p.Reader)
	if err == nil {
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

func TestWriteFile(t *testing.T) {
	t.Parallel()
	// create file with contents
	want := "Hello, world"
	testFile := "testdata/writefile.txt"
	wrote, err := Echo(want).WriteFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)
	if int(wrote) != len(want) {
		t.Fatalf("want %d bytes written, got %d", len(want), int(wrote))
	}
	// check file contains expected
	got, err := File(testFile).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestAppendFile(t *testing.T) {
	t.Parallel()
	// create test file with some contents
	orig := "Hello, world"
	testFile := "testdata/appendfile.txt"
	// don't care about results; we're testing AppendFile, not WriteFile
	_, _ = Echo(orig).WriteFile(testFile)
	defer os.Remove(testFile)
	// append some more contents
	extra := " and goodbye"
	wrote, err := Echo(extra).AppendFile(testFile)
	if int(wrote) != len(extra) {
		t.Fatalf("want %d bytes written, got %d", len(extra), int(wrote))
	}
	// check file contains both contents
	got, err := File(testFile).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != orig+extra {
		t.Fatalf("want %q, got %q", orig+extra, got)
	}
}

func TestStdout(t *testing.T) {
	t.Parallel()
	// Temporarily point os.Stdout to a file so that we can capture it for
	// testing purposes. This is not concurrency-safe (but this is the only
	// test that does it).
	realStdout := os.Stdout
	fake, err := ioutil.TempFile("testdata", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fake.Name())
	defer fake.Close()
	os.Stdout = fake
	defer func() {
		os.Stdout = realStdout
	}()
	want := "hello world"
	p := File("testdata/hello.txt")
	wrote, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if wrote != len(want) {
		t.Fatalf("want %d bytes written, got %d", len(want), wrote)
	}
	got, err := ioutil.ReadFile(fake.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("want %q, got %q", want, string(got))
	}
	_, err = p.String()
	if err == nil {
		t.Fatal("input reader not closed")
	}
}
