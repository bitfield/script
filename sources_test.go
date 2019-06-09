package script

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestFile(t *testing.T) {
	t.Parallel()
	wantRaw, _ := ioutil.ReadFile("testdata/test.txt") // ignoring error
	want := string(wantRaw)
	p := File("testdata/test.txt")
	gotRaw, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		t.Error(err)
	}
	got := string(gotRaw)
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
	q := File("doesntexist")
	if q.Error() == nil {
		t.Errorf("want error status on opening non-existent file, but got nil")
	}
}

func TestListFiles(t *testing.T) {
	// first case
	want := "testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip"
	p := ListFiles("testdata/multiple_files")
	gotRaw, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		t.Error(err)
	}

	got := string(gotRaw)
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}

	// second case
	q := ListFiles("nonexistentpath")
	if q.Error() == nil {
		t.Errorf("want error status on listing non-existent path, but got nil")
	}

	// third case
	want = "testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt"
	z := ListFiles("testdata/multiple_files/*.txt")
	gotRaw, err = ioutil.ReadAll(z.Reader)
	if err != nil {
		t.Error(err)
	}
	got = string(gotRaw)
	if want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestEcho(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := Echo(want)
	got, err := p.String()
	if err != nil {
		t.Error(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExec(t *testing.T) {
	t.Parallel()
	want := "Usage"
	p := Exec("go")
	if p.Error() == nil {
		t.Error("want error from command, but got nil")
	}
	if p.Error().Error() != "exit status 2" {
		t.Errorf("want error 'exit status 2' but got %v", p.Error())
	}
	p.SetError(nil)
	output, err := p.String()
	if err != nil {
		t.Error(err)
	}
	matches, err := Echo(output).Match(want).CountLines()
	if err != nil {
		t.Error(err)
	}
	if matches == 0 {
		t.Errorf("want output of command to match %q, but no matches in %q", want, output)
	}
	q := Exec("doesntexist")
	if q.Error() == nil {
		t.Errorf("want error executing non-existent program, but got nil")
	}
	// ignoring error because we already checked it
	output, _ = q.String()
	if output != "" {
		t.Errorf("want zero output from running non-existent program, but got %q", output)
	}
	r := Exec("go help")
	if r.Error() != nil {
		t.Errorf("want no error running 'go help', but got %v", r.Error())
	}
}

func TestStdin(t *testing.T) {
	t.Parallel()
	// dummy test to prove coverage
	Stdin()
	// now the real test
	want := "hello world"
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), "SCRIPT_TEST=stdin")
	cmd.Stdin = Echo(want).Reader
	got, err := cmd.Output()
	if err != nil {
		t.Error(err)
	}
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}
}

func TestArgs(t *testing.T) {
	t.Parallel()
	// dummy test to prove coverage
	Args()
	// now the real test
	cmd := exec.Command(os.Args[0], "hello", "world")
	cmd.Env = append(os.Environ(), "SCRIPT_TEST=args")
	got, err := cmd.Output()
	if err != nil {
		t.Error(err)
	}
	want := "hello\nworld\n"
	if string(got) != want {
		t.Errorf("want %q, got %q", want, string(got))
	}

}
