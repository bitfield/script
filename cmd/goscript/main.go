package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"

	"github.com/bitfield/script"
)

func main() {
	if err := run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	sd := parseFlags(args)

	tmpDir, err := os.MkdirTemp("", "goscript.*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	}()

	mainFile, err := buildGoMain(tmpDir, sd)
	if err != nil {
		return err
	}

	goBinPath, err := exec.LookPath("go")
	if err != nil {
		goBinPath, err = shotgunLookup()
		if err != nil {
			return err
		}
	}
	cmd := exec.Command(goBinPath, "mod", "init", "tmp")
	cmd.Dir = tmpDir
	if _, err := cmd.Output(); err != nil {
		return err
	}
	cmd = exec.Command(goBinPath, "get", "github.com/bitfield/script")
	cmd.Dir = tmpDir

	if _, err := cmd.Output(); err != nil {
		return err
	}
	cmd = exec.Command(goBinPath, "run", mainFile)
	pwd, _ := os.Getwd()
	cmd.Dir = pwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shotgunLookup() (string, error) {
	locations := []string{
		"/usr/local/go/bin/go",
		"/usr/bin/go",
		filepath.Join(os.Getenv("HOME"), "go/bin/go"),
		filepath.Join(os.Getenv("HOME"), ".go/bin/go"),
	}
	if runtime.GOOS == "windows" {
		locations = []string{
			"C:\\Program Files\\Go\\bin\\go.exe",
			"C:\\Go\\bin\\go.exe",
		}
	}
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}
	return "", errors.New("could not locate Go executable")
}

func parseFlags(args []string) scriptData {
	var (
		inlineCode string
		imports    arrayFlags
	)
	flagSet := flag.NewFlagSet(args[0], flag.ExitOnError)
	flagSet.StringVar(&inlineCode, "c", "", "")
	flagSet.Var(&imports, "i", "")
	if err := flagSet.Parse(args[1:]); err != nil {
		flagSet.Usage()
		os.Exit(2)
	}

	sd := scriptData{
		Imports: imports,
	}
	switch {
	case inlineCode != "":
		sd.Script = inlineCode
	case flagSet.NArg() != 0:
		var buf bytes.Buffer
		file := flagSet.Arg(0)
		re := regexp.MustCompile("^#!")
		_, err := script.File(file).
			WithStdout(&buf).
			RejectRegexp(re).
			Stdout()
		if err != nil {
			flagSet.Usage()
			os.Exit(2)
		}
		sd.Script = buf.String()
	default:
		flagSet.Usage()
		os.Exit(2)
	}
	return sd
}

type scriptData struct {
	Script  string
	Imports []string
}

func buildGoMain(scriptDir string, sd scriptData) (string, error) {
	tmpFile := filepath.Join(scriptDir, "script.go")
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer func() {
		err := f.Close()
		if err != nil && !errors.Is(err, fs.ErrClosed) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	}()
	if err := scripTemp.Execute(f, sd); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}

var scripTemp = template.Must(template.New("").Parse(`package main

import (
	{{ range $import := .Imports -}}
	"{{ $import }}"
	{{ end -}}
    "github.com/bitfield/script"
)

func main() {
	{{ .Script }}
}
`))

type arrayFlags []string

// String is an implementation of the flag.Value interface
func (i *arrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

// Set is an implementation of the flag.Value interface
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, strings.Split(value, ",")...)
	return nil
}
