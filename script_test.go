package script_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/bitfield/script"
	"github.com/google/go-cmp/cmp"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"args": func() int {
			script.Args().Stdout()
			return 0
		},
		"echostdin": func() int {
			script.Stdin().Stdout()
			return 0
		},
	}))
}

func TestScript(t *testing.T) {
	t.Parallel()
	testscript.Run(t, testscript.Params{
		Dir: "testdata/script",
	})
}

func TestBasenameRemovesLeadingPathComponentsFromInputLines(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		path string
		want string
	}{
		{"\n", ".\n"},
		{"/", "/\n"},
		{"/root", "root\n"},
		{"/tmp/example.php", "example.php\n"},
		{"./src/filters", "filters\n"},
		{"/var/tmp/example.php", "example.php\n"},
		{"/tmp/script-21345.txt\n/tmp/script-5371253.txt", "script-21345.txt\nscript-5371253.txt\n"},
		{"C:/Program Files", "Program Files\n"},
		{"C:/Program Files/", "Program Files\n"},
	}
	for _, tc := range tcs {
		// Expect results to use this platform's path separator
		want := filepath.Clean(tc.want)
		got, err := script.Echo(tc.path).Basename().String()
		if err != nil {
			t.Fatal(err)
		}
		if want != got {
			t.Errorf("%q: want %q, got %q", tc.path, want, got)
		}
	}
}

