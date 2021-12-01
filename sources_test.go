package script

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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
	tcs := []struct {
		Command           string
		ErrExpected       bool
		WantErrContain    string
		WantOutputContain string
	}{
		{
			Command:           "doesntexist",
			ErrExpected:       true,
			WantErrContain:    "file not found",
			WantOutputContain: "",
		},
		{
			Command:           "go",
			ErrExpected:       true,
			WantErrContain:    "exit status 2",
			WantOutputContain: "Usage",
		},
		{
			Command:           "go help",
			ErrExpected:       false,
			WantErrContain:    "",
			WantOutputContain: "Usage",
		},
		{
			Command:           "sh -c 'echo hello'",
			ErrExpected:       false,
			WantErrContain:    "",
			WantOutputContain: "hello\n",
		},
		{
			Command:           "sh -c 'echo oh no",
			ErrExpected:       true,
			WantErrContain:    "",
			WantOutputContain: "",
		},
		{
			Command:           "sh -c 'sh -c \"echo inception\"'",
			ErrExpected:       false,
			WantErrContain:    "",
			WantOutputContain: "inception\n",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Command, func(t *testing.T) {
			p := Exec(tc.Command)
			output, err := p.String()
			if tc.ErrExpected != (err != nil) {
				t.Fatalf("unexpected error value: %v", err)
			}
			if err != nil && !strings.Contains(err.Error(), tc.WantErrContain) {
				t.Fatalf("want error string %q to contain %q", err.Error(), tc.WantErrContain)
			}
			if err == nil && !strings.Contains(output, tc.WantOutputContain) {
				t.Fatalf("want output %q to contain %q", output, tc.WantOutputContain)
			}
		})
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

func TestFindFiles(t *testing.T) {
	tcs := []struct {
		Path           string
		ErrExpected    bool
		wantErrContain string
		Want           string
	}{
		{
			Path:        "testdata/multiple_files",
			ErrExpected: false,
			Want:        "testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n",
		},
		{
			Path:        "testdata/multiple_files_with_subdirectory",
			ErrExpected: false,
			Want:        "testdata/multiple_files_with_subdirectory/1.txt\ntestdata/multiple_files_with_subdirectory/2.txt\ntestdata/multiple_files_with_subdirectory/3.tar.zip\ntestdata/multiple_files_with_subdirectory/dir/.hidden\ntestdata/multiple_files_with_subdirectory/dir/1.txt\ntestdata/multiple_files_with_subdirectory/dir/2.txt\n",
		},
		{
			Path:        "noneexistentpath",
			ErrExpected: true,
			Want:        "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Path, func(t *testing.T) {
			p := FindFiles(tc.Path)
			if tc.ErrExpected != (p.Error() != nil) {
				t.Fatalf("unexpected error value: %v", p.Error())
			}
			p.SetError(nil) // else p.String() would be a no-op
			got, err := p.String()
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !cmp.Equal(tc.Want, got) {
				t.Fatalf("want %q, got %q", tc.Want, got)
			}
		})
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

func TestStream(t *testing.T) {
	t.Parallel()
	n := 0
	round := 5
	plusOneAndDoubleLine := func(s string, b *strings.Builder) {
		time.Sleep(10 * time.Millisecond)
		n++
		b.WriteRune('\n')
		b.WriteRune('\n')
	}
	timesTwo := func(s string, b *strings.Builder) {
		n *= 2
		b.WriteRune('\n')
	}
	err := Stream().Exec("bash -c 'yes 1 | head -n 5; wait'").EachLine(plusOneAndDoubleLine).EachLine(timesTwo).Wait()
	if err != nil {
		t.Errorf("Got unexpected error %q", err)
	}
	want := 0
	for i := 0; i < round; i++ {
		want++
		want *= 4
	}
	if n != want {
		t.Errorf("want n = %d, got %d", want, n)
	}
}
