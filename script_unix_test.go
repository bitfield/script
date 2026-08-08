//go:build !windows

package script_test

import (
	"os"
	"path/filepath"
	"testing"

	"bytes"
	"errors"
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

func TestExecRunsShWithEchoHelloAndGetsOutputHello(t *testing.T) {
	t.Parallel()
	p := script.Exec("sh -c 'echo hello'")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	want := "hello\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestExecRunsShWithinShWithEchoInceptionAndGetsOutputInception(t *testing.T) {
	t.Parallel()
	p := script.Exec("sh -c 'sh -c \"echo inception\"'")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	want := "inception\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestExecErrorsRunningShellCommandWithUnterminatedStringArgument(t *testing.T) {
	t.Parallel()
	p := script.Exec("sh -c 'echo oh no")
	p.Wait()
	if p.Error() == nil {
		t.Error("want error running 'sh' command line containing unterminated string")
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

func TestExecPipesDataToExternalCommandAndGetsExpectedOutput(t *testing.T) {
	t.Parallel()
	p := script.File("testdata/hello.txt").Exec("cat")
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

func ExampleExec_ok() {
	script.Exec("echo Hello, world!").Stdout()
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
	script.IfExists("./testdata/hello.txt").Exec("echo hello").Stdout()
	// Output:
	// hello
}

func ExampleIfExists_noExec() {
	script.IfExists("doesntexist").Exec("echo hello").Stdout()
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

func ExamplePipe_Exec() {
	script.Echo("Hello, world!").Exec("tr a-z A-Z").Stdout()
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

func TestShellRunsShWithEchoHelloAndGetsOutputHello(t *testing.T) {
	t.Parallel()
	p := script.Shell("echo hello")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	want := "hello\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
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

func TestShellErrorsRunningCommandThatDoesNotExist(t *testing.T) {
	t.Parallel()
	p := script.Shell("doesntexist_command_xyz")
	p.Wait()
	if p.Error() == nil {
		t.Error("want error running non-existent command")
	}
}

func TestShellSendsStderrOutputToPipeStderr(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	out, err := script.NewPipe().WithStderr(buf).Shell("go").String()
	if err == nil {
		t.Fatal("want error when command returns a non-zero exit status")
	}
	if out != "" {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(buf.String(), "Usage") {
		t.Errorf("want stderr output containing the word 'Usage', got %q", buf.String())
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

func TestShellOnPipeWithExistingErrorIsNoOp(t *testing.T) {
	t.Parallel()
	fakeErr := errors.New("existing error")
	p := script.NewPipe().WithError(fakeErr).Shell("echo hello")
	if p.Error() != fakeErr {
		t.Errorf("want existing error %v preserved, got %v", fakeErr, p.Error())
	}
}

func ExamplePipe_Shell() {
	script.Echo("Hello, world!").Shell("tr a-z A-Z").Stdout()
	// Output:
	// HELLO, WORLD!
}
