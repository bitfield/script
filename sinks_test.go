package script_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/bitfield/script"
	"github.com/google/go-cmp/cmp"
)

func TestSinksOnNilPipes(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, nil, "nil")
}

func TestSinksOnZeroPipes(t *testing.T) {
	t.Parallel()
	doSinksOnPipe(t, &script.Pipe{}, "zero")
}

// doSinksOnPipe calls every kind of sink method on the supplied pipe and
// tries to trigger a panic.
func doSinksOnPipe(t *testing.T, p *script.Pipe, kind string) {
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
	_, err = p.WriteFile(t.TempDir() + "/" + kind)
	if err != nil {
		t.Error(err)
	}
	action = "AppendFile()"
	_, err = p.AppendFile(t.TempDir() + "/" + kind)
	if err != nil {
		t.Error(err)
	}
}

func TestAppendFile(t *testing.T) {
	t.Parallel()
	orig := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	// ignore results; we're testing AppendFile, not WriteFile
	_, _ = script.Echo(orig).WriteFile(path)
	extra := " and goodbye"
	wrote, err := script.Echo(extra).AppendFile(path)
	if err != nil {
		t.Error(err)
	}
	if int(wrote) != len(extra) {
		t.Errorf("want %d bytes written, got %d", len(extra), int(wrote))
	}
	// check file contains both contents
	got, err := script.File(path).String()
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
	got, err := script.File(inFile).Bytes()
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
	got, err := script.File("testdata/test.txt").CountLines()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("counting non-empty file: want %d, got %d", want, got)
	}
	want = 0
	got, err = script.File("testdata/empty.txt").CountLines()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("counting empty file: want %d, got %d", want, got)
	}
	want = 3
	p := script.File("testdata/test.txt")
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

func TestSHA256Sum(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testFileName string
		want         string
	}{
		{"testdata/empty.txt", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"testdata/test.txt", "4aea3d470d3f667fdd2dcdb9dbbc77db43872cfbfac1ce15682c4928eae9a146"},
		{"testdata/bytes.bin", "b267dc7e66ee428bc8b51b1114bd0e05bde5c8c5d20ce828fbc95b83060c2f17"},
	}

	for _, tc := range testCases {
		p := script.File(tc.testFileName)
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
	tests := []struct {
		name    string
		fields  *script.Pipe
		want    []string
		wantErr bool
	}{
		{
			name:   "Multiple lines pipe",
			fields: script.Echo("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n"),
			want: []string{
				"testdata/multiple_files/1.txt",
				"testdata/multiple_files/2.txt",
				"testdata/multiple_files/3.tar.zip",
			},
			wantErr: false,
		},
		{
			name:    "Empty pipe",
			fields:  script.Echo(""),
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "Single newline",
			fields:  script.Echo("\n"),
			want:    []string{""},
			wantErr: false,
		},
		{
			name:   "Empty line between two existing lines",
			fields: script.Echo("testdata/multiple_files/1.txt\n\ntestdata/multiple_files/3.tar.zip"),
			want: []string{
				"testdata/multiple_files/1.txt",
				"",
				"testdata/multiple_files/3.tar.zip",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fields
			got, err := p.Slice()
			if (err != nil) != tt.wantErr {
				t.Errorf("Slice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(tt.want, got) {
				t.Error(cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestStdout(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	want := "hello world"
	p := script.File("testdata/hello.txt").WithStdout(buf)
	wrote, err := p.Stdout()
	if err != nil {
		t.Error(err)
	}
	if wrote != len(want) {
		t.Errorf("want %d bytes written, got %d", len(want), wrote)
	}
	got := buf.String()
	if want != got {
		t.Errorf("want %q, got %q", want, string(got))
	}
	_, err = p.String()
	if err == nil {
		t.Error("input not closed after reading")
	}
}

func TestStdoutNoPanicOnNilOrZero(t *testing.T) {
	t.Parallel()
	kind := "nil pipe"
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: Stdout on %s", kind)
		}
	}()
	var p *script.Pipe
	_, _ = p.Stdout()
	kind = "zero pipe"
	p = &script.Pipe{}
	_, _ = p.Stdout()
	kind = "zero pipe with non-empty reader"
	p.Reader = script.NewReadAutoCloser(strings.NewReader("bogus"))
	_, _ = p.Stdout()
}

func TestString(t *testing.T) {
	t.Parallel()
	wantRaw, err := ioutil.ReadFile("testdata/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := string(wantRaw)
	p := script.File("testdata/test.txt")
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

func TestWriteFileNew(t *testing.T) {
	t.Parallel()
	want := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	wrote, err := script.Echo(want).WriteFile(path)
	if err != nil {
		t.Error(err)
	}
	if int(wrote) != len(want) {
		t.Errorf("want %d bytes written, got %d", len(want), int(wrote))
	}
	got, err := script.File(path).String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWriteFileTruncatesExisting(t *testing.T) {
	t.Parallel()
	want := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	// write some data first so we can check for truncation
	data := make([]byte, 15)
	err := os.WriteFile(path, data, 0600)
	if err != nil {
		t.Fatal(err)
	}
	wrote, err := script.Echo(want).WriteFile(path)
	if err != nil {
		t.Error(err)
	}
	if int(wrote) != len(want) {
		t.Errorf("want %d bytes written, got %d", len(want), int(wrote))
	}
	got, err := script.File(path).String()
	if err != nil {
		t.Error(err)
	}
	if got == want+"\x00\x00\x00" {
		t.Fatalf("file not truncated on write")
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
