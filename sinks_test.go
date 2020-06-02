package script

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"io/ioutil"
	"os"
	"testing"
)

// doSinksOnPipe calls every kind of sink method on the supplied pipe and
// tries to trigger a panic.
func doSinksOnPipe(t *testing.T, p *Pipe, kind string) {
	var action string
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %s on %s pipe", action, kind)
		}
	}()
	action = "String()"
	_, err := p.String()
	if err != nil {
		t.Error(err)
	}
	action = "CountLines()"
	_, err = p.CountLines()
	if err != nil {
		t.Error(err)
	}
	action = "SHA256Sum()"
	_, err = p.SHA256Sum()
	if err != nil {
		t.Error(err)
	}
	action = "Slice()"
	_, err = p.Slice()
	if err != nil {
		t.Error(err)
	}
	action = "WriteFile()"
	_, err = p.WriteFile("testdata/tmp" + kind)
	defer os.Remove("testdata/tmp" + kind)
	if err != nil {
		t.Error(err)
	}
	action = "AppendFile()"
	_, err = p.AppendFile("testdata/tmp" + kind)
	if err != nil {
		t.Error(err)
	}
	action = "Stdout()"
	// Ensure we don't clash with TestStdout
	stdoutM.Lock()
	defer stdoutM.Unlock()
	_, err = p.Stdout()
	if err != nil {
		t.Error(err)
	}
}

func TestAppendFile(t *testing.T) {
	t.Parallel()
	orig := "Hello, world"
	testFile := "testdata/appendfile.txt"
	defer os.Remove(testFile)
	// ignore results; we're testing AppendFile, not WriteFile
	_, _ = Echo(orig).WriteFile(testFile)
	extra := " and goodbye"
	wrote, err := Echo(extra).AppendFile(testFile)
	if err != nil {
		t.Error(err)
	}
	if int(wrote) != len(extra) {
		t.Errorf("want %d bytes written, got %d", len(extra), int(wrote))
	}
	// check file contains both contents
	got, err := File(testFile).String()
	if err != nil {
		t.Error(err)
	}
	if got != orig+extra {
		t.Errorf("want %q, got %q", orig+extra, got)
	}
}

func TestBytes(t *testing.T) {
	t.Parallel()
	inFile := "testdata/bytes.bin"
	got, err := File(inFile).Bytes()
	if err != nil {
		t.Error(err)
	}
	want, err := ioutil.ReadFile(inFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	want := 3
	got, err := File("testdata/test.txt").CountLines()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got, err = File("testdata/empty.txt").CountLines()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("counting empty file: want %d, got %d", want, got)
	}
	want = 3
	p := File("testdata/test.txt")
	got, err = p.CountLines()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %d lines, got %d", want, got)
	}
	_, err = ioutil.ReadAll(p.Reader)
	if err == nil {
		t.Error("input not closed after reading")
	}
	_, err = p.CountLines() // result should be zero
	if p.Error() == nil {
		t.Error("want error reading closed pipe, got nil")
	}
	if err != p.Error() {
		t.Errorf("got error %v but pipe error status was %v", err, p.Error())
	}
}

func TestSinksOnNilPipes(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, nil, "nil")
}

func TestSinksOnZeroPipes(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, &Pipe{}, "zero")
}

func TestSHA256Sum(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		want         string
	}{
		{"testdata/empty.txt", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"testdata/test.txt", "a562c9c95e2ff3403e7ffcd8508c6b54d47d5f251387758d3e63dbaaa8296341"},
		{"testdata/bytes.bin", "b267dc7e66ee428bc8b51b1114bd0e05bde5c8c5d20ce828fbc95b83060c2f17"},
	}

	for _, tc := range testCases {
		p := File(tc.testFileName)
		got, err := p.SHA256Sum()
		if err != nil {
			t.Error(err)
		}
		if got != tc.want {
			t.Errorf("want %q, got %q", tc.want, got)
		}
	}
}

func TestSliceSink(t *testing.T) {
	t.Parallel()
	input := Echo("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n")

	want := []string{
		"testdata/multiple_files/1.txt",
		"testdata/multiple_files/2.txt",
		"testdata/multiple_files/3.tar.zip",
	}
	got, err := input.Slice()
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}

	// Empty pipe, should return empty slice
	got, err = Echo("").Slice()
	if err != nil {
		t.Error(err)
	}
	if len(got) != 0 {
		t.Errorf("want zero-length slice, got %v", got)
	}

	// Pipe consists of a single newline, should return 1 element
	want = []string{""}
	got, err = Echo("\n").Slice()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}

	// Empty line between two existing lines
	input = Echo("testdata/multiple_files/1.txt\n\ntestdata/multiple_files/3.tar.zip")

	want = []string{
		"testdata/multiple_files/1.txt",
		"",
		"testdata/multiple_files/3.tar.zip",
	}
	got, err = input.Slice()
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStdout(t *testing.T) {
	t.Parallel()
	// Temporarily point os.Stdout to a file so that we can capture it for
	// testing purposes.
	stdoutM.Lock()
	realStdout := os.Stdout
	stdoutM.Unlock()
	fake, err := ioutil.TempFile("testdata", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fake.Name())
	defer fake.Close()
	// Make sure no other goroutine writes to our fake stdout.
	stdoutM.Lock()
	os.Stdout = fake
	defer func() {
		os.Stdout = realStdout
		stdoutM.Unlock()
	}()
	want := "hello world"
	p := File("testdata/hello.txt")
	wrote, err := p.Stdout()
	if err != nil {
		t.Error(err)
	}
	if wrote != len(want) {
		t.Errorf("want %d bytes written, got %d", len(want), wrote)
	}
	got, err := ioutil.ReadFile(fake.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
	_, err = p.String()
	if err == nil {
		t.Error("input not closed after reading")
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	wantRaw, err := ioutil.ReadFile("testdata/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := string(wantRaw)
	p := File("testdata/test.txt")
	got, err := p.String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
	_, err = p.String() // result should be empty
	if p.Error() == nil {
		t.Error("want error status after read from closed pipe, got nil")
	}
	if err != p.Error() {
		t.Errorf("got error %v but pipe error status was %v", err, p.Error())
	}
	_, err = p.String()
	if err == nil {
		t.Error("input not closed after reading")
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()
	want := "Hello, world"
	testFile := "testdata/writefile.txt"
	defer os.Remove(testFile)
	wrote, err := Echo(want).WriteFile(testFile)
	if err != nil {
		t.Error(err)
	}
	if int(wrote) != len(want) {
		t.Errorf("want %d bytes written, got %d", len(want), int(wrote))
	}
	got, err := File(testFile).String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
