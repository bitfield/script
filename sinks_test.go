package script

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

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
func TestNilPipeSinks(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, nil, "nil")
}

func TestZeroPipeSinks(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, &Pipe{}, "zero")
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
