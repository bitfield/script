[![Go Reference](https://pkg.go.dev/badge/github.com/bitfield/script.svg)](https://pkg.go.dev/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)[![CircleCI](https://circleci.com/gh/bitfield/script.svg?style=svg)](https://circleci.com/gh/bitfield/script)

```go
import "github.com/bitfield/script"
```

[!['Excited gopher](img/magic.png)](https://bitfieldconsulting.com/golang/scripting)

# What is `script`?

`script` is a Go library for doing the kind of tasks that shell scripts are good at: reading files, executing subprocesses, counting lines, matching strings, and so on.

Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

Shell scripts often compose a sequence of operations on a stream of data (a _pipeline_). This is how `script` works, too.

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

# Real use cases

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

Let's try it out with some [sample data](examples/visitors/access.log):

**`cd examples/visitors`**\
**`go run main.go <access.log`**

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
| [`EachLine`](https://pkg.go.dev/github.com/bitfield/script#Pipe.EachLine) | user-supplied function |
| [`Echo`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Echo) | all input replaced by given string |
| [`Exec`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Exec) | filtered through external command |
| [`ExecForEach`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ExecForEach) | execute given command template for each line of input |
| [`First`](https://pkg.go.dev/github.com/bitfield/script#Pipe.First) | first N lines |
| [`Freq`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Freq) | frequency count of unique input lines, most frequent first |
| [`Join`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Join) | replace all newlines with spaces |
| [`Last`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Last) | last N lines |
| [`Match`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Match) | matching lines |
| [`MatchRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.MatchRegexp) | matching lines |
| [`Reject`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Reject) | non-matching lines |
| [`RejectRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.RejectRegexp) | non-matching lines |
| [`Replace`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Replace) | matching lines replaced with string |
| [`ReplaceRegexp`](https://pkg.go.dev/github.com/bitfield/script#Pipe.ReplaceRegexp) | matching lines replaced with string |
| [`SHA256Sums`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sums) | SHA-256 hashes of each listed file |

## Sinks

Sinks are methods that return some data from a pipe, ending the pipeline and extracting its full contents in a specified way:

| Sink | Destination | Results |
| ---- | ----------- | ------- |
| [`AppendFile`](https://pkg.go.dev/github.com/bitfield/script#Pipe.AppendFile) | appended to file, creating if it exists | bytes written, error |
| [`Bytes`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Bytes) | | data as `[]byte`, error
| [`CountLines`](https://pkg.go.dev/github.com/bitfield/script#Pipe.CountLines) | |number of lines, error  |
| [`Read`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Read) | given `[]byte` | bytes read, error  |
| [`SHA256Sum`](https://pkg.go.dev/github.com/bitfield/script#Pipe.SHA256Sum) | | SHA-256 hash, error  |
| [`Slice`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Slice) | | data as `[]string`, error  |
| [`Stdout`](https://pkg.go.dev/github.com/bitfield/script#Pipe.Stdout) | standard output | bytes written, error  |
| [`String`](https://pkg.go.dev/github.com/bitfield/script#Pipe.String) | | data as `string`, error  |
| [`WriteFile`](https://pkg.go.dev/github.com/bitfield/script#Pipe.WriteFile) | specified file, truncating if it exists | bytes written, error  |

## Examples

Since `script` is designed to help you write system administration programs, a few simple examples of such programs are included in the [examples](examples/) directory.

# Contributing

See the [contributor's guide](CONTRIBUTING.md) for some helpful tips if you'd like to contribute to the `script` project.

# Links

- [Scripting with Go](https://bitfieldconsulting.com/golang/scripting)
- [Code Club: Script](https://www.youtube.com/watch?v=6S5EqzVwpEg)
- [Bitfield Consulting](https://bitfieldconsulting.com/)
- [Go books by John Arundel](https://bitfieldconsulting.com/books)

<small>Gopher image by [MariaLetta](https://github.com/MariaLetta/free-gophers-pack)</small>
