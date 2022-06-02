[![Go Reference](https://pkg.go.dev/badge/github.com/bitfield/script.svg)](https://pkg.go.dev/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)[![CircleCI](https://circleci.com/gh/bitfield/script.svg?style=svg)](https://circleci.com/gh/bitfield/script)

```go
import "github.com/bitfield/script"
```

[![Magical gopher logo](img/magic.png)](https://bitfieldconsulting.com/golang/scripting)

# What is `script`?

`script` is a Go library for doing the kind of tasks that shell scripts are good at: reading files, executing subprocesses, counting lines, matching strings, and so on.

Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

Shell scripts often compose a sequence of operations on a stream of data (a _pipeline_). This is how `script` works, too.

> *This is one absolutely superb API design. Taking inspiration from shell pipes and turning it into a Go library with syntax this clean is really impressive.*\
> —[Simon Willison](https://news.ycombinator.com/item?id=30649524)

Read more: [Scripting with Go](https://bitfieldconsulting.com/golang/scripting)

# Quick start: Unix equivalents

If you're already familiar with shell scripting and the Unix toolset, here is a rough guide to the equivalent `script` operation for each listed Unix command.

| Unix / shell       | `script` equivalent |
| ------------------ | ------------------- |
| (any program name) | [`Exec()`](https://pkg.go.dev/github.com/bitfield/script#Exec) |
| `[ -f FILE ]`      | [`IfExists()`](https://pkg.go.dev/github.com/bitfield/script#IfExists) |
| `>`                | [`WriteFile()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.WriteFile) |
| `>>`               | [`AppendFile()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.AppendFile) |
| `$*`               | [`Args()`](https://pkg.go.dev/github.com/bitfield/script#Args) |
| `basename`         | [`Basename()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Basename) |
| `cat`              | [`File()`](https://pkg.go.dev/github.com/bitfield/script#File) / [`Concat()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Concat) |
| `cut`              | [`Column()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Column) |
| `dirname`          | [`Dirname()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Dirname) |
| `echo`             | [`Echo()`](https://pkg.go.dev/github.com/bitfield/script#Echo) |
| `grep`             | [`Match()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Match) / [`MatchRegexp()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.MatchRegexp) |
| `grep -v`          | [`Reject()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Reject) / [`RejectRegexp()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.RejectRegexp) |
| `head`             | [`First()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.First) |
| `find -type f`     | [`FindFiles`](https://pkg.go.dev/github.com/bitfield/script#FindFiles) |
| `jq`     | [`JQ`](https://pkg.go.dev/github.com/bitfield/script#Pipe.JQ) |
| `ls`               | [`ListFiles()`](https://pkg.go.dev/github.com/bitfield/script#ListFiles) |
| `sed`              | [`Replace()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Replace) / [`ReplaceRegexp()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ReplaceRegexp) |
| `sha256sum`        | [`SHA256Sum()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sum) / [`SHA256Sums()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sums) |
| `tail`             | [`Last()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Last) |
| `uniq -c`          | [`Freq()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Freq) |
| `wc -l`            | [`CountLines()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.CountLines) |
| `xargs`            | [`ExecForEach()`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ExecForEach) |

# Some examples

Let's see some simple examples. Suppose you want to read the contents of a file as a string:

```go
contents, err := script.File("test.txt").String()
```

That looks straightforward enough, but suppose you now want to count the lines in that file.

```go
numLines, err := script.File("test.txt").CountLines()
```

For something a bit more challenging, let's try counting the number of lines in the file that match the string "Error":

```go
numErrors, err := script.File("test.txt").Match("Error").CountLines()
```

But what if, instead of reading a specific file, we want to simply pipe input into this program, and have it output only matching lines (like `grep`)?

```go
script.Stdin().Match("Error").Stdout()
```

Just for fun, let's filter all the results through some arbitrary Go function:

```go
script.Stdin().Match("Error").FilterLine(strings.ToUpper).Stdout()
```

That was almost too easy! So let's pass in a list of files on the command line, and have our program read them all in sequence and output the matching lines:

```go
script.Args().Concat().Match("Error").Stdout()
```

Maybe we're only interested in the first 10 matches. No problem:

```go
script.Args().Concat().Match("Error").First(10).Stdout()
```

What's that? You want to append that output to a file instead of printing it to the terminal? _You've got some attitude, mister_.

```go
script.Args().Concat().Match("Error").First(10).AppendFile("/var/log/errors.txt")
```

If the data is JSON, we can do better than simple string-matching. We can use [JQ](https://stedolan.github.io/jq/) queries:

```go
script.File("commits.json").JQ(".[0] | {message: .commit.message, name: .commit.committer.name}").Stdout()
```

Suppose we want to execute some external program instead of doing the work ourselves. We can do that too:

```go
script.Exec("ping 127.0.0.1").Stdout()
```

But maybe we don't know the arguments yet; we might get them from the user, for example. We'd like to be able to run the external command repeatedly, each time passing it the next line of input. No worries:

```go
script.Args().ExecForEach("ping -c 1 {{.}}").Stdout()
```

If there isn't a built-in operation that does what we want, we can just write our own:

```go
script.Echo("hello world").Filter(func (r io.Reader, w io.Writer) error {
	n, err := io.Copy(w, r)
	fmt.Fprintf(w, "\nfiltered %d bytes\n", n)
	return err
}).Stdout()
// Output:
// hello world
// filtered 11 bytes
```

Notice that the "hello world" appeared before the "filtered n bytes". Filters run concurrently, so the pipeline can start producing output before the input has been fully read.

If we want to scan input line by line, we could do that with a `Filter` function that creates a `bufio.Scanner` on its input, but we don't need to:

```go
script.Echo("a\nb\nc").FilterScan(func(line string, w io.Writer) {
	fmt.Fprintf(w, "scanned line: %q\n", line)
}).Stdout()
// Output:
// scanned line: "a"
// scanned line: "b"
// scanned line: "c"
```

And there's more. Much more. [Read the docs](https://pkg.go.dev/github.com/bitfield/script) for full details, and more examples.

# A realistic use case

Let's use `script` to write a program that system administrators might actually need. One thing I often find myself doing is counting the most frequent visitors to a website over a given period of time. Given an Apache log in the Common Log Format like this:

```
212.205.21.11 - - [30/Jun/2019:17:06:15 +0000] "GET / HTTP/1.1" 200 2028 "https://example.com/ "Mozilla/5.0 (Linux; Android 8.0.0; FIG-LX1 Build/HUAWEIFIG-LX1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.156 Mobile Safari/537.36"
```

we would like to extract the visitor's IP address (the first column in the logfile), and count the number of times this IP address occurs in the file. Finally, we might like to list the top 10 visitors by frequency. In a shell script we might do something like:

```sh
cut -d' ' -f 1 access.log |sort |uniq -c |sort -rn |head
```

There's a lot going on there, and it's pleasing to find that the equivalent `script` program is quite brief:

```go
package main

import (
	"github.com/bitfield/script"
)

func main() {
	script.Stdin().Column(1).Freq().First(10).Stdout()
}
```

Let's try it out with some [sample data](testdata/access.log):

```
16 176.182.2.191
 7 212.205.21.11
 1 190.253.121.1
 1 90.53.111.17
```

# Documentation

See [pkg.go.dev](https://pkg.go.dev/github.com/bitfield/script) for the full documentation, or read on for a summary.

## Sources

These are functions that create a pipe with a given contents:

| Source | Contents |
| -------- | ------------- |
| [`Args`](https://pkg.go.dev/github.com/bitfield/script#Args) | command-line arguments
| [`Echo`](https://pkg.go.dev/github.com/bitfield/script#Echo) | a string
| [`Exec`](https://pkg.go.dev/github.com/bitfield/script#Exec) | command output
| [`File`](https://pkg.go.dev/github.com/bitfield/script#File) | file contents
| [`FindFiles`](https://pkg.go.dev/github.com/bitfield/script#FindFiles) | recursive file listing
| [`IfExists`](https://pkg.go.dev/github.com/bitfield/script#IfExists) | do something only if some file exists
| [`ListFiles`](https://pkg.go.dev/github.com/bitfield/script#ListFiles) | file listing (including wildcards)
| [`Slice`](https://pkg.go.dev/github.com/bitfield/script#Slice) | slice elements, one per line
| [`Stdin`](https://pkg.go.dev/github.com/bitfield/script#Stdin) | standard input

## Filters

Filters are methods on an existing pipe that also return a pipe, allowing you to chain filters indefinitely. The filters modify each line of their input according to the following rules:

| Filter | Results |
| -------- | ------------- |
| [`Basename`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Basename) | removes leading path components from each line, leaving only the filename |
| [`Column`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Column) | Nth column of input |
| [`Concat`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Concat) | contents of multiple files |
| [`Dirname`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Dirname) | removes filename from each line, leaving only leading path components |
| [`Echo`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Echo) | all input replaced by given string |
| [`Exec`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Exec) | filtered through external command |
| [`ExecForEach`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ExecForEach) | execute given command template for each line of input |
| [`Filter`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Filter) | user-supplied function filtering a reader to a writer |
| [`FilterLine`](https://pkg.go.dev/github.com/bitfield/script#Pipe.FilterLine) | user-supplied function filtering each line to a string|
| [`FilterScan`](https://pkg.go.dev/github.com/bitfield/script#Pipe.FilterScan) | user-supplied function filtering each line to a writer |
| [`First`](https://pkg.go.dev/github.com/bitfield/script#Pipe.First) | first N lines of input |
| [`Freq`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Freq) | frequency count of unique input lines, most frequent first |
| [`Join`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Join) | replace all newlines with spaces |
| [`JQ`](https://pkg.go.dev/github.com/bitfield/script#Pipe.JQ) | result of `jq` query |
| [`Last`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Last) | last N lines of input|
| [`Match`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Match) | lines matching given string |
| [`MatchRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.MatchRegexp) | lines matching given regexp |
| [`Reject`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Reject) | lines not matching given string |
| [`RejectRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.RejectRegexp) | lines not matching given regexp |
| [`Replace`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Replace) | matching text replaced with given string |
| [`ReplaceRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ReplaceRegexp) | matching text replaced with given string |
| [`SHA256Sums`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sums) | SHA-256 hashes of each listed file |

Note that filters run concurrently, rather than producing nothing until each stage has fully read its input. This is convenient for executing long-running comands, for example. If you do need to wait for the pipeline to complete, call `Wait`.

## Sinks

Sinks are methods that return some data from a pipe, ending the pipeline and extracting its full contents in a specified way:

| Sink | Destination | Results |
| ---- | ----------- | ------- |
| [`AppendFile`](https://pkg.go.dev/github.com/bitfield/script#Pipe.AppendFile) | appended to file, creating if it doesn't exist | bytes written, error |
| [`Bytes`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Bytes) | | data as `[]byte`, error
| [`CountLines`](https://pkg.go.dev/github.com/bitfield/script#Pipe.CountLines) | |number of lines, error  |
| [`Read`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Read) | given `[]byte` | bytes read, error  |
| [`SHA256Sum`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sum) | | SHA-256 hash, error  |
| [`Slice`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Slice) | | data as `[]string`, error  |
| [`Stdout`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Stdout) | standard output | bytes written, error  |
| [`String`](https://pkg.go.dev/github.com/bitfield/script#Pipe.String) | | data as `string`, error  |
| [`Wait`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Wait) | | none  |
| [`WriteFile`](https://pkg.go.dev/github.com/bitfield/script#Pipe.WriteFile) | specified file, truncating if it exists | bytes written, error  |

# What's new

| Version | New |
| ----------- | ------- |
| v0.20.0 | [`JQ`](https://pkg.go.dev/github.com/bitfield/script#Pipe.JQ) |

# Contributing

See the [contributor's guide](CONTRIBUTING.md) for some helpful tips if you'd like to contribute to the `script` project.

# Links

- [Scripting with Go](https://bitfieldconsulting.com/golang/scripting)
- [Code Club: Script](https://www.youtube.com/watch?v=6S5EqzVwpEg)
- [Bitfield Consulting](https://bitfieldconsulting.com/)
- [Go books by John Arundel](https://bitfieldconsulting.com/books)

<small>Gopher image by [MariaLetta](https://github.com/MariaLetta/free-gophers-pack)</small>
