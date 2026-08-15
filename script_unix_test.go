//go:build !windows

package script_test

import (
	"os"
	"path/filepath"
	"testing"

	"strings"

	"github.com/bitfield/script"
	"github.com/google/go-cmp/cmp"
)

func TestExecForEach_HandlesLongLines(t *testing.T) {
	t.Parallel()
	got, err := script.Echo(longLine).ExecForEach(`echo "{{.}}"`).String()
	if err != nil {
		t.Fatal(err)
	}
	if longLine != got {
		t.Error(cmp.Diff(longLine, got))
	}
}

func TestExecForEach_RunsEchoWithABCAndGetsOutputABC(t *testing.T) {
	t.Parallel()
	p := script.Echo("a\nb\nc\n").ExecForEach("echo {{.}}")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	want := "a\nb\nc\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestExecForEach_CorrectlyEvaluatesTemplateContainingIfStatement(t *testing.T) {
	t.Parallel()
	p := script.Echo("a").ExecForEach("echo {{if .}}it worked!{{end}}")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	want := "it worked!\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestExecCommandPipesDataToExternalCommandAndGetsExpectedOutput(t *testing.T) {
	t.Parallel()
	p := script.File("testdata/hello.txt").ExecCommand("cat")
	want := "hello world"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFindFiles_DoesNotErrorWhenSubDirectoryIsNotReadable(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	restrictedDirPath := filepath.Join(tmpDir, "a_restricted_dir")
	if err := os.Mkdir(restrictedDirPath, 0o000); err != nil {
		t.Fatal(err)
	}
	fileAPath := filepath.Join(tmpDir, "file_a.txt")
	if err := os.WriteFile(fileAPath, []byte("hello world!"), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	got, err := script.FindFiles(tmpDir).String()
	if err != nil {
		t.Fatal(err)
	}
	want := fileAPath + "\n"
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func ExampleExecCommand_ok() {
	script.ExecCommand("echo", "Hello, world!").Stdout()
	// Output:
	// Hello, world!
}

func ExampleFindFiles() {
	script.FindFiles("testdata/multiple_files_with_subdirectory").Stdout()
	// Output:
	// testdata/multiple_files_with_subdirectory/1.txt
	// testdata/multiple_files_with_subdirectory/2.txt
	// testdata/multiple_files_with_subdirectory/3.tar.zip
	// testdata/multiple_files_with_subdirectory/dir/.hidden
	// testdata/multiple_files_with_subdirectory/dir/1.txt
	// testdata/multiple_files_with_subdirectory/dir/2.txt
}

func ExampleIfExists_exec() {
	script.IfExists("./testdata/hello.txt").ExecCommand("echo", "hello").Stdout()
	// Output:
	// hello
}

func ExampleIfExists_noExec() {
	script.IfExists("doesntexist").ExecCommand("echo", "hello").Stdout()
	// Output:
	//
}

func ExampleListFiles() {
	script.ListFiles("testdata/multiple_files_with_subdirectory").Stdout()
	// Output:
	// testdata/multiple_files_with_subdirectory/1.txt
	// testdata/multiple_files_with_subdirectory/2.txt
	// testdata/multiple_files_with_subdirectory/3.tar.zip
	// testdata/multiple_files_with_subdirectory/dir
}

func ExamplePipe_Basename() {
	input := []string{
		"",
		"/",
		"/root",
		"/tmp/example.php",
		"/var/tmp/",
		"./src/filters",
		"C:/Program Files",
	}
	script.Slice(input).Basename().Stdout()
	// Output:
	// .
	// /
	// root
	// example.php
	// tmp
	// filters
	// Program Files
}

func ExamplePipe_Dirname() {
	input := []string{
		"",
		"/",
		"/root",
		"/tmp/example.php",
		"/var/tmp/",
		"./src/filters",
		"C:/Program Files",
	}
	script.Slice(input).Dirname().Stdout()
	// Output:
	// .
	// /
	// /
	// /tmp
	// /var
	// ./src
	// C:
}

func ExamplePipe_ExecCommand() {
	script.Echo("Hello, world!").ExecCommand("tr", "a-z", "A-Z").Stdout()
	// Output:
	// HELLO, WORLD!
}

func ExamplePipe_ExecForEach() {
	script.Echo("a\nb\nc\n").ExecForEach("echo {{.}}").Stdout()
	// Output:
	// a
	// b
	// c
}

func ExamplePipe_Shell() {
	script.Echo("Hello, world!").Shell("tr a-z A-Z").Stdout()
	// Output:
	// HELLO, WORLD!
}

func TestShell_ExpandsEnvironmentVariablesSetViaWithEnv(t *testing.T) {
	t.Parallel()
	env := []string{"ENV1=test1", "ENV2=test2"}
	got, err := script.NewPipe().WithEnv(env).Shell("echo ENV1=$ENV1 ENV2=$ENV2").String()
	if err != nil {
		t.Fatal(err)
	}
	want := "ENV1=test1 ENV2=test2\n"
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestShellExpandsHomeVariableWithoutWithEnv(t *testing.T) {
	t.Parallel()
	p := script.Shell("echo $HOME")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) == "" {
		t.Error("want non-empty $HOME expansion, got empty string")
	}
}

func TestShellPipesDataToExternalCommandAndGetsExpectedOutput(t *testing.T) {
	t.Parallel()
	p := script.File("testdata/hello.txt").Shell("cat")
	want := "hello world"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestShellOnEmptyPipeProducesNoOutputAndNoError(t *testing.T) {
	t.Parallel()
	got, err := script.NewPipe().Shell("cat").String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("want empty output, got %q", got)
	}
}

func TestWithEnv_SetsGivenVariablesForSubsequentExec(t *testing.T) {
	t.Parallel()
	env := []string{"ENV1=test1", "ENV2=test2"}
	got, err := script.NewPipe().WithEnv(env).Shell("echo ENV1=$ENV1 ENV2=$ENV2").String()
	if err != nil {
		t.Fatal(err)
	}
	want := "ENV1=test1 ENV2=test2\n"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWithEnv_UnsetsAllEnvVarsGivenEmptySlice(t *testing.T) {
	t.Parallel()
	p := script.NewPipe().WithEnv([]string{"ENV1=test1"}).Shell("echo ENV1=$ENV1")
	want := "ENV1=test1\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	got, err = p.Echo("").WithEnv([]string{}).Shell("echo ENV1=$ENV1").String()
	if err != nil {
		t.Fatal(err)
	}
	want = "ENV1=\n"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