func TestColumnSelects(t *testing.T) {
	t.Parallel()
	input := []string{
		"60916 s003  Ss+    0:00.51 /bin/bash -l",
		" 6653 s004  R+     0:00.01 ps ax",
		"short line",
		"80159 s004  Ss     0:00.56 /bin/bash -l",
		"60942 s006  Ss+    0:00.53 /bin/bash -l",
		"60943 s007  Ss+    0:00.51 /bin/bash -l",
		"60977 s009  Ss+  	0:00.52 /bin/bash -l",
		"  60978 s010  Ss+    0:00.53 /bin/bash -l",
		"61356 s011	Ss     0:00.54 /bin/bash -l",
	}
	tcs := []struct {
		col  int
		want []string
	}{
		{
			col:  -1,
			want: []string{},
		},
		{
			col:  0,
			want: []string{},
		},
		{
			col: 1,
			want: []string{
				"60916",
				"6653",
				"short",
				"80159",
				"60942",
				"60943",
				"60977",
				"60978",
				"61356",
			},
		},
		{
			col: 2,
			want: []string{
				"s003",
				"s004",
				"line",
				"s004",
				"s006",
				"s007",
				"s009",
				"s010",
				"s011",
			},
		},
		{
			col: 3,
			want: []string{
				"Ss+",
				"R+",
				"Ss",
				"Ss+",
				"Ss+",
				"Ss+",
				"Ss+",
				"Ss",
			},
		},
		{
			col: 4,
			want: []string{
				"0:00.51",
				"0:00.01",
				"0:00.56",
				"0:00.53",
				"0:00.51",
				"0:00.52",
				"0:00.53",
				"0:00.54",
			},
		},
		{
			col: 5,
			want: []string{
				"/bin/bash",
				"ps",
				"/bin/bash",
				"/bin/bash",
				"/bin/bash",
				"/bin/bash",
				"/bin/bash",
				"/bin/bash",
			},
		},
		{
			col: 6,
			want: []string{
				"-l",
				"ax",
				"-l",
				"-l",
				"-l",
				"-l",
				"-l",
				"-l",
			},
		},
		{
			col:  7,
			want: []string{},
		},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("column %d of input", tc.col), func(t *testing.T) {
			got, err := script.Slice(input).Column(tc.col).Slice()
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Error(cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestConcatOutputsContentsOfSpecifiedFilesInOrder(t *testing.T) {
	t.Parallel()
	want := "This is the first line in the file.\nHello, world.\nThis is another line in the file.\nhello world"
	got, err := script.Echo("testdata/test.txt\ntestdata/doesntexist.txt\ntestdata/hello.txt").Concat().String()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestDirname_RemovesFilenameComponentFromInputLines(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		path string
		want string
	}{
		{"/", "/\n"},
		{"./", ".\n"},
		{"\n", ".\n"},
		{"/root", "/\n"},
		{"/tmp/example.php", "/tmp\n"},
		{"/var/tmp/example.php", "/var/tmp\n"},
		{"/var/tmp", "/var\n"},
		{"/var/tmp/", "/var\n"},
		{"src/filters", "./src\n"},
		{"src/filters/", "./src\n"},
		{"/tmp/script-21345.txt\n/tmp/script-5371253.txt", "/tmp\n/tmp\n"},
		{"C:/Program Files/PHP", "C:/Program Files\n"},
		{"C:/Program Files/PHP/", "C:/Program Files\n"},
	}
	for _, tc := range tcs {
		// Expect results to use this platform's path separator
		want := filepath.Clean(tc.want)
		got, err := script.Echo(tc.path).Dirname().String()
		if err != nil {
			t.Fatal(err)
		}
		if want != got {
			t.Errorf("%q: want %q, got %q", tc.path, want, got)
		}
	}
}

func TestDoPerformsSuppliedHTTPRequest(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	want := "some data\n"
	got, err := script.Do(req).String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestEachLine_FiltersInputThroughSuppliedFunction(t *testing.T) {
	t.Parallel()
	want := "Hello world\nGoodbye world\n"
	got, err := script.Echo("Hello\nGoodbye").
		EachLine(func(line string, out *strings.Builder) {
			out.WriteString(line + " world\n")
		}).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestEachLine_HandlesLongLines(t *testing.T) {
	t.Parallel()
	var got int
	_, err := script.Echo(longLine).
		EachLine(func(line string, out *strings.Builder) {
			got++
		}).String()
	if err != nil {
		t.Fatal(err)
	}
	want := 2
	if want != got {
		t.Errorf("want %d lines counted, got %d", want, got)
	}
}

func TestEchoProducesSuppliedString(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := script.Echo(want)
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestEchoReplacesInputWithSuppliedStringWhenUsedAsFilter(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := script.Echo("bogus").Echo(want)
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExecForEach_ErrorsOnInvalidTemplateSyntax(t *testing.T) {
	t.Parallel()
	p := script.Echo("a\nb\nc\n").ExecForEach("{{invalid template syntax}}")
	p.Wait()
	if p.Error() == nil {
		t.Error("want error with invalid template syntax")
	}
}

func TestExecForEach_ErrorsOnUnbalancedQuotes(t *testing.T) {
	t.Parallel()
	p := script.Echo("a\nb\nc\n").ExecForEach("echo \"{{.}}")
	p.Wait()
	if p.Error() == nil {
		t.Error("want error with unbalanced quotes in command line")
	}
}

func TestExecForEach_SendsStderrOutputToPipeStderr(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	out, err := script.Echo("go").WithStderr(buf).ExecForEach("{{.}}").String()
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(buf.String(), "Usage") {
		t.Errorf("want stderr output containing the word 'Usage', got %q", buf.String())
	}
}

func TestExecSendsStderrOutputToPipeStderr(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	out, err := script.NewPipe().WithStderr(buf).Exec("go").String()
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

func TestFilterByCopyPassesInputThroughUnchanged(t *testing.T) {
	t.Parallel()
	p := script.Echo("hello").Filter(func(r io.Reader, w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	})
	want := "hello"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterCanChainFilters(t *testing.T) {
	t.Parallel()
	p := script.Echo("hello").Filter(func(r io.Reader, w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	}).Filter(func(r io.Reader, w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	})
	want := "hello"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterByCopyToDiscardGivesNoOutput(t *testing.T) {
	t.Parallel()
	p := script.Echo("hello").Filter(func(r io.Reader, w io.Writer) error {
		_, err := io.Copy(io.Discard, r)
		return err
	})
	want := ""
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterReadsNoMoreThanRequested(t *testing.T) {
	t.Parallel()
	input := "firstline\nsecondline"
	source := bytes.NewBufferString(input)
	p := script.NewPipe().WithReader(source).Filter(func(r io.Reader, w io.Writer) error {
		// read just the first line of input
		var text string
		_, err := fmt.Fscanln(r, &text)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, text)
		return nil
	})
	want := "firstline\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Fatal(cmp.Diff(want, got))
	}
	wantRemaining := "secondline"
	if wantRemaining != source.String() {
		t.Errorf("want %q remaining, got %q", wantRemaining, source.String())
	}
}

func TestFilterByFirstLineOnlyGivesFirstLineOfInput(t *testing.T) {
	t.Parallel()
	p := script.Echo("hello\nworld").Filter(func(r io.Reader, w io.Writer) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			fmt.Fprintln(w, scanner.Text())
			break
		}
		return scanner.Err()
	})
	want := "hello\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterSetsErrorOnPipeIfFilterFuncReturnsError(t *testing.T) {
	t.Parallel()
	p := script.Echo("hello").Filter(func(io.Reader, io.Writer) error {
		return errors.New("oh no")
	})
	io.ReadAll(p)
	if p.Error() == nil {
		t.Error("no error")
	}
}

func TestFilterLine_FiltersEachLineThroughSuppliedFunction(t *testing.T) {
	t.Parallel()
	input := "hello\nworld"
	want := "HELLO\nWORLD\n"
	got, err := script.Echo(input).FilterLine(strings.ToUpper).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterScan_FiltersInputLineByLine(t *testing.T) {
	t.Parallel()
	input := "hello\nworld\ngoodbye"
	want := "world\n"
	got, err := script.Echo(input).
		FilterScan(func(line string, w io.Writer) {
			if strings.HasPrefix(line, "w") {
				fmt.Fprintln(w, line)
			}
		}).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFilterScan_HandlesLongLines(t *testing.T) {
	t.Parallel()
	want := "last line\n"
	got, err := script.Echo(longLine).
		FilterScan(func(line string, w io.Writer) {
			if strings.HasPrefix(line, "last") {
				fmt.Fprintln(w, line)
			}
		}).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFirstDropsAllButFirstNLinesOfInput(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := "a\nb\n"
	got, err := script.Echo(input).First(2).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFirstHasNoOutputWhenNIs0(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := ""
	got, err := script.Echo(input).First(0).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFirstHasNoOutputWhenNIsNegative(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := ""
	got, err := script.Echo(input).First(-1).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFirstHasNoEffectGivenLessThanNInputLines(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := "a\nb\nc\n"
	got, err := script.Echo(input).First(4).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFirstDoesNotConsumeUnnecessaryData(t *testing.T) {
	t.Parallel()
	// First uses a 4096-byte buffer, so will always read at least
	// that much, but no more (once N lines have been read).
	r := strings.NewReader(strings.Repeat("line\n", 1000))
	got, err := script.NewPipe().WithReader(r).First(1).String()
	if err != nil {
		t.Fatal(err)
	}
	want := "line\n"
	if want != got {
		t.Errorf("want output %q, got %q", want, got)
	}
	if r.Len() == 0 {
		t.Errorf("no data left in reader")
	}
}

func TestFreqHandlesLongLines(t *testing.T) {
	t.Parallel()
	got, err := script.Echo(longLine).Freq().Slice()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 results, got %d: %q", len(got), got)
	}
	if got[0] != "1 last line" {
		t.Fatalf("wrong result: %q", got)
	}
}

func TestFreqProducesCorrectFrequencyTableForInput(t *testing.T) {
	t.Parallel()
	input := strings.Join([]string{
		"apple",
		"orange",
		"banana",
		"banana",
		"apple",
		"orange",
		"kumquat",
		"apple",
		"orange",
		"apple",
		"banana",
		"banana",
		"apple",
		"apple",
		"orange",
		"apple",
		"apple",
		"apple",
		"apple",
	}, "\n")
	want := "10 apple\n 4 banana\n 4 orange\n 1 kumquat\n"
	got, err := script.Echo(input).Freq().String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetMakesHTTPGetRequestToGivenURL(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want HTTP method GET, got %q", r.Method)
		}
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	want := "some data\n"
	got, err := script.Get(ts.URL).String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetSetsErrorStatusWhenHTTPResponseStatusIsNotOK(t *testing.T) {
	t.Parallel()
	// With no handler, all requests will get 404
	ts := httptest.NewServer(nil)
	defer ts.Close()
	p := script.Get(ts.URL)
	p.Wait()
	if p.Error() == nil {
		t.Fatalf("want error for non-OK request, got nil")
	}
}

func TestGetConsidersHTTPStatus201CreatedToBeOK(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	want := "some data\n"
	got, err := script.Get(ts.URL).String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetUsesPipeContentsAsRequestBody(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := []byte("request data")
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal("reading request body", err)
		}
		if !cmp.Equal(want, got) {
			t.Fatalf(cmp.Diff(want, string(got)))
		}
	}))
	defer ts.Close()
	_, err := script.Echo("request data").Get(ts.URL).String()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJoinHandlesLongLines(t *testing.T) {
	t.Parallel()
	result, err := script.Echo(longLine).Join().String()
	if err != nil {
		t.Fatal(err)
	}
	want := len(longLine)
	if want != len(result) {
		t.Errorf("want result length %d, got %d", want, len(result))
	}
}

func TestJoinJoinsInputLinesIntoSpaceSeparatedString(t *testing.T) {
	t.Parallel()
	input := "hello\nfrom\nthe\njoin\ntest"
	want := "hello from the join test\n"
	got, err := script.Echo(input).Join().String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestJQWithDotQueryPrettyPrintsInput(t *testing.T) {
	t.Parallel()
	input := `{"timestamp": 1649264191, "iss_position": {"longitude": "52.8439", "latitude": "10.8107"}, "message": "success"}`
	// Fields should be sorted by key, with whitespace removed
	want := `{"iss_position":{"latitude":"10.8107","longitude":"52.8439"},"message":"success","timestamp":1649264191}` + "\n"
	got, err := script.Echo(input).JQ(".").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(want, got)
		t.Error(cmp.Diff(want, got))
	}
}

func TestJQWithFieldQueryProducesSelectedField(t *testing.T) {
	t.Parallel()
	input := `{"timestamp": 1649264191, "iss_position": {"longitude": "52.8439", "latitude": "10.8107"}, "message": "success"}`
	want := `{"latitude":"10.8107","longitude":"52.8439"}` + "\n"
	got, err := script.Echo(input).JQ(".iss_position").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(want, got)
		t.Error(cmp.Diff(want, got))
	}
}

func TestJQWithArrayQueryProducesRequiredArray(t *testing.T) {
	t.Parallel()
	input := `{"timestamp": 1649264191, "iss_position": {"longitude": "52.8439", "latitude": "10.8107"}, "message": "success"}`
	want := `["10.8107","52.8439"]` + "\n"
	got, err := script.Echo(input).JQ("[.iss_position.latitude, .iss_position.longitude]").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(want, got)
		t.Error(cmp.Diff(want, got))
	}
}

func TestJQWithArrayInputAndElementQueryProducesSelectedElement(t *testing.T) {
	t.Parallel()
	input := `[1, 2, 3]`
	want := "1\n"
	got, err := script.Echo(input).JQ(".[0]").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(want, got)
		t.Error(cmp.Diff(want, got))
	}
}

func TestJQHandlesGithubJSONWithRealWorldExampleQuery(t *testing.T) {
	t.Parallel()
	want := `{"message":"restore sample log data (fixes #102)","name":"John Arundel"}` + "\n"
	got, err := script.File("testdata/commits.json").
		JQ(".[0] | {message: .commit.message, name: .commit.committer.name}").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(want, got)
		t.Error(cmp.Diff(want, got))
	}
}

func TestJQErrorsWithInvalidQuery(t *testing.T) {
	t.Parallel()
	input := `[1, 2, 3]`
	_, err := script.Echo(input).JQ(".foo & .bar").String()
	if err == nil {
		t.Error("want error from invalid JQ query, got nil")
	}
}

func TestLastDropsAllButLastNLinesOfInput(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := "b\nc\n"
	got, err := script.Echo(input).Last(2).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLastHandlesLongLines(t *testing.T) {
	t.Parallel()
	want := "last line\n"
	got, err := script.Echo(longLine).Last(1).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLastHasNoOutputWhenNIs0(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := ""
	got, err := script.Echo(input).Last(0).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLastHasNoOutputWhenNIsNegative(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := ""
	got, err := script.Echo(input).Last(-1).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLastHasNoEffectGivenLessThanNInputLines(t *testing.T) {
	t.Parallel()
	input := "a\nb\nc\n"
	want := "a\nb\nc\n"
	got, err := script.Echo(input).Last(4).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestMatchOutputsOnlyMatchingLinesOfInput(t *testing.T) {
	t.Parallel()
	input := "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n"
	tcs := []struct {
		match, want string
	}{
		{
			match: "line",
			want:  "This is the first line in the file.\nThis is another line in the file.\n",
		},
		{
			match: "another",
			want:  "This is another line in the file.\n",
		},
		{
			match: "definitely won't match any lines",
			want:  "",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).Match(tc.match).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestMatchOutputsNothingGivenEmptyInput(t *testing.T) {
	t.Parallel()
	got, err := script.NewPipe().Match("anything").String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Error("want no output given empty input")
	}
}

func TestMatchRegexp_OutputsOnlyLinesMatchingRegexp(t *testing.T) {
	t.Parallel()
	input := "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n"
	tcs := []struct {
		regex, want string
	}{
		{
			regex: `Hello|file`,
			want:  "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n",
		},
		{
			regex: `an.ther`,
			want:  "This is another line in the file.\n",
		},
		{
			regex: `r[a-z]*s`,
			want:  "This is the first line in the file.\n",
		},
		{
			regex: `r[a-z]+s`,
			want:  "",
		},
		{
			regex: `bogus$`,
			want:  "",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).MatchRegexp(regexp.MustCompile(tc.regex)).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestReplaceReplacesMatchesWithSpecifiedText(t *testing.T) {
	t.Parallel()
	input := "hello world"
	tcs := []struct {
		search, replace, want string
	}{
		{
			search:  "hello",
			replace: "bye",
			want:    "bye world\n",
		},
		{
			search:  "Does not exist in input",
			replace: "Will not appear in output",
			want:    "hello world\n",
		},
		{
			search:  " world",
			replace: " string with newline\n",
			want:    "hello string with newline\n\n",
		},
		{
			search:  "hello",
			replace: "Ж9",
			want:    "Ж9 world\n",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).Replace(tc.search, tc.replace).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestReplaceRegexp_ReplacesMatchesWithSpecifiedText(t *testing.T) {
	t.Parallel()
	input := "hello world"
	tcs := []struct {
		regex, replace, want string
	}{
		{
			regex:   "hel+o",
			replace: "bye",
			want:    "bye world\n",
		},
		{
			regex:   "Does not .* in input",
			replace: "Will not appear in output",
			want:    "hello world\n",
		},
		{
			regex:   "^([a-z]+) ([a-z]+)",
			replace: "$1 cruel $2",
			want:    "hello cruel world\n",
		},
		{
			regex:   "hello{1}",
			replace: "Ж9",
			want:    "Ж9 world\n",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).ReplaceRegexp(regexp.MustCompile(tc.regex), tc.replace).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestRejectDropsMatchingLinesFromInput(t *testing.T) {
	t.Parallel()
	input := "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n"
	tcs := []struct {
		reject, want string
	}{
		{
			reject: "line",
			want:   "Hello, world.\n",
		},
		{
			reject: "another",
			want:   "This is the first line in the file.\nHello, world.\n",
		},
		{
			reject: "definitely won't match any lines",
			want:   "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).Reject(tc.reject).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestPostPostsToGivenURLUsingPipeAsRequestBody(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("want HTTP method POST, got %q", r.Method)
		}
		want := []byte("request data")
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal("reading request body", err)
		}
		if !cmp.Equal(want, got) {
			t.Fatal(cmp.Diff(want, string(got)))
		}
		fmt.Fprintln(w, "response data")
	}))
	defer ts.Close()
	want := "response data\n"
	got, err := script.Echo("request data").Post(ts.URL).String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestRejectRegexp_DropsMatchingLinesFromInput(t *testing.T) {
	t.Parallel()
	input := "hello world"
	tcs := []struct {
		regex, want string
	}{
		{
			regex: `Hello|line`,
			want:  "hello world\n",
		},
		{
			regex: `hello|bogus`,
			want:  "",
		},
		{
			regex: `w.*d`,
			want:  "",
		},
		{
			regex: "wontmatch",
			want:  "hello world\n",
		},
	}
	for _, tc := range tcs {
		got, err := script.Echo(input).RejectRegexp(regexp.MustCompile(tc.regex)).String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Error(cmp.Diff(tc.want, got))
		}
	}
}

func TestSHA256Sums_OutputsCorrectHashForEachSpecifiedFile(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		testFileName string
		want         string
	}{
		// To get the checksum run: openssl dgst -sha256 <file_name>
		{"testdata/hashSum.input.txt", "1870478d23b0b4db37735d917f4f0ff9393dd3e52d8b0efa852ab85536ddad8e\n"},
		{"testdata/hello.txt", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\n"},
		{"testdata/multiple_files", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n"},
	}
	for _, tc := range tcs {
		got, err := script.ListFiles(tc.testFileName).SHA256Sums().String()
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%q: want %q, got %q", tc.testFileName, tc.want, got)
		}
	}
}

func TestTeeUsesConfiguredStdoutAsDefault(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	_, err := script.Echo("hello").WithStdout(buf).Tee().String()
	if err != nil {
		t.Fatal(err)
	}
	want := "hello"
	got := buf.String()
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestTeeWritesDataToSuppliedWritersAsWellAsToPipe(t *testing.T) {
	t.Parallel()
	buf1, buf2 := new(bytes.Buffer), new(bytes.Buffer)
	got, err := script.Echo("hello world").Tee(buf1, buf2).String()
	if err != nil {
		t.Fatal(err)
	}
	want := "hello world"
	if want != got {
		t.Errorf("want %q on pipe, got %q", want, got)
	}
	if want != buf1.String() {
		t.Errorf("want %q on writer 1, got %q", want, buf1.String())
	}
	if want != buf2.String() {
		t.Errorf("want %q on writer 2, got %q", want, buf2.String())
	}
}

func TestExecErrorsWhenTheSpecifiedCommandDoesNotExist(t *testing.T) {
	t.Parallel()
	p := script.Exec("doesntexist")
	p.Wait()
	if p.Error() == nil {
		t.Error("want error running non-existent command")
	}
}

func TestExecRunsGoWithNoArgsAndGetsUsageMessagePlusErrorExitStatus2(t *testing.T) {
	t.Parallel()
	// We can't make many cross-platform assumptions about what external
	// commands will be available, but it seems logical that 'go' would be
	// (though it may not be in the user's path)
	p := script.Exec("go")
	output, err := p.String()
	if err == nil {
		t.Fatal("want error when command returns a non-zero exit status")
	}
	if !strings.Contains(output, "Usage") {
		t.Fatalf("want output containing the word 'Usage', got %q", output)
	}
	want := 2
	got := p.ExitStatus()
	if want != got {
		t.Errorf("want exit status %d, got %d", want, got)
	}
}

func TestExecRunsGoHelpAndGetsUsageMessage(t *testing.T) {
	t.Parallel()
	p := script.Exec("go help")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	output, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "Usage") {
		t.Fatalf("want output containing the word 'Usage', got %q", output)
	}
}

func TestFileOutputsContentsOfSpecifiedFile(t *testing.T) {
	t.Parallel()
	want := "This is the first line in the file.\nHello, world.\nThis is another line in the file.\n"
	got, err := script.File("testdata/test.txt").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestFileErrorsOnNonexistentFile(t *testing.T) {
	t.Parallel()
	p := script.File("doesntexist")
	if p.Error() == nil {
		t.Error("want error for non-existent file")
	}
}

func TestFindFiles_ReturnsListOfFiles(t *testing.T) {
	t.Parallel()
	p := script.FindFiles("testdata/multiple_files")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	p.SetError(nil) // else p.String() would be a no-op
	// Expect result to use this platform's path separator
	want := filepath.Clean("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n")
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestFindFiles_RecursesIntoSubdirectories(t *testing.T) {
	t.Parallel()
	p := script.FindFiles("testdata/multiple_files_with_subdirectory")
	if p.Error() != nil {
		t.Fatal(p.Error())
	}
	p.SetError(nil) // else p.String() would be a no-op
	// Expect result to use this platform's path separator
	want := filepath.Clean("testdata/multiple_files_with_subdirectory/1.txt\ntestdata/multiple_files_with_subdirectory/2.txt\ntestdata/multiple_files_with_subdirectory/3.tar.zip\ntestdata/multiple_files_with_subdirectory/dir/.hidden\ntestdata/multiple_files_with_subdirectory/dir/1.txt\ntestdata/multiple_files_with_subdirectory/dir/2.txt\n")
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestFindFiles_InNonexistentPathReturnsError(t *testing.T) {
	t.Parallel()
	p := script.FindFiles("nonexistent_path")
	if p.Error() == nil {
		t.Fatal("want error for nonexistent path")
	}
}

func TestIfExists_ProducesErrorPlusNoOutputForNonexistentFile(t *testing.T) {
	t.Parallel()
	want := ""
	got, err := script.IfExists("testdata/doesntexist").Echo("hello").String()
	if err == nil {
		t.Fatal("want error")
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestIfExists_ProducesOutputAndNoErrorWhenFileExists(t *testing.T) {
	t.Parallel()
	want := "hello"
	got, err := script.IfExists("testdata/empty.txt").Echo("hello").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestListFiles_OutputsDirectoryContentsGivenDirectoryPath(t *testing.T) {
	t.Parallel()
	want := filepath.Clean("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n")
	got, err := script.ListFiles("testdata/multiple_files").String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestListFiles_ErrorsOnNonexistentPath(t *testing.T) {
	t.Parallel()
	p := script.ListFiles("nonexistentpath")
	if p.Error() == nil {
		t.Error("want error status on listing non-existent path, but got nil")
	}
}

func TestListFiles_OutputsSingleFileGivenFilePath(t *testing.T) {
	t.Parallel()
	got, err := script.ListFiles("testdata/multiple_files/1.txt").String()
	if err != nil {
		t.Fatal(err)
	}
	want := "testdata/multiple_files/1.txt"
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestListFiles_OutputsAllFilesMatchingSpecifiedGlobExpression(t *testing.T) {
	t.Parallel()
	want := filepath.Clean("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\n")
	got, err := script.ListFiles("testdata/multi?le_files/*.txt").String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Errorf("Want %q, got %q", want, got)
	}
}

func TestReadAutoCloser_ReadsAllDataFromSourceAndClosesItAutomatically(t *testing.T) {
	t.Parallel()
	want := []byte("hello world")
	input, err := os.Open("testdata/hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	acr := script.NewReadAutoCloser(input)
	got, err := io.ReadAll(acr)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
	_, err = io.ReadAll(acr)
	if err == nil {
		t.Error("input not closed after reading")
	}
}

func TestSliceProducesElementsOfSpecifiedSliceOnePerLine(t *testing.T) {
	t.Parallel()
	want := "1\n2\n3\n"
	got, err := script.Slice([]string{"1", "2", "3"}).String()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestSliceGivenEmptySliceProducesEmptyPipe(t *testing.T) {
	t.Parallel()
	want := ""
	got, err := script.Slice([]string{}).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestStdoutReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithStdout(io.Discard).
		WithReader(brokenReader).Stdout()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestStdoutSendsPipeContentsToConfiguredStandardOutput(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	want := "hello world"
	p := script.File("testdata/hello.txt").WithStdout(buf)
	wrote, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if wrote != len(want) {
		t.Fatalf("want %d bytes written, got %d", len(want), wrote)
	}
	got := buf.String()
	if want != got {
		t.Fatalf("want %q, got %q", want, string(got))
	}
	_, err = p.String()
	if err == nil {
		t.Error("input not closed after reading")
	}
}

func TestAppendFile_AppendsAllItsInputToSpecifiedFile(t *testing.T) {
	t.Parallel()
	orig := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	_, err := script.Echo(orig).WriteFile(path)
	if err != nil {
		t.Fatal(err)
	}
	extra := " and goodbye"
	wrote, err := script.Echo(extra).AppendFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int(wrote) != len(extra) {
		t.Fatalf("want %d bytes written, got %d", len(extra), int(wrote))
	}
	// check file contains both contents
	got, err := script.File(path).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != orig+extra {
		t.Errorf("want %q, got %q", orig+extra, got)
	}
}

func TestAppendFile_ReturnsBytesWrittenAndErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	var want int64 = 1
	got, err := script.NewPipe().WithReader(partialErrReader{}).AppendFile(t.TempDir() + "/tmp")
	if err == nil {
		t.Fatal("want error reading pipe with error status, got nil")
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestBytesOutputsInputBytesUnchanged(t *testing.T) {
	t.Parallel()
	want := []byte{8, 0, 0, 16}
	input := bytes.NewReader(want)
	got, err := script.NewPipe().WithReader(input).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestBytesReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).Bytes()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestCountLines_CountsCorrectNumberOfLinesInInput(t *testing.T) {
	t.Parallel()
	want := 3
	got, err := script.Echo("a\nb\nc").CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %d, got %d", want, got)
	}
}

func TestCountLines_Counts0LinesInEmptyInput(t *testing.T) {
	t.Parallel()
	want := 0
	got, err := script.Echo("").CountLines()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %d, got %d", want, got)
	}
}

func TestCountLines_ReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).CountLines()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestSHA256Sum_OutputsCorrectHash(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name, input, want string
	}{
		{
			name:  "for no data",
			input: "",
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:  "for short string",
			input: "hello, world",
			want:  "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b",
		},
		{
			name:  "for string containing newline",
			input: "The tao that can be told\nis not the eternal Tao",
			want:  "788542cb92d37f67e187992bdb402fdfb68228a1802947f74c6576e04790a688",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := script.Echo(tc.input).SHA256Sum()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestSHA256Sum_ReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).SHA256Sum()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestSliceSink_ReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).Slice()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestSliceSink_(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		pipe *script.Pipe
		want []string
	}{
		{
			name: "returns three elements for three lines of input",
			pipe: script.Echo("testdata/multiple_files/1.txt\ntestdata/multiple_files/2.txt\ntestdata/multiple_files/3.tar.zip\n"),
			want: []string{
				"testdata/multiple_files/1.txt",
				"testdata/multiple_files/2.txt",
				"testdata/multiple_files/3.tar.zip",
			},
		},
		{
			name: "returns an empty slice given empty input",
			pipe: script.Echo(""),
			want: []string{},
		},
		{
			name: "returns one empty string given input containing a single newline",
			pipe: script.Echo("\n"),
			want: []string{""},
		},
		{
			name: "returns an empty string for each empty input line",
			pipe: script.Echo("testdata/multiple_files/1.txt\n\ntestdata/multiple_files/3.tar.zip"),
			want: []string{
				"testdata/multiple_files/1.txt",
				"",
				"testdata/multiple_files/3.tar.zip",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.pipe
			got, err := p.Slice()
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(tt.want, got) {
				t.Error(cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestStringOutputsInputStringUnchanged(t *testing.T) {
	t.Parallel()
	want := "hello, world"
	got, err := script.Echo(want).String()
	if err != nil {
		t.Fatal(err)
	}
	if want != got {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStringReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).String()
	if err == nil {
		t.Fatal(nil)
	}
}

func TestWaitReadsPipeSourceToCompletion(t *testing.T) {
	t.Parallel()
	source := bytes.NewBufferString("hello")
	script.NewPipe().WithReader(source).FilterLine(strings.ToUpper).Wait()
	if source.Len() > 0 {
		t.Errorf("incomplete read: %d bytes of input remaining: %q", source.Len(), source.String())
	}
}

func TestWriteFile_WritesInputToFileCreatingItIfNecessary(t *testing.T) {
	t.Parallel()
	want := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	wrote, err := script.Echo(want).WriteFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int(wrote) != len(want) {
		t.Fatalf("want %d bytes written, got %d", len(want), int(wrote))
	}
	got, err := script.File(path).String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// partialErrReader returns 1 and a non-EOF error on reading.
type partialErrReader struct{}

func (r partialErrReader) Read(p []byte) (int, error) {
	return 1, errors.New("oh no")
}

func TestWriteFile_ReturnsBytesWrittenAndErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	var want int64 = 1
	got, err := script.NewPipe().WithReader(partialErrReader{}).WriteFile(t.TempDir() + "/tmp")
	if err == nil {
		t.Fatal("want error reading pipe with error status, got nil")
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestWriteFile_TruncatesExistingFile(t *testing.T) {
	t.Parallel()
	want := "Hello, world"
	path := t.TempDir() + "/" + t.Name()
	// write some data first so we can check for truncation
	data := make([]byte, 15)
	err := os.WriteFile(path, data, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	wrote, err := script.Echo(want).WriteFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int(wrote) != len(want) {
		t.Fatalf("want %d bytes written, got %d", len(want), int(wrote))
	}
	got, err := script.File(path).String()
	if err != nil {
		t.Fatal(err)
	}
	if got == want+"\x00\x00\x00" {
		t.Fatalf("file not truncated on write")
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWithHTTPClient_SetsSuppliedClientOnPipe(t *testing.T) {
	t.Parallel()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	want := "some data\n"
	// Unless the pipe uses the supplied ts.Client, we'll get a
	// 'certificate is not trusted' error on making the request
	got, err := script.NewPipe().WithHTTPClient(ts.Client()).Do(req).String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestWithReader_SetsSuppliedReaderOnPipe(t *testing.T) {
	t.Parallel()
	want := "Hello, world."
	p := script.NewPipe().WithReader(strings.NewReader(want))
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWithError_SetsSpecifiedErrorOnPipe(t *testing.T) {
	t.Parallel()
	fakeErr := errors.New("oh no")
	p := script.NewPipe().WithError(fakeErr)
	if p.Error() != fakeErr {
		t.Errorf("want %q, got %q", fakeErr, p.Error())
	}
}

func TestWithStdout_SetsSpecifiedWriterAsStdout(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	want := "Hello, world."
	_, err := script.Echo(want).WithStdout(buf).Stdout()
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWithEnv_UnsetsAllEnvVarsGivenEmptySlice(t *testing.T) {
	t.Parallel()
	p := script.NewPipe().WithEnv([]string{"ENV1=test1"}).Exec("sh -c 'echo ENV1=$ENV1'")
	want := "ENV1=test1\n"
	got, err := p.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
	got, err = p.Echo("").WithEnv([]string{}).Exec("sh -c 'echo ENV1=$ENV1'").String()
	if err != nil {
		t.Fatal(err)
	}
	want = "ENV1=\n"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestWithEnv_SetsGivenVariablesForSubsequentExec(t *testing.T) {
	t.Parallel()
	env := []string{"ENV1=test1", "ENV2=test2"}
	got, err := script.NewPipe().WithEnv(env).Exec("sh -c 'echo ENV1=$ENV1 ENV2=$ENV2'").String()
	if err != nil {
		t.Fatal(err)
	}
	want := "ENV1=test1 ENV2=test2\n"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestErrorReturnsErrorSetByPreviousPipeStage(t *testing.T) {
	t.Parallel()
	p := script.File("testdata/nonexistent.txt")
	if p.Error() == nil {
		t.Error("want error status reading nonexistent file, but got nil")
	}
}

func TestSetError_SetsSuppliedErrorOnPipe(t *testing.T) {
	t.Parallel()
	p := script.NewPipe()
	e := errors.New("fake error")
	p.SetError(e)
	if p.Error() != e {
		t.Errorf("want %v when setting pipe error, got %v", e, p.Error())
	}
}

func TestExitStatus_CorrectlyParsesExitStatusValueFromErrorMessage(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"bogus", 0},
		{"exit status bogus", 0},
		{"exit status 127", 127},
		{"exit status 1", 1},
		{"exit status 0", 0},
		{"exit status 1 followed by junk", 0},
	}
	for _, tc := range tcs {
		p := script.NewPipe()
		p.SetError(errors.New(tc.input))
		got := p.ExitStatus()
		if got != tc.want {
			t.Errorf("input %q: want %d, got %d", tc.input, tc.want, got)
		}
	}
	got := script.NewPipe().ExitStatus()
	if got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

func TestReadProducesCompletePipeContents(t *testing.T) {
	t.Parallel()
	want := []byte("hello")
	p := script.Echo("hello")
	got, err := io.ReadAll(p)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestReadReturnsEOFOnUninitialisedPipe(t *testing.T) {
	t.Parallel()
	p := &script.Pipe{}
	buf := []byte{0} // try to read at least 1 byte
	n, err := p.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("want io.EOF, got %v", err)
	}
	if n > 0 {
		t.Errorf("unexpectedly read %d bytes", n)
	}
}

func TestReadReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	buf := make([]byte, 0)
	_, err := script.NewPipe().WithReader(brokenReader).Read(buf)
	if err == nil {
		t.Fatal(nil)
	}
}

func TestWait_ReturnsErrorPresentOnPipe(t *testing.T) {
	t.Parallel()
	p := script.Echo("a\nb\nc\n").ExecForEach("{{invalid template syntax}}")
	if p.Wait() == nil {
		t.Error("want error, got nil")
	}
}

func TestWait_DoesNotReturnErrorForValidExecution(t *testing.T) {
	t.Parallel()
	p := script.Echo("a\nb\nc\n").ExecForEach("echo \"{{.}}\"")
	if err := p.Wait(); err != nil {
		t.Fatal(err)
	}
}

var base64Cases = []struct {
	name    string
	decoded string
	encoded string
}{
	{
		name:    "empty string",
		decoded: "",
		encoded: "",
	},
	{
		name:    "single line string",
		decoded: "hello world",
		encoded: "aGVsbG8gd29ybGQ=",
	},
	{
		name:    "multi line string",
		decoded: "hello\nthere\nworld\n",
		encoded: "aGVsbG8KdGhlcmUKd29ybGQK",
	},
}

func TestEncodeBase64_CorrectlyEncodes(t *testing.T) {
	t.Parallel()
	for _, tc := range base64Cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := script.Echo(tc.decoded).EncodeBase64().String()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.encoded {
				t.Logf("input %q incorrectly encoded:", tc.decoded)
				t.Error(cmp.Diff(tc.encoded, got))
			}
		})
	}
}

func TestDecodeBase64_CorrectlyDecodes(t *testing.T) {
	t.Parallel()
	for _, tc := range base64Cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := script.Echo(tc.encoded).DecodeBase64().String()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.decoded {
				t.Logf("input %q incorrectly decoded:", tc.encoded)
				t.Error(cmp.Diff(tc.decoded, got))
			}
		})
	}
}

func TestEncodeBase64_FollowedByDecodeRecoversOriginal(t *testing.T) {
	t.Parallel()
	for _, tc := range base64Cases {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := script.Echo(tc.decoded).EncodeBase64().DecodeBase64().String()
			if err != nil {
				t.Fatal(err)
			}
			if decoded != tc.decoded {
				t.Error("encode-decode round trip failed:", cmp.Diff(tc.decoded, decoded))
			}
			encoded, err := script.Echo(tc.encoded).DecodeBase64().EncodeBase64().String()
			if err != nil {
				t.Fatal(err)
			}
			if encoded != tc.encoded {
				t.Error("decode-encode round trip failed:", cmp.Diff(tc.encoded, encoded))
			}
		})
	}
}

func TestDecodeBase64_CorrectlyDecodesInputToBytes(t *testing.T) {
	t.Parallel()
	input := "CAAAEA=="
	got, err := script.Echo(input).DecodeBase64().Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{8, 0, 0, 16}
	if !bytes.Equal(want, got) {
		t.Logf("input %#v incorrectly decoded:", input)
		t.Error(cmp.Diff(want, got))
	}
}

func TestEncodeBase64_CorrectlyEncodesInputBytes(t *testing.T) {
	t.Parallel()
	input := []byte{8, 0, 0, 16}
	reader := bytes.NewReader(input)
	want := "CAAAEA=="
	got, err := script.NewPipe().WithReader(reader).EncodeBase64().String()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Logf("input %#v incorrectly encoded:", input)
		t.Error(cmp.Diff(want, got))
	}
}

func TestWithStdErr_IsConcurrencySafeAfterExec(t *testing.T) {
	t.Parallel()
	err := script.Exec("echo").WithStderr(nil).Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func TestHash_OutputsCorrectHash(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name, input, want string
		hasher            hash.Hash
	}{
		{
			name:   "for no data",
			input:  "",
			hasher: sha256.New(),
			want:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:   "for short string with SHA 256 hasher",
			input:  "hello, world",
			hasher: sha256.New(),
			want:   "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b",
		},
		{
			name:   "for short string with SHA 512 hasher",
			input:  "hello, world",
			hasher: sha512.New(),
			want:   "8710339dcb6814d0d9d2290ef422285c9322b7163951f9a0ca8f883d3305286f44139aa374848e4174f5aada663027e4548637b6d19894aec4fb6c46a139fbf9",
		},
		{
			name:   "for string containing newline with SHA 256 hasher",
			input:  "The tao that can be told\nis not the eternal Tao",
			hasher: sha256.New(),
			want:   "788542cb92d37f67e187992bdb402fdfb68228a1802947f74c6576e04790a688",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := script.Echo(tc.input).Hash(tc.hasher)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestHashSums_OutputsCorrectHashForEachSpecifiedFile(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		testFileName string
		hasher       hash.Hash
		want         string
	}{
		// To get the checksum run: openssl dgst -sha256 <file_name>
		{
			testFileName: "testdata/hashSum.input.txt",
			hasher:       sha256.New(),
			want:         "1870478d23b0b4db37735d917f4f0ff9393dd3e52d8b0efa852ab85536ddad8e\n",
		},
		{
			testFileName: "testdata/hashSum.input.txt",
			hasher:       sha512.New(),
			want:         "3543bd0d68129e860598ccabcee1beb6bb90d91105cea74a8e555588634ec6f6d6d02033139972da2dc4929b1fb61bd24c91c8e82054e9ae865cf7f70909be8c\n",
		},
		{
			testFileName: "testdata/hello.txt",
			hasher:       sha256.New(),
			want:         "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9\n",
		},
		{
			testFileName: "testdata/multiple_files",
			hasher:       sha256.New(),
			want:         "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\ne3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855\n",
		},
	}
	for _, tc := range tcs {
		got, err := script.ListFiles(tc.testFileName).HashSums(tc.hasher).String()
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%q: want %q, got %q", tc.testFileName, tc.want, got)
		}
	}
}

func TestHash_ReturnsErrorGivenReadErrorOnPipe(t *testing.T) {
	t.Parallel()
	brokenReader := iotest.ErrReader(errors.New("oh no"))
	_, err := script.NewPipe().WithReader(brokenReader).Hash(sha256.New())
	if err == nil {
		t.Fatal(nil)
	}
}

func TestHashSums_OutputsEmptyStringForFileThatCannotBeHashed(t *testing.T) {
	got, err := script.Echo("file_does_not_exist.txt").HashSums(sha256.New()).String()
	if err != nil {
		t.Fatal(err)
	}
	want := ""
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func ExampleArgs() {
	script.Args().Stdout()
	// prints command-line arguments
}

func ExampleDo() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		log.Println(err)
		return
	}
	script.Do(req).Stdout()
	// Output:
	// some data
}

func ExampleEcho() {
	script.Echo("Hello, world!").Stdout()
	// Output:
	// Hello, world!
}

func ExampleExec_exit_status_zero() {
	p := script.Exec("echo")
	p.Wait()
	fmt.Println(p.ExitStatus())
	// Output:
	// 0
}

func ExampleExec_exit_status_not_zero() {
	p := script.Exec("false")
	p.Wait()
	fmt.Println(p.ExitStatus())
	// Output:
	// 1
}

func ExampleFile() {
	script.File("testdata/hello.txt").Stdout()
	// Output:
	// hello world
}

func ExampleGet() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "some data")
	}))
	defer ts.Close()
	script.Get(ts.URL).Stdout()
	// Output:
	// some data
}

func ExampleIfExists_true() {
	script.IfExists("./testdata/hello.txt").Echo("found it").Stdout()
	// Output:
	// found it
}

func ExampleIfExists_false() {
	script.IfExists("doesntexist").Echo("found it").Stdout()
	// Output:
	//
}

func ExamplePipe_Bytes() {
	data, err := script.Echo("hello").Bytes()
	if err != nil {
		panic(err)
	}
	fmt.Println(data)
	// Output:
	// [104 101 108 108 111]
}

func ExamplePipe_Column() {
	input := []string{
		"PID   TT  STAT      TIME COMMAND",
		"  1   ??  Ss   873:17.62 /sbin/launchd",
		" 50   ??  Ss    13:18.13 /usr/libexec/UserEventAgent (System)",
		" 51   ??  Ss    22:56.75 /usr/sbin/syslogd",
	}
	script.Slice(input).Column(1).Stdout()
	// Output:
	// PID
	// 1
	// 50
	// 51
}

func ExamplePipe_Concat() {
	input := []string{
		"testdata/test.txt",
		"testdata/doesntexist.txt",
		"testdata/hello.txt",
	}
	script.Slice(input).Concat().Stdout()
	// Output:
	// This is the first line in the file.
	// Hello, world.
	// This is another line in the file.
	// hello world
}

func ExamplePipe_CountLines() {
	n, err := script.Echo("a\nb\nc\n").CountLines()
	if err != nil {
		panic(err)
	}
	fmt.Println(n)
	// Output:
	// 3
}

func ExamplePipe_DecodeBase64() {
	script.Echo("SGVsbG8sIHdvcmxkIQ==").DecodeBase64().Stdout()
	// Output:
	// Hello, world!
}

func ExamplePipe_Do() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "You said: %s", data)
	}))
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL, strings.NewReader("hello"))
	if err != nil {
		log.Println(err)
		return
	}
	script.NewPipe().Do(req).Stdout()
	// Output:
	// You said: hello
}

func ExamplePipe_EachLine() {
	script.File("testdata/test.txt").EachLine(func(line string, out *strings.Builder) {
		out.WriteString("> " + line + "\n")
	}).Stdout()
	// Output:
	// > This is the first line in the file.
	// > Hello, world.
	// > This is another line in the file.
}

func ExamplePipe_Echo() {
	script.NewPipe().Echo("Hello, world!").Stdout()
	// Output:
	// Hello, world!
}

func ExamplePipe_EncodeBase64() {
	script.Echo("Hello, world!").EncodeBase64().Stdout()
	// Output:
	// SGVsbG8sIHdvcmxkIQ==
}

func ExamplePipe_ExitStatus() {
	p := script.Exec("echo")
	fmt.Println(p.ExitStatus())
	// Output:
	// 0
}

func ExamplePipe_First() {
	script.Echo("a\nb\nc\n").First(2).Stdout()
	// Output:
	// a
	// b
}

func ExamplePipe_Filter() {
	script.Echo("hello world").Filter(func(r io.Reader, w io.Writer) error {
		n, err := io.Copy(w, r)
		fmt.Fprintf(w, "\nfiltered %d bytes\n", n)
		return err
	}).Stdout()
	// Output:
	// hello world
	// filtered 11 bytes
}

func ExamplePipe_FilterScan() {
	script.Echo("a\nb\nc").FilterScan(func(line string, w io.Writer) {
		fmt.Fprintf(w, "scanned line: %q\n", line)
	}).Stdout()
	// Output:
	// scanned line: "a"
	// scanned line: "b"
	// scanned line: "c"
}

func ExamplePipe_FilterLine_user() {
	script.Echo("a\nb\nc").FilterLine(func(line string) string {
		return "> " + line
	}).Stdout()
	// Output:
	// > a
	// > b
	// > c
}

func ExamplePipe_FilterLine_stdlib() {
	script.Echo("a\nb\nc").FilterLine(strings.ToUpper).Stdout()
	// Output:
	// A
	// B
	// C
}

func ExamplePipe_Freq() {
	input := strings.Join([]string{
		"apple",
		"orange",
		"banana",
		"banana",
		"apple",
		"orange",
		"kumquat",
		"apple",
		"orange",
		"apple",
		"banana",
		"banana",
		"apple",
		"apple",
		"orange",
		"apple",
		"apple",
		"apple",
		"apple",
	}, "\n")
	script.Echo(input).Freq().Stdout()
	// Output:
	// 10 apple
	//  4 banana
	//  4 orange
	//  1 kumquat
}

func ExamplePipe_Get() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "You said: %s", data)
	}))
	defer ts.Close()
	script.Echo("hello").Get(ts.URL).Stdout()
	// Output:
	// You said: hello
}

func ExamplePipe_Hash() {
	sum, err := script.Echo("hello world").Hash(sha512.New())
	if err != nil {
		panic(err)
	}
	fmt.Println(sum)
	// Output:
	// 309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f
}

func ExamplePipe_HashSums() {
	script.ListFiles("testdata/multiple_files").HashSums(sha256.New()).Stdout()
	// Output:
	// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	// e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
}

func ExamplePipe_Join() {
	script.Echo("hello\nworld\n").Join().Stdout()
	// Output:
	// hello world
}

func ExamplePipe_JQ() {
	kernel := "Darwin"
	arch := "x86_64"
	query := fmt.Sprintf(".assets[] | select(.name | endswith(\"%s_%s.tar.gz\")).browser_download_url", kernel, arch)
	script.File("testdata/releases.json").JQ(query).Stdout()
	// Output:
	// "https://github.com/mbarley333/blackjack/releases/download/v0.3.3/blackjack_0.3.3_Darwin_x86_64.tar.gz"
}

func ExamplePipe_Last() {
	script.Echo("a\nb\nc\n").Last(2).Stdout()
	// Output:
	// b
	// c
}

func ExamplePipe_Match() {
	script.Echo("a\nb\nc\n").Match("b").Stdout()
	// Output:
	// b
}

func ExamplePipe_MatchRegexp() {
	re := regexp.MustCompile("w.*d")
	script.Echo("hello\nworld\n").MatchRegexp(re).Stdout()
	// Output:
	// world
}

func ExamplePipe_Post() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(w, "You said: %s", data)
	}))
	defer ts.Close()
	script.Echo("hello").Post(ts.URL).Stdout()
	// Output:
	// You said: hello
}

func ExamplePipe_Read() {
	buf := make([]byte, 12)
	n, err := script.Echo("hello world\n").Read(buf)
	if err != nil {
		panic(err)
	}
	fmt.Println(n)
	// Output:
	// 12
}

func ExamplePipe_Reject() {
	script.Echo("a\nb\nc\n").Reject("b").Stdout()
	// Output:
	// a
	// c
}

func ExamplePipe_RejectRegexp() {
	re := regexp.MustCompile("w.*d")
	script.Echo("hello\nworld\n").RejectRegexp(re).Stdout()
	// Output:
	// hello
}

func ExamplePipe_Replace() {
	script.Echo("a\nb\nc\n").Replace("b", "replacement").Stdout()
	// Output:
	// a
	// replacement
	// c
}

func ExamplePipe_ReplaceRegexp() {
	re := regexp.MustCompile("w.*d")
	script.Echo("hello\nworld\n").ReplaceRegexp(re, "replacement").Stdout()
	// Output:
	// hello
	// replacement
}

func ExamplePipe_SHA256Sum() {
	sum, err := script.Echo("hello world").SHA256Sum()
	if err != nil {
		panic(err)
	}
	fmt.Println(sum)
	// Output:
	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
}

func ExamplePipe_SHA256Sums() {
	script.Echo("testdata/test.txt").SHA256Sums().Stdout()
	// Output:
	// a562c9c95e2ff3403e7ffcd8508c6b54d47d5f251387758d3e63dbaaa8296341
}

func ExamplePipe_Slice() {
	s, err := script.Echo("a\nb\nc\n").Slice()
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	// Output:
	// [a b c]
}

func ExamplePipe_Stdout() {
	n, err := script.Echo("a\nb\nc\n").Stdout()
	if err != nil {
		panic(err)
	}
	fmt.Println(n)
	// Output:
	// a
	// b
	// c
	// 6
}

func ExamplePipe_String() {
	s, err := script.Echo("hello\nworld").String()
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	// Output:
	// hello
	// world
}

func ExamplePipe_Tee_stdout() {
	s, err := script.Echo("hello\n").Tee().String()
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	// Output:
	// hello
	// hello
}

func ExamplePipe_Tee_writers() {
	buf1, buf2 := new(bytes.Buffer), new(bytes.Buffer)
	s, err := script.Echo("hello\n").Tee(buf1, buf2).String()
	if err != nil {
		panic(err)
	}
	fmt.Print(s)
	fmt.Print(buf1.String())
	fmt.Print(buf2.String())
	// Output:
	// hello
	// hello
	// hello
}

func ExamplePipe_WithStderr() {
	buf := new(bytes.Buffer)
	script.NewPipe().WithStderr(buf).Exec("go").Wait()
	fmt.Println(strings.Contains(buf.String(), "Usage"))
	// Output:
	// true
}

func ExampleSlice() {
	input := []string{"1", "2", "3"}
	script.Slice(input).Stdout()
	// Output:
	// 1
	// 2
	// 3
}

// A string containing a line longer than bufio.MaxScanTokenSize, for testing
// methods that buffer input. We want to make sure they don't throw
// "bufio.Scanner: token too long" errors.
var longLine = strings.Repeat("super long line ", 4096) + "\nlast line\n"
