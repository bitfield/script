package script_test

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bitfield/script"
)

func ExampleArgs() {
	script.Args().Stdout()
	// prints command-line arguments
}

func ExampleEcho() {
	script.Echo("Hello, world!").Stdout()
	// Output:
	// Hello, world!
}

func ExampleExec_ok() {
	script.Exec("echo Hello, world!").Stdout()
	// Output:
	// Hello, world!
}

func ExampleExec_exitstatus() {
	p := script.Exec("ls doesntexist")
	fmt.Println(p.ExitStatus())
	// Output:
	// 1
}

func ExampleExec_errorOutput() {
	p := script.Exec("man bogus")
	p.SetError(nil)
	p.Stdout()
	// Output:
	// No manual entry for bogus
}

func ExampleFile() {
	script.File("testdata/hello.txt").Stdout()
	// Output:
	// hello world
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

func ExamplePipe_ExitStatus() {
	p := script.Exec("ls doesntexist")
	fmt.Println(p.ExitStatus())
	// Output:
	// 1
}

func ExamplePipe_First() {
	script.Echo("a\nb\nc\n").First(2).Stdout()
	// Output:
	// a
	// b
}

func ExamplePipe_Freq() {
	script.File("testdata/freq.input.txt").Freq().Stdout()
	// Output:
	// 10 apple
	//  4 banana
	//  4 orange
	//  1 kumquat
}

func ExamplePipe_Join() {
	script.Echo("hello\nworld\n").Join().Stdout()
	// Output:
	// hello world
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

func ExampleSlice() {
	input := []string{"1", "2", "3"}
	script.Slice(input).Stdout()
	// Output:
	// 1
	// 2
	// 3
}
