[![GoDoc](https://godoc.org/github.com/bitfield/script?status.png)](http://godoc.org/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)[![CircleCI](https://circleci.com/gh/bitfield/script.svg?style=svg)](https://circleci.com/gh/bitfield/script)

# What is `script`?

`script` is a Go library for doing the kind of tasks that shell scripts are good at: reading files, executing subprocesses, counting lines, matching strings, and so on.

Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

Shell scripts often compose a sequence of operations on a stream of data (a _pipeline_). This is how `script` works, too.

# How do I import it?

```go
import github.com/bitfield/script
```

# What can I do with it?

Let's see a simple example. Suppose you want to read the contents of a file as a string:

```go
contents, err := script.File("test.txt").String()
```

That looks straightforward enough, but suppose you now want to count the lines in that file.

```go
numLines, err := script.File("test.txt").CountLines()
```

For something a bit more challenging, let's try counting the number of lines in the file which match the string "Error":

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

# Want some help with Go—or anything else?

Not content with maintaining this library, John Arundel, of [Bitfield Consulting](https://bitfieldconsulting.com), is a highly experienced Go trainer and mentor who can teach you Go from scratch, take you beyond the basics, or even help you reach complete mastery of the Go programming language. See [Learn Go with Bitfield](https://bitfieldconsulting.com/golang) for details, or email go@bitfieldconsulting.com to find out more.

> John's Golang mentoring has helped me build confidence and fill in gaps in my knowledge. It has provided an immeasurable amount of help and guidance, and as a result I'm applying for my dream job as an SRE!<br />
—Melina Boutierou

John is also a [Kubernetes and cloud infrastructure consultant](https://bitfieldconsulting.com/kubernetes) and the author of the book [Cloud Native DevOps with Kubernetes](https://amzn.to/2PEPTjc). If John can help you with your infrastructure or DevOps projects, [get in touch](https://bitfieldconsulting.com/contact)! He'd love to hear from you.

# Table of contents<!-- omit in toc -->
- [What is `script`?](#what-is-script)
- [How do I import it?](#how-do-i-import-it)
- [What can I do with it?](#what-can-i-do-with-it)
- [Want some help with Go—or anything else?](#want-some-help-with-goor-anything-else)
- [How does it work?](#how-does-it-work)
- [Everything is a pipe](#everything-is-a-pipe)
- [What use is a pipe?](#what-use-is-a-pipe)
- [Handling errors](#handling-errors)
- [Getting output](#getting-output)
- [Errors](#errors)
- [Closing pipes](#closing-pipes)
- [Why not just use shell?](#why-not-just-use-shell)
- [A real-world example](#a-real-world-example)
- [Quick start: Unix equivalents](#quick-start-unix-equivalents)
- [Sources, filters, and sinks](#sources-filters-and-sinks)
- [Sources](#sources)
	- [Args](#args)
	- [Echo](#echo)
	- [Exec](#exec)
		- [Exit status](#exit-status)
		- [Error output](#error-output)
	- [File](#file)
	- [IfExists](#ifexists)
	- [FindFiles](#findfiles)
	- [ListFiles](#listfiles)
	- [Slice](#slice)
	- [Stdin](#stdin)
- [Filters](#filters)
	- [Basename](#basename)
	- [Column](#column)
	- [Concat](#concat)
	- [Dirname](#dirname)
	- [EachLine](#eachline)
	- [Exec](#exec-1)
	- [ExecForEach](#execforeach)
	- [First](#first)
	- [Freq](#freq)
	- [Join](#join)
	- [Last](#last)
	- [Match](#match)
	- [MatchRegexp](#matchregexp)
	- [Reject](#reject)
	- [RejectRegexp](#rejectregexp)
	- [Replace](#replace)
	- [ReplaceRegexp](#replaceregexp)
	- [SHA256Sums](#sha256sums)
- [Sinks](#sinks)
	- [AppendFile](#appendfile)
	- [Bytes](#bytes)
	- [CountLines](#countlines)
	- [Read](#read)
	- [SHA256Sum](#sha256sum)
		- [Why not MD5?](#why-not-md5)
	- [Slice](#slice-1)
	- [Stdout](#stdout)
	- [String](#string)
	- [WriteFile](#writefile)
- [Examples](#examples)
- [How can I contribute?](#how-can-i-contribute)

# How does it work?

Those chained function calls look a bit weird. What's going on there?

One of the neat things about the Unix shell, and its many imitators, is the way you can compose operations into a _pipeline_:

```sh
cat test.txt | grep Error | wc -l
```

The output from each stage of the pipeline feeds into the next, and you can think of each stage as a _filter_ which passes on only certain parts of its input to its output.

By comparison, writing shell-like scripts in raw Go is much less convenient, because everything you do returns a different data type, and you must (or at least should) check errors following every operation.

In scripts for system administration we often want to compose different operations like this in a quick and convenient way. If an error occurs somewhere along the pipeline, we would like to check this just once at the end, rather than after every operation.

# Everything is a pipe

The `script` library allows us to do this because everything is a pipe (specifically, a `script.Pipe`). To create a pipe, start with a _source_ like `File()`:

```go
var p script.Pipe
p = script.File("test.txt")
```

You might expect `File()` to return an error if there is a problem opening the file, but it doesn't. We will want to call a chain of methods on the result of `File()`, and it's inconvenient to do that if it also returns an error.

Instead, you can check the error status of the pipe at any time by calling its `Error()` method:

```go
p = script.File("test.txt")
if p.Error() != nil {
    log.Fatalf("oh no: %v", p.Error())
}
```

# What use is a pipe?

Now, what can you do with this pipe? You can call a method on it:

```go
var q script.Pipe
q = p.Match("Error")
```

Note that the result of calling a method on a pipe is another pipe. You can do this in one step, for convenience:

```go
var q script.Pipe
q = script.File("test.txt").Match("Error")
```

# Handling errors

Woah, woah! Just a minute! What if there was an error opening the file in the first place? Won't `Match` blow up if it tries to read from a non-existent file?

No, it won't. As soon as an error status is set on a pipe, all operations on the pipe become no-ops. Any operation which would normally return a new pipe just returns the old pipe unchanged. So you can run as long a pipeline as you want to, and if an error occurs at any stage, nothing will crash, and you can check the error status of the pipe at the end.

(Seasoned Gophers will recognise this as the `errWriter` pattern described by Rob Pike in the blog post [Errors are values](https://blog.golang.org/errors-are-values).)

# Getting output

A pipe is useless if we can't get some output from it. To do this, you can use a _sink_, such as `String()`:

```go
result, err := q.String()
if err != nil {
    log.Fatalf("oh no: %v", err)
}
fmt.Println(result)
```

# Errors

Note that sinks return an error value in addition to the data. This is the same value you would get by calling `p.Error()`. If the pipe had an error in any operation along the pipeline, the pipe's error status will be set, and a sink operation which gets output will return the zero value, plus the error.

```go
numLines, err := script.File("doesnt_exist.txt").CountLines()
fmt.Println(numLines)
// Output: 0
if err != nil {
	    log.Fatal(err)
}
// Output: open doesnt_exist.txt: no such file or directory
```

`CountLines()` is another useful sink, which simply returns the number of lines read from the pipe.

# Closing pipes

If you've dealt with files in Go before, you'll know that you need to _close_ the file once you've finished with it. Otherwise, the program will retain what's called a _file handle_ (the kernel data structure which represents an open file). There is a limit to the total number of open file handles for a given program, and for the system as a whole, so a program which leaks file handles will eventually crash, and will waste resources in the meantime.

Files aren't the only things which need to be closed after reading: so do network connections, HTTP response bodies, and so on.

How does `script` handle this? Simple. The data source associated with a pipe will be automatically closed once it is read completely. Therefore, calling any sink method which reads the pipe to completion (such as `String()`) will close its data source. The only case in which you need to call `Close()` on a pipe is when you don't read from it, or you don't read it to completion.

If the pipe was created from something that doesn't need to be closed, such as a string, then calling `Close()` simply does nothing.

This is implemented using a type called `ReadAutoCloser`, which takes an `io.Reader` and wraps it so that:

1. it is always safe to close (if it's not a closable resource, it will be wrapped in an `ioutil.NopCloser` to make it one), and
2. it is closed automatically once read to completion (specifically, once the `Read()` call on it returns `io.EOF`).

_It is your responsibility to close a pipe if you do not read it to completion_.

# Why not just use shell?

It's a fair question. Shell scripts and one-liners are perfectly adequate for building one-off tasks, initialization scripts, and the kind of 'glue code' that holds the internet together. I speak as someone who's spent at least thirty years doing this for a living. But in many ways they're not ideal for important, non-trivial programs:

* Trying to build portable shell scripts is a nightmare. The exact syntax and options of Unix commands varies from one distribution to another. Although in theory POSIX is a workable common subset of functionality, in practice it's usually precisely the non-POSIX behaviour that you need.

* Shell scripts are hard to test (though test frameworks have been written, and if you're seriously putting mission-critical shell scripts into production, you should be using them, or reconsidering your technology choices).

* Shell scripts don't scale. Because there are very limited facilities for logic and abstraction, and because any successful program tends to grow remorselessly over time, shell scripts can become an unreadable mess of special cases and spaghetti code. We've all seen it, if not, indeed, done it.

* Shell syntax is awkward: quoting, whitespace, and brackets can require a lot of fiddling to get right, and so many characters are magic to the shell (`*`, `?`, `>` and so on) that this can lead to subtle bugs. Scripts can work fine for years until you suddenly encounter a file whose name contains whitespace, and then everything breaks horribly.

* Deploying shell scripts obviously requires at least a (sizable) shell binary in addition to the source code, but it usually also requires an unknown and variable number of extra userland programs (`cut`, `grep`, `head`, and friends). If you're building container images, for example, you effectively need to include a whole Unix distribution with your program, which runs to hundreds of megabytes, and is not at all in the spirit of containers.

To be fair to the shell, this kind of thing is not what it was ever intended for. Shell is an interactive job control tool for launching programs, connecting programs together, and to a limited extent, manipulating text. It's not for building portable, scalable, reliable, and elegant programs. That's what Go is for.

Go has a superb testing framework built right into the standard library. It has a superb standard library, and thousands of high-quality third-party packages for just about any functionality you can imagine. It is compiled, so it's fast, and statically typed, so it's reliable. It's efficient and memory-safe. Go programs can be distributed as a single binary. Go scales to enormous projects (Kubernetes, for example).

The `script` library is implemented entirely in Go, and does not require any userland programs (or any other dependencies) to be present. Thus you can build your `script` program as a container image containing a single (very small) binary, which is quick to build, quick to upload, quick to deploy, quick to run, and economical with resources.

If you've ever struggled to get a shell script containing a simple `if` statement to work (and who hasn't?), then the `script` library is dedicated to you.

# A real-world example

Let's use `script` to write a program which system administrators might actually need. One thing I often find myself doing is counting the most frequent visitors to a website over a given period of time. Given an Apache log in the Common Log Format like this:

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

(Thanks to [Lucas Bremgartner](https://github.com/breml) for suggesting this example. You can find the complete [program](examples/visitors/main.go), along with a sample [logfile](examples/visitors/access.log), in the [`examples/visitors/`](examples/visitors) directory.)

# Quick start: Unix equivalents

If you're already familiar with shell scripting and the Unix toolset, here is a rough guide to the equivalent `script` operation for each listed Unix command.

| Unix / shell       | `script` equivalent                                           |
| ------------------ | ------------------------------------------------------------- |
| (any program name) | [`Exec()`](#exec)                                             |
| `[ -f FILE ]`      | [`IfExists()`](#ifexists)                                     |
| `>`                | [`WriteFile()`](#writefile)                                   |
| `>>`               | [`AppendFile()`](#appendfile)                                 |
| `$*`               | [`Args()`](#args)                                             |
| `basename`         | [`Basename()`](#basename)                                     |
| `cat`              | [`File()`](#file) / [`Concat()`](#concat)                     |
| `cut`              | [`Column()`](#column)                                         |
| `dirname`          | [`Dirname()`](#dirname)                                       |
| `echo`             | [`Echo()`](#echo)                                             |
| `grep`             | [`Match()`](#match) / [`MatchRegexp()`](#matchregexp)         |
| `grep -v`          | [`Reject()`](#reject) / [`RejectRegexp()`](#rejectregexp)     |
| `head`             | [`First()`](#first)                                           |
| `find`             | [`FindFiles`](#findfiles)                                     |
| `ls`               | [`ListFiles()`](#listfiles)                                   |
| `sed`              | [`Replace()`](#replace) / [`ReplaceRegexp()`](#replaceregexp) |
| `sha256sum`        | [`SHA256Sum()`](#sha256Sum) / [`SHA256Sums()`](#sha256sums)   |
| `tail`             | [`Last()`](#last)                                             |
| `uniq -c`          | [`Freq()`](#freq)                                             |
| `wc -l`            | [`CountLines()`](#countlines)                                 |
| `xargs`            | [`ExecForEach()`](#execforeach)                               |

# Sources, filters, and sinks

`script` provides three types of pipe operations: sources, filters, and sinks.

1. _Sources_ create pipes from input in some way (for example, `File()` opens a file).
2. _Filters_ read from a pipe and filter the data in some way (for example `Match()` passes on only lines which contain a given string).
3. _Sinks_ get the output from a pipeline in some useful form (for example `String()` returns the contents of the pipe as a string), along with any error status.

Let's look at the source, filter, and sink options that `script` provides.

# Sources

These are operations which create a pipe.

## Args

`Args()` creates a pipe containing the program's command-line arguments, one per line.

```go
p := script.Args()
output, err := p.String()
fmt.Println(output)
// Output: command-line arguments
```

## Echo

`Echo()` creates a pipe containing a given string:

```go
p := script.Echo("Hello, world!")
output, err := p.String()
fmt.Println(output)
// Output: Hello, world!
```

## Exec

`Exec()` runs a given command and creates a pipe containing its combined output (`stdout` and `stderr`). If there was an error running the command, the pipe's error status will be set.

```go
p := script.Exec("bash -c 'echo hello'")
output, err := p.String()
fmt.Println(output)
// Output: hello
```

Note that `Exec()` can also be used as a filter, in which case the given command will read from the pipe as its standard input.

### Exit status

If the command returns a non-zero exit status, the pipe's error status will be set to the string "exit status X", where X is the integer exit status.

```go
p := script.Exec("ls doesntexist")
output, err := p.String()
fmt.Println(err)
// Output: exit status 1
```

For convenience, you can get this value directly as an integer by calling `ExitStatus()` on the pipe:

```go

p := script.Exec("ls doesntexist")
var exit int = p.ExitStatus()
fmt.Println(exit)
// Output: 1
```

The value of `ExitStatus()` will be zero unless the pipe's error status matches the string "exit status X", where X is a non-zero integer.

### Error output

Even in the event of a non-zero exit status, the command's output will still be available in the pipe. This is often helpful for debugging. However, because `String()` is a no-op if the pipe's error status is set, if you want output you will need to reset the error status before calling `String()`:

```go
p := Exec("man bogus")
p.SetError(nil)
output, err := p.String()
fmt.Println(output)
// Output: No manual entry for bogus
```

## File

`File()` creates a pipe that reads from a file.

```go
p = script.File("test.txt")
output, err := p.String()
fmt.Println(output)
// Output: contents of file
```

## IfExists

`IfExists()` tests whether the specified file exists. If so, the returned pipe will have no error status. If it doesn't exist, the returned pipe will have an appropriate error set.

```go
p = script.IfExists("doesntexist.txt")
output, err := p.String()
fmt.Println(err)
// Output: stat doesntexist.txt: no such file or directory
```

This can be used to create pipes which take some action only if a certain file exists:

```go
script.IfExists("/foo/bar").Exec("/usr/bin/yada")
```

## FindFiles

`FindFiles()` lists all files in a directory and its subdirectories recursively, like Unix [`find -type f`](examples/find/main.go).

```go
script.FindFiles("/tmp").Stdout()
// lists all files in /tmp and its subtrees
```

## ListFiles

`ListFiles()` lists files, like Unix [`ls`](examples/ls/main.go). It creates a pipe containing all files and directories matching the supplied path specification, one per line. This can be the name of a directory (`/path/to/dir`), the name of a file (`/path/to/file`), or a _glob_ (wildcard expression) conforming to the syntax accepted by [filepath.Match()](https://golang.org/pkg/path/filepath/#Match) (`/path/to/*`).

```go
p := script.ListFiles("/tmp/*.php")
files, err := p.String()
if err != nil {
	log.Fatal(err)
}
fmt.Println("found suspicious PHP files in /tmp:")
fmt.Println(files)
```

## Slice

`Slice()` creates a pipe from a slice of strings, one per line.

```go
p := script.Slice([]string{"1", "2", "3"})
output, err := p.String()
fmt.Println(output)
// Output:
// 1
// 2
// 3
```

## Stdin

`Stdin()` creates a pipe which reads from the program's standard input.

```go
p := script.Stdin()
output, err := p.String()
fmt.Println(output)
// Output: [contents of standard input]
```

# Filters

Filters are operations on an existing pipe that also return a pipe, allowing you to chain filters indefinitely.

## Basename

`Basename()` reads a list of filepaths from the pipe, one per line, and removes any leading directory components from each line (so, for example, `/usr/local/bin/foo` would become just `foo`). This is the complement of [Dirname](#dirname).

If a line is empty, `Basename()` will produce a single dot: `.`. Trailing slashes are removed.

Examples:

| Input              | `Basename` output |
| ------------------ | ----------------- |
|                    | `.`               |
| `/`                | `.`               |
| `/root`            | `root`            |
| `/tmp/example.php` | `example.php`     |
| `/var/tmp/`        | `tmp`             |
| `./src/filters`    | `filters`         |
| `C:/Program Files` | `Program Files`   |

## Column

`Column()` reads input tabulated by whitespace, and outputs only the Nth column of each input line (like Unix `cut`). Lines containing less than N columns will be ignored.

For example, given this input:

```
  PID   TT  STAT      TIME COMMAND
    1   ??  Ss   873:17.62 /sbin/launchd
   50   ??  Ss    13:18.13 /usr/libexec/UserEventAgent (System)
   51   ??  Ss    22:56.75 /usr/sbin/syslogd
```

and this program:

```go
script.Stdin().Column(1).Stdout()
```

this will be the output:

```
PID
1
50
51
```

## Concat

`Concat()` reads a list of filenames from the pipe, one per line, and creates a pipe which concatenates the contents of those files. For example, if you have files `a`, `b`, and `c`:

```go
output, err := Echo("a\nb\nc\n").Concat().String()
fmt.Println(output)
// Output: contents of a, followed by contents of b, followed
// by contents of c
```

This makes it convenient to write programs which take a list of input files on the command line, for example:

```go
func main() {
	script.Args().Concat().Stdout()
}
```

The list of files could also come from a file:

```go
// Read all files in filelist.txt
p := File("filelist.txt").Concat()
```

...or from the output of a command:

```go
// Print all config files to the terminal.
p := Exec("ls /var/app/config/").Concat().Stdout()
```

Each input file will be closed once it has been fully read. If any of the files can't be opened or read, `Concat()` will simply skip these and carry on, without setting the pipe's error status. This mimics the behaviour of Unix `cat`.

## Dirname

`Dirname()` reads a list of pathnames from the pipe, one per line, and returns a pipe which contains only the parent directories of each pathname (so, for example, `/usr/local/bin/foo` would become just `/usr/local/bin`). This is the complement of [Basename](#basename).

If a line is empty, `Dirname()` will convert it to a single dot: `.` (this is the behaviour of Unix `dirname` and the Go standard library's `filepath.Dir`).

Trailing slashes are removed, unless `Dirname()` returns the root folder.

Examples:

| Input              | `Dirname` output |
| ------------------ | ---------------- |
|                    | `.`              |
| `/`                | `/`              |
| `/root`            | `/`              |
| `/tmp/example.php` | `/tmp`           |
| `/var/tmp/`        | `/var`           |
| `./src/filters`    | `./src`          |
| `C:/Program Files` | `C:`             |

## EachLine

`EachLine()` lets you create custom filters. You provide a function, and it will be called once for each line of input. If you want to produce output, your function can write to a supplied `strings.Builder`. The return value from EachLine is a pipe containing your output.

```go
p := script.File("test.txt")
q := p.EachLine(func(line string, out *strings.Builder) {
	out.WriteString("> " + line + "\n")
})
output, err := q.String()
fmt.Println(output)
```

## Exec

`Exec()` runs a given command, which will read from the pipe as its standard input, and returns a pipe containing the command's combined output (`stdout` and `stderr`). If there was an error running the command, the pipe's error status will be set.

Apart from connecting the pipe to the command's standard input, the behaviour of an `Exec()` filter is the same as that of an `Exec()` source.

```go
// `cat` copies its standard input to its standard output.
p := script.Echo("hello world").Exec("cat")
output, err := p.String()
fmt.Println(output)
// Output: hello world
```

## ExecForEach

ExecForEach runs the supplied command once for each line of input, and returns a pipe containing the output, like Unix `xargs`.

The command string is interpreted as a Go template, so `{{.}}` will be replaced with the input value, for example.

The first command which results in an error will set the pipe's error status accordingly, and no subsequent commands will be run.

```go
// Execute all PHP files in current directory and print output
script.ListFiles("*.php").ExecForEach("php {{.}}").Stdout()
```

## First

`First()` reads its input and passes on the first N lines of it (like Unix [`head`](examples/head/main.go)):

```go
script.Stdin().First(10).Stdout()
```

## Freq

`Freq()` counts the frequencies of input lines, and outputs only the unique lines in the input, each prefixed with a count of its frequency, in descending order of frequency (that is, most frequent lines first). Lines with the same frequency will be sorted alphabetically. For example, given this input:

```
banana
apple
orange
apple
banana
```

and a program like:

```go
script.Stdin().Freq().Stdout()
```

the output will be:

```
2 apple
2 banana
1 orange
```

This is a common pattern in shell scripts to find the most frequently-occurring lines in a file:

```sh
sort testdata/freq.input.txt |uniq -c |sort -rn
```

`Freq()`'s behaviour is like the combination of Unix `sort`, `uniq -c`, and `sort -rn` used here. You can use `Freq()` in combination with `First()` to get, for example, the ten most common lines in a file:

```go
script.Stdin().Freq().First(10).Stdout()
```

Like `uniq -c`, `Freq()` left-pads its count values if necessary to make them easier to read:

```
10 apple
 4 banana
 2 orange
 1 kumquat
```

## Join

`Join()` reads its input and replaces newlines with spaces, preserving a terminating newline if there is one.

```go
p := script.Echo("hello\nworld\n").Join()
output, err := p.String()
fmt.Println(output)
// Output: hello world\n
```

## Last

`Last()` reads its input and passes on the last N lines of it (like Unix [`tail`](examples/tail/main.go)):

```go
script.Stdin().Last(10).Stdout()
```

## Match

`Match()` returns a pipe containing only the input lines which match the supplied string:

```go
p := script.File("test.txt").Match("Error")
```

## MatchRegexp

`MatchRegexp()` is like `Match()`, but takes a compiled regular expression instead of a string.

```go
p := script.File("test.txt").MatchRegexp(regexp.MustCompile(`E.*r`))
```

## Reject

`Reject()` is the inverse of `Match()`. Its pipe produces only lines which _don't_ contain the given string:

```go
p := script.File("test.txt").Match("Error").Reject("false alarm")
```

## RejectRegexp

`RejectRegexp()` is like `Reject()`, but takes a compiled regular expression instead of a string.

```go
p := script.File("test.txt").Match("Error").RejectRegexp(regexp.MustCompile(`false|bogus`))
```

## Replace

`Replace()` returns a pipe which filters its input by replacing all occurrences of one string with another, like Unix `sed`:

```go
p := script.File("test.txt").Replace("old", "new")
```

## ReplaceRegexp

`ReplaceRegexp()` returns a pipe which filters its input by replacing all matches of a compiled regular expression with a supplied replacement string, like Unix `sed`:

```go
p := script.File("test.txt").ReplaceRegexp(regexp.MustCompile("Gol[a-z]{1}ng"), "Go")
```

## SHA256Sums
`SHA256Sums()` reads a list of file paths from the pipe, one per line, and returns a pipe which contains the SHA-256 checksum of each file.
If there are any errors (for example, non-existent files), the pipe's error status will be set to the first error encountered, but execution will continue.

Examples:

| Input                                                                                                    | `SHA256Sums` output                                                                                                                                                                                            |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `testdata/sha256Sum.input.txt`                                                                           | `1870478d23b0b4db37735d917f4f0ff9393dd3e52d8b0efa852ab85536ddad8e`                                                                                                                                             |
| `testdata/multiple_files/1.txt`<br>`testdata/multiple_files/2.txt`<br>`testdata/multiple_files/3.tar.gz` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`<br>`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`<br>`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

# Sinks

Sinks are operations which return some data from a pipe, ending the pipeline.

## AppendFile

`AppendFile()` is like `WriteFile()`, but appends to the destination file instead of overwriting it. It returns the number of bytes written, or an error:

```go
var wrote int
wrote, err := script.Echo("Got this far!").AppendFile("logfile.txt")
```

## Bytes

`Bytes()` returns the contents of the pipe as a slice of byte, plus an error:

```go
var data []byte
data, err := script.File("test.bin").Bytes()
```

## CountLines

`CountLines()`, as the name suggests, counts lines in its input, and returns the number of lines as an integer, plus an error:

```go
var numLines int
numLines, err := script.File("test.txt").CountLines()
```

## Read

`Read()` behaves just like the standard `Read()` method on any `io.Reader`:

```go
buf := make([]byte, 256)
n, err := r.Read(buf)
```

Because a Pipe is an `io.Reader`, you can use it anywhere you would use a file, network connection, and so on. You can pass it to `ioutil.ReadAll`, `io.Copy`, `json.NewDecoder`, and anything else which takes an `io.Reader`.

Unlike most sinks, `Read()` does not read the whole contents of the pipe (unless the supplied buffer is big enough to hold them).

## SHA256Sum

`SHA256Sum()`, as the name suggests, returns the [SHA256 checksum](https://en.wikipedia.org/wiki/SHA-2) of the file as a hexadecimal number stored in a string, plus an error:
```go
var sha256Sum string
sha256Sum, err := script.File("test.txt").SHA256Sum()
```
### Why not MD5?

[MD5 is insecure](https://en.wikipedia.org/wiki/MD5#Security).

## Slice

`Slice()` returns the contents of the pipe as a slice of strings, one element per line, plus an error. An empty pipe will produce an empty slice. A pipe containing a single empty line (that is, a single `\n` character) will produce a slice of one element which is the empty string.

```go
args, err := script.Args().Slice()
for _, a := range args {
	fmt.Println(a)
}
```

## Stdout

`Stdout()` writes the contents of the pipe to the program's standard output. It returns the number of bytes written, or an error:

```go
p := Echo("hello world")
wrote, err := p.Stdout()
```

In conjunction with `Stdin()`, `Stdout()` is useful for writing programs which filter input. For example, here is a program which simply copies its input to its output, like `cat`:

```go
func main() {
	script.Stdin().Stdout()
}
```

To filter only lines matching a string:

```go
func main() {
	script.Stdin().Match("hello").Stdout()
}
```

## String

`String()` returns the contents of the pipe as a string, plus an error:

```go
contents, err := script.File("test.txt").String()
```

Note that `String()`, like all sinks, consumes the complete output of the pipe, which closes the input reader automatically. Therefore, calling `String()` (or any other sink method) again on the same pipe will return an error:

```go
p := script.File("test.txt")
_, _ = p.String()
_, err := p.String()
fmt.Println(err)
// Output: read test.txt: file already closed
```

## WriteFile

`WriteFile()` writes the contents of the pipe to a named file. It returns the number of bytes written, or an error:

```go
var wrote int
wrote, err := script.File("source.txt").WriteFile("destination.txt")
```

# Examples

Since `script` is designed to help you write system administration programs, a few simple examples of such programs are included in the [examples](examples/) directory:

* [cat](examples/cat/main.go) (copies stdin to stdout)
* [cat 2](examples/cat2/main.go) (takes a list of files on the command line and concatenates their contents to stdout)
* [echo](examples/echo/main.go)
* [execforeach](examples/execforeach/main.go)
* [execute](examples/execute/main.go)
* [grep](examples/grep/main.go)
* [head](examples/head/main.go)
* [least_freq](examples/least_freq/main.go)
* [ls](examples/ls/main.go)
* [sha256sum](examples/sha256sum/main.go)
* [slice](examples/slice/main.go)
* [tail](examples/tail/main.go)
* [visitors](examples/visitors/main.go)

[More examples would be welcome!](https://github.com/bitfield/script/pulls)

If you use `script` for real work (or, for that matter, real play), I'm always very interested to hear about it. Drop me a line to john@bitfieldconsulting.com and tell me how you're using `script` and what you think of it!

# How can I contribute?

See the [contributor's guide](CONTRIBUTING.md) for some helpful tips.
