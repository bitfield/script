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
		t.Fatal("failed to read file")
	}
	got := string(gotRaw)
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	q := File("doesntexist")
	if q.Error() == nil {
		t.Fatalf("expected error status on opening non-existent file, but got nil")
	}

}

func TestEcho(t *testing.T) {
	t.Parallel()
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

func TestExec(t *testing.T) {
	t.Parallel()
	want := "Usage"
	p := Exec("go")
	if p.Error() == nil {
		t.Fatal("expected error from command, but got nil")
	}
	if p.Error().Error() != "exit status 2" {
		t.Fatalf("Expected error 'exit status 2' but got %v", p.Error())
	}
	p.SetError(nil)
	output, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	matches, err := Echo(output).Match(want).CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if matches == 0 {
		t.Fatalf("expected output of command to match %q, but no matches in %q", want, output)
	}
	q := Exec("doesntexist")
	if q.Error() == nil {
		t.Fatal("expected error executing non-existent program, but got nil")
	}
	// ignoring error because we already checked it
	output, _ = q.String()
	if output != "" {
		t.Fatalf("expected no output from running non-existent program, but got %q", output)
	}
	r := Exec("go help")
	if r.Error() != nil {
		t.Fatalf("expected no error running 'go help', but got %v", r.Error())
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
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("want %q, got %q", want, string(got))
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
		t.Fatal(err)
	}
	want := "hello\nworld\n"
	if string(got) != want {
		t.Fatalf("want %q, got %q", want, string(got))
	}

}
