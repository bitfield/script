[![GoDoc](https://godoc.org/github.com/bitfield/script?status.png)](http://godoc.org/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)

# script

`script` is a Go library for doing the kind of tasks that shell scripts are good at: reading files, executing subprocesses, counting lines, matching strings, and so on.

Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

## What can I do with it?

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

## Wait, what?

Those chained function calls look a bit weird. What's going on there?

One of the neat things about the Unix shell, and its many imitators, is the way you can compose operations into a _pipeline_:

```sh
cat test.txt | grep Error | wc -l
```

The output from each stage of the pipeline feeds into the next, and you can think of each stage as a _filter_ which passes on only certain parts of its input to its output.

By comparison, writing shell-like scripts in raw Go is much less convenient, because everything you do returns a different data type, and you must (or at least should) check errors following every operation.

In scripts for system administration we often want to compose different operations like this in a quick and convenient way. If an error occurs somewhere along the pipeline, we would like to check this just once at the end, rather than after every operation.

## Everything is a pipe

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

## What use is a pipe?

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

## Handling errors

Woah, woah! Just a minute! What if there was an error opening the file in the first place? Won't `Match` blow up if it tries to read from a non-existent file?

No, it won't. As soon as an error status is set on a pipe, all operations on the pipe become no-ops. Any operation which would normally return a new pipe just returns the old pipe unchanged. So you can run as long a pipeline as you want to, and if an error occurs at any stage, nothing will crash, and you can check the error status of the pipe at the end.

(Seasoned Gophers will recognise this as the `errWriter` pattern described by Rob Pike in the blog post [Errors are values](https://blog.golang.org/errors-are-values).)

## Getting output

A pipe is useless if we can't get some output from it. To do this, you can use a _sink_, such as `String()`:

```go
result, err := q.String()
if err != nil {
    log.Fatalf("oh no: %v", err)
}
fmt.Println(result)
```

## Errors

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

## Closing pipes

If you've dealt with files in Go before, you'll know that you need to _close_ the file once you've finished with it. Otherwise, the program will retain what's called a _file handle_ (the kernel data structure which represents an open file). There is a limit to the total number of open file handles for a given program, and for the system as a whole, so a program which leaks file handles will eventually crash, and will waste resources in the meantime.

Files aren't the only things which need to be closed after reading: so do network connections, HTTP response bodies, and so on.

How does `script` handle this? Simple. The data source associated with a pipe will be automatically closed once it is read completely. Therefore, calling any sink method which reads the pipe to completion (such as `String()`) will close its data source. The only case in which you need to call `Close()` on a pipe is when you don't read from it, or you don't read it to completion.

If the pipe was created from something that doesn't need to be closed, such as a string, then calling `Close()` simply does nothing.

This is implemented using a type called `ReadAutoCloser`, which takes an `io.Reader` and wraps it so that:

1. it is always safe to close (if it's not a closable resource, it will be wrapped in an `ioutil.NopCloser` to make it one), and
2. it is closed automatically once read to completion (specifically, once the `Read()` call on it returns `io.EOF`).

_It is your responsibility to close a pipe if you do not read it to completion_.

## Sources, filters, and sinks

`script` provides three types of pipe operations: sources, filters, and sinks.

1. _Sources_ create pipes from input in some way (for example, `File()` opens a file).
2. _Filters_ read from a pipe and filter the data in some way (for example `Match()` passes on only lines which contain a given string).
3. _Sinks_ get the output from a pipeline in some useful form (for example `String()` returns the contents of the pipe as a string), along with any error status.

Let's look at the source, filter, and sink options that `script` provides.

## Sources

These are operations which create a pipe.

### File

`File()` creates a pipe that reads from a file.

```go
p = script.File("test.txt")
output, err := p.String()
fmt.Println(output)
// Output: contents of file
```

### Echo

`Echo()` creates a pipe containing a given string:

```go
p := script.Echo("Hello, world!")
output, err := p.String()
fmt.Println(output)
// Output: Hello, world!
```

### Exec

`Exec()` runs a given command and creates a pipe containing its combined output (`stdout` and `stderr`). If there was an error running the command, the pipe's error status will be set.

```go
p := script.Exec("echo hello")
output, err := p.String()
fmt.Println(output)
// Output: hello
```

Note that `Exec()` can also be used as a filter, in which case the given command will read from the pipe as its standard input.

#### Exit status

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

#### Error output

Even in the event of a non-zero exit status, the command's output will still be available in the pipe. This is often helpful for debugging. However, because `String()` is a no-op if the pipe's error status is set, if you want output you will need to reset the error status before calling `String()`:

```go
p := Exec("man bogus")
p.SetError(nil)
output, err := p.String()
fmt.Println(output)
// Output: No manual entry for bogus
```

### Stdin

`Stdin()` creates a pipe which reads from the program's standard input.

```go
p := script.Stdin()
output, err := p.String()
fmt.Println(output)
// Output: [contents of standard input]
```

### Args

`Args()` creates a pipe containing the program's command-line arguments, one per line.

```go
p := script.Args()
output, err := p.String()
fmt.Println(output)
// Output: command-line arguments
```

## Filters

Filters are operations on an existing pipe that also return a pipe, allowing you to chain filters indefinitely.

### Match

`Match()` returns a pipe containing only the input lines which match the supplied string:

```go
p := script.File("test.txt").Match("Error")
```

### MatchRegexp

`MatchRegexp()` is like `Match()`, but takes a compiled regular expression instead of a string.

```go
p := script.File("test.txt").MatchRegexp(regexp.MustCompile(`E.*r`))
```

### Reject

`Reject()` is the inverse of `Match()`. Its pipe produces only lines which _don't_ contain the given string:

```go
p := script.File("test.txt").Match("Error").Reject("false alarm")
```

### RejectRegexp

`RejectRegexp()` is like `Reject()`, but takes a compiled regular expression instead of a string.

```go
p := script.File("test.txt").Match("Error").RejectRegexp(regexp.MustCompile(`false|bogus`))
```

### EachLine

`EachLine()` lets you create custom filters. You provide a function, and it will be called once for each line of input. If you want to produce output, your function can write to a supplied `strings.Builder`. The return value from EachLine is a pipe containing your output.

```go
p := script.File("test.txt")
q := p.EachLine(func(line string, out *strings.Builder) {
	out.WriteString("> " + line + "\n")
})
output, err := q.String()
fmt.Println(output)
```

### Exec

`Exec()` runs a given command, which will read from the pipe as its standard input, and returns a pipe containing the command's combined output (`stdout` and `stderr`). If there was an error running the command, the pipe's error status will be set.

Apart from connecting the pipe to the command's standard input, the behaviour of an `Exec()` filter is the same as that of an `Exec()` source.

```go
// `cat` copies its standard input to its standard output.
p := script.Echo("hello world").Exec("cat")
output, err := p.String()
fmt.Println(output)
// Output: hello world
```

### Join

`Join()` reads its input and replaces newlines with spaces, preserving a terminating newline if there is one.

```go
p := script.Echo("hello\nworld\n").Join()
output, err := p.String()
fmt.Println(output)
// Output: hello world\n
```

### Concat

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

Each input file will be closed once it has been fully read.

## Sinks

Sinks are operations which return some data from a pipe, ending the pipeline.

### String

The simplest sink is `String()`, which just returns the contents of the pipe as a string, plus an error:

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

### Bytes

`Bytes()` returns the contents of the pipe as a slice of byte, plus an error:

```go
var data []byte
data, err := script.File("test.bin").Bytes()
```

### CountLines

`CountLines()`, as the name suggests, counts lines in its input, and returns the number of lines as an integer, plus an error:

```go
var numLines int
numLines, err := script.File("test.txt").CountLines()
```

### WriteFile

`WriteFile()` writes the contents of the pipe to a named file. It returns the number of bytes written, or an error:

```go
var wrote int
wrote, err := script.File("source.txt").WriteFile("destination.txt")
```

### AppendFile

`AppendFile()` is like `WriteFile()`, but appends to the destination file instead of overwriting it. It returns the number of bytes written, or an error:

```go
var wrote int
wrote, err := script.Echo("Got this far!").AppendFile("logfile.txt")
```

### Stdout

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

## Writing your own pipe operations

There's nothing to stop you writing your own sources, sinks, or filters (in fact, that would be excellent. Please submit a pull request if you want to add them to the standard operations supplied with `script`.)

### Writing a source

All a pipe source has to do is return a pointer to a `script.Pipe`. To be useful, a pipe needs to have a reader (a data source, such as a file) associated with it.

`Echo()` is a simple example, which just creates a pipe containing a string:

```go
func Echo(s string) *script.Pipe {
	return script.NewPipe().WithReader(strings.NewReader(s))
}
```

Let's break this down:

* We create a `strings.Reader` to be our data source, using `strings.NewReader` on the supplied string.
* We create a new pipe with `NewPipe()`.
* We attach the reader to the pipe with `WithReader()`.

In fact, any `io.Reader` can be the data source for a pipe. Passing it to `WithReader()` will turn it into a `ReadAutoCloser`, which is a wrapper for `io.Reader` that automatically closes the reader once it has been fully read.


Here's an implementation of `File()`, for example:

```go
func File(name string) *script.Pipe {
	p := script.NewPipe()
	f, err := os.Open(name)
	if err != nil {
		return p.WithError(err)
	}
	return p.WithReader(f)
}
```

### Writing a filter

Filters are methods on pipes, that return pipes. For example, here's a simple filter which just reads and rejects all input, returning an empty pipe:

```go
func (p *script.Pipe) RejectEverything() *script.Pipe {
	if p.Error() != nil {
		return &p
	}
	defer p.Close()
	_, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return &p
	}
	return script.Echo("")
}
```

Important things to note here:

* The first thing we do is check the pipe's error status. If this is set, we do nothing, and just return the original pipe.
* We close the source pipe once we successfully read all data from it.
* If an error occurs, we set the pipe's error status, using `p.SetError()`, and return the pipe.

Filters must not log anything, terminate the program, or return anything but `*script.Pipe`.

As you can see from the example, the pipe's reader is available to you as `p.Reader`. You can do anything with that that you can with an `io.Reader`.

If your method modifies the pipe (for example if it can set an error on the pipe), it must take a pointer receiver, as in this example. Otherwise, it can take a value receiver.

### Writing a sink

Any method on a pipe which returns something other than a pipe is a sink. For example, here's an implementation of `String()`:

```go
func (p *script.Pipe) String() (string, error) {
	if p.Error() != nil {
		return "", p.Error()
	}
	defer p.Close()
	res, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return "", err
	}
	return string(res), nil
}
```

Again, the first thing we do is check the error status on the pipe. If it's set, we return the zero value (empty string in this case) and the error.

We then defer closing the pipe, and try to read from it. If we get an error reading the pipe, we set the pipe's error status and return the zero value and the error.

Otherwise, we return the result of reading the pipe, and a nil error.

## Ideas

These are some ideas I'm playing with for additional features. If you feel like working on one of them, send a pull request. If you have ideas for other features, open an issue (or, better, a pull request).

### Sources

* `Get()` makes a web request, like `curl`, and pipes the result
* `Net()` makes a network connection to a specified address and port, and reads the connection until it's closed
* `ListFiles()` takes a filesystem path or glob, and pipes the list of matching files
* `Find()` pipes a list of files matching various criteria (name, modified time, and so on)
* `Processes()` pipes the list of running processes, like `ps`.

### Filters

* Ideas welcome!

### Sinks

* Ideas equally welcome!

### Examples

Since `script` is designed to help you write system administration programs, a few simple examples of such programs are included in the [examples](examples/) directory:

* [cat](examples/cat/main.go) (copies stdin to stdout)
* [cat 2](examples/cat2/main.go) (takes a list of files on the command line and concatenates their contents to stdout)
* [grep](examples/grep/main.go)
* [echo](examples/echo/main.go)

More examples would be welcome!

### Use cases

The best libraries are designed to satisfy real use cases. If you have a sysadmin task which you'd like to implement with `script`, let me know by opening an issue.
