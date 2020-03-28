package script

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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
	tcs := []struct {
		Command           string
		ErrExpected       bool
		WantErrContain    string
		WantOutputContain string
	}{
		{
			Command:           "go",
			ErrExpected:       true,
			WantErrContain:    "exit status 2",
			WantOutputContain: "Usage",
		},
		{
			Command:           "doesntexist",
			ErrExpected:       true,
			WantErrContain:    "file not found",
			WantOutputContain: "",
		},
		{
			Command:           "go help",
			ErrExpected:       false,
			WantErrContain:    "",
			WantOutputContain: "Usage",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Command, func(t *testing.T) {
			p := Exec(tc.Command)
			if tc.ErrExpected != (p.Error() != nil) {
				t.Fatalf("unexpected error value: %v", p.Error())
			}
			if p.Error() != nil && !strings.Contains(p.Error().Error(), tc.WantErrContain) {
				t.Fatalf("want error string %q to contain %q", p.Error().Error(), tc.WantErrContain)
			}
			p.SetError(nil) // else p.String() would be a no-op
			output, err := p.String()
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !strings.Contains(output, tc.WantOutputContain) {
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
	type args struct {
		path string
	}
	test1Path := "testdata/multiple_files"
	test2Path := "testdata/multiple_files_with_subdirectory"
	test1Output := fmt.Sprintf("%s/1.txt\n%s/2.txt\n%s/3.tar.zip\n", test1Path, test1Path, test1Path)
	test2Output := fmt.Sprintf("%s/1.txt\n%s/2.txt\n%s/3.tar.zip\n%s/dir/.hidden\n%s/dir/1.txt\n%s/dir/2.txt\n", test2Path, test2Path, test2Path, test2Path, test2Path, test2Path)
	tests := []struct {
		name              string
		args              args
		errExpected       bool
		wantErrContain    string
		wantOutputContain string
	}{
		{
			name: "Multiple Files",
			args: args{
				path: test1Path,
			},
			errExpected:       false,
			wantErrContain:    "",
			wantOutputContain: test1Output,
		},
		{
			name: "Multiple Files with Subdirectories",
			args: args{
				path: test2Path,
			},
			errExpected:       false,
			wantErrContain:    "",
			wantOutputContain: test2Output,
		},
		{
			name: "Non Existent File Path",
			args: args{
				path: "noneexistentpath",
			},
			errExpected:       true,
			wantErrContain:    "no such file or directory",
			wantOutputContain: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FindFiles(tt.args.path)
			if tt.errExpected != (p.Error() != nil) {
				t.Fatalf("unexpected error value: %v", p.Error())
			}
			if p.Error() != nil && !strings.Contains(p.Error().Error(), tt.wantErrContain) {
				t.Fatalf("want error string %q to contain %q", p.Error().Error(), tt.wantErrContain)
			}
			p.SetError(nil) // else p.String() would be a no-op
			output, err := p.String()
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if !strings.Contains(output, tt.wantOutputContain) {
				t.Fatalf("want output %q to contain %q", output, tt.wantOutputContain)
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
