package script_test

import (
	"testing"

	"github.com/bitfield/script"
)

func TestDirnameReturnsExpectedResultsOnPlatformsWithBackslashPathSeparator(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		path string
		want string
	}{
		{`C:\`, "C:\\\n"},
		{`C:\a\b`, "C:\\a\n"},
		{`C:a\b`, "C:a\n"},
		{`\\host\share`, "\\\\host\\share\n"},
		{`\\host\share\a\b`, "\\\\host\\share\\a\n"},
		{`C:\Program Files\PHP\a`, "C:\\Program Files\\PHP\n"},
	}
	for _, tc := range testCases {
		got, err := script.Echo(tc.path).Dirname().String()
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != got {
			t.Errorf("%q: want %q, got %q", tc.path, tc.want, got)
		}
	}
}

func ExampleFindFiles() {
	script.FindFiles("testdata/multiple_files_with_subdirectory").Stdout()
	// Output:
	// testdata\multiple_files_with_subdirectory\1.txt
	// testdata\multiple_files_with_subdirectory\2.txt
	// testdata\multiple_files_with_subdirectory\3.tar.zip
	// testdata\multiple_files_with_subdirectory\dir\.hidden
	// testdata\multiple_files_with_subdirectory\dir\1.txt
	// testdata\multiple_files_with_subdirectory\dir\2.txt
}

func ExampleListFiles() {
	script.ListFiles("testdata/multiple_files_with_subdirectory").Stdout()
	// Output:
	// testdata\multiple_files_with_subdirectory\1.txt
	// testdata\multiple_files_with_subdirectory\2.txt
	// testdata\multiple_files_with_subdirectory\3.tar.zip
	// testdata\multiple_files_with_subdirectory\dir
}

func ExamplePipe_Basename() {
	input := []string{
		"",
		"/",
		"/root",
		"/tmp/example.php",
		"/var/tmp/",
		"./src/filters",
		"C:\\Program Files",
	}
	script.Slice(input).Basename().Stdout()
	// Output:
	// .
	// \
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
	// \
	// \
	// \tmp
	// \var
	// ./src
	// C:\
}
