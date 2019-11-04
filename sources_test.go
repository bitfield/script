package script

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
		t.Error("want error executing non-existent program, but got nil")
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
		t.Error("want error status on opening non-existent file, but got nil")
	}
}

func TestFindFilesMultipleFiles(t *testing.T) {
	t.Parallel()
	dir := "testdata/multiple_files"
	want := fmt.Sprintf("%s/1.txt\n%s/2.txt\n%s/3.tar.zip\n", dir, dir, dir)
	got, err := FindFiles(dir).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestFindFilesMultipleFilesWithSubdirectory(t *testing.T) {
	t.Parallel()
	dir := "testdata/multiple_files_with_subdirectory"
	want := fmt.Sprintf("%s/1.txt\n%s/2.txt\n%s/3.tar.zip\n%s/dir/.hidden\n%s/dir/1.txt\n%s/dir/2.txt\n", dir, dir, dir, dir, dir, dir)
	got, err := FindFiles(dir).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestFindFilesNonexistent(t *testing.T) {
	t.Parallel()
	p := FindFiles("nonexistentpath")
	if p.Error() == nil {
		t.Error("want error status on listing non-existent path, but got nil")
	}
}

func TestIfExists(t *testing.T) {
	t.Parallel()
	p := IfExists("testdata/doesntexist")
	if p.Error() == nil {
		t.Errorf("want error from IfExists on non-existent file, but got nil")
	}
	p = IfExists("testdata/empty.txt")
	if p.Error() != nil {
		t.Errorf("want no error from IfExists on existing file, but got %v", p.Error())
	}
}

func TestListFilesMultipleFiles(t *testing.T) {
	t.Parallel()
	dir := "testdata/multiple_files"
	want := fmt.Sprintf("%s/1.txt\n%s/2.txt\n%s/3.tar.zip\n", dir, dir, dir)
	got, err := ListFiles(dir).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestListFilesNonexistent(t *testing.T) {
	t.Parallel()
	p := ListFiles("nonexistentpath")
	if p.Error() == nil {
		t.Error("want error status on listing non-existent path, but got nil")
	}
}

func TestListFilesSingle(t *testing.T) {
	t.Parallel()
	got, err := ListFiles("testdata/multiple_files/1.txt").String()
	if err != nil {
		t.Fatal(err)
	}
	want := "testdata/multiple_files/1.txt"
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestListFilesGlob(t *testing.T) {
	t.Parallel()
	dir := "testdata/multiple_files"
	want := fmt.Sprintf("%s/1.txt\n%s/2.txt\n", dir, dir)
	got, err := ListFiles("testdata/multi?le_files/*.txt").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestSlice(t *testing.T) {
	t.Parallel()
	want := "1\n2\n3\n"
	got, err := Slice([]string{"1", "2", "3"}).String()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
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
