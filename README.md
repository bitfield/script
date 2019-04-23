[![GoDoc](https://godoc.org/github.com/bitfield/script?status.png)](http://godoc.org/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)

# script

`script` is a collection of utilities for doing the kind of tasks that shell scripts are good at: reading files, counting lines, matching strings, and so on. Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

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
p = File("test.txt")
```

You might expect `File()` to return an error if there is a problem opening the file, but it doesn't. We will want to call a chain of methods on the result of `File()`, and it's inconvenient to do that if it also returns an error.

Instead, you can check the error status of the pipe at any time by calling its `Error()` method:

```go
p = File("test.txt")
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
q = File("test.txt").Match("Error")
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

Note that sinks return an error value in addition to the data. This is the same value you would get by calling `p.Error()`. If the pipe had an error in any operation along the pipeline, the pipe's error status will be set, and a sink operation which gets output will return the zero value, plus the error.

```go
result, _ := File("doesnt_exist.txt").CountLines()
// Output: 0
```

`CountLines()` is another useful sink, which simply returns the number of lines read from the pipe.

## Closing pipes

If you've dealt with files in Go before, you'll know that you need to _close_ the file once you've finished with it. Otherwise, the program will retain what's called a _file handle_ (the kernel data structure which represents an open file). There is a limit to the total number of open file handles for a given program, and for the system as a whole, so a program which leaks file handles will eventually crash, and will waste resources in the meantime.

How does `script` handle this? Simple. Any operation which completely reads the data from a pipe (such as `Match()` or `String()`) also closes the pipe by calling `p.Close()`. This closes the original data source that was used to create the pipe.

If the pipe was created from something that doesn't need to be closed, such as a string, then calling `Close()` simply does nothing.

TL;DR you don't need to worry about closing pipes; `script` will do it for you.

## Sources, filters, and sinks

So `script` provides three types of pipe operations:

1. _Sources_ create pipes from input in some way (for example, `File()` opens a file).
2. _Filters_ read from a pipe and filter the data in some way (for example `Match()` passes on only lines which contain a given string).
3. _Sinks_ get the output from a pipeline in some useful form (for example `String()` returns the contents of the pipe as a string), along with any error status.

Let's look at the source, filter, and sink options that `script` provides.

## Sources

These are operations which create a pipe.

### File

`File()` creates a pipe that reads from a file.

```go
p = File("test.txt")
```

### Echo

`Echo()` creates a pipe containing a given string:

```go
p := script.Echo("Hello, world!")
fmt.Println(p.String())
// Output: Hello, world!
```

## Filters

Filters are operations on an existing pipe that also return a pipe, allowing you to chain filters indefinitely.

### Match

`Match()` returns a pipe containing only the input lines which match the supplied string:

```go
p := File("test.txt").Match("Error")
```

### Reject

`Reject()` is the inverse of `Match()`. Its pipe produces only lines which _don't_ contain the given string:

```go
p := File("test.txt").Match("Error").Reject("false alarm")
```

### EachLine

`EachLine()` lets you create custom filters. You provide a function, and it will be called once for each line of input. If you want to produce output, your function can write to a supplied `strings.Builder`. The return value from EachLine is a pipe containing your output.

```go
p := File("test.txt")
q := p.EachLine(func(line string, out *strings.Builder) {
	out.WriteString("> " + line + "\n")
})
fmt.Println(q.String())
```

## Sinks

Sinks are operations which return some data from a pipe, ending the pipeline.

### String

The simplest sink is `String()`, which just returns the contents of the pipe as a string, plus an error:

```go
contents, err := script.File("test.txt").String()
```

### CountLines

`CountLines`, as the name suggests, counts lines in its input, and returns the number of lines as an integer, plus an error:

```go
numLines, err := script.File("test.txt").CountLines()
```

## Writing your own pipe operations

There's nothing to stop you writing your own sources, sinks, or filters (in fact, that would be excellent. Please submit a pull request if you want to add them to the standard operations supplied with `script`.)

### Writing a source

All a pipe source has to do is return a pointer to a `script.Pipe`. To be useful, a pipe needs to have a `Reader` attached to it. This is simply anything that implements `io.Reader`.

`Echo()` is a simple example, which just creates a pipe containing a string:

```go
func Echo(s string) *Pipe {
	return NewPipe().WithReader(strings.NewReader(s))
}
```

Let's break this down:

* We create a `strings.Reader` to be our data source, using `strings.NewReader` on the supplied string.
* We create a new pipe with `NewPipe()`.
* We attach the reader to the pipe with `WithReader()`.

Simple, right? One more thing: when you create a pipe from a data source that needs to be closed after reading (such as a file), use `WithCloser()` instead of `WithReader()`.

Here's the implementation of `File()`, for example:

```go
func File(name string) *Pipe {
	r, err := os.Open(name)
	if err != nil {
		return NewPipe().WithError(err)
	}
	return NewPipe().WithCloser(r)
}
```

You can also see from this example how to create a pipe with error status:

```go
return NewPipe().WithError(err)
```

### Writing a filter

Filters are methods on pipes, that return pipes. For example, here's a simple filter which just reads and rejects all input, returning an empty pipe:

```go
func (p *Pipe) RejectEverything() *Pipe {
	if p.Error() != nil {
		return &p
	}
	defer p.Close()
	_, err := ioutil.ReadAll(p.Reader)
	if err != nil {
		p.SetError(err)
		return &p
	}
	return Echo("")
}
```

Important things to note here:

* The first thing we do is check the pipe's error status. If this is set, we do nothing, and just return the original pipe.
* We close the source pipe once we successfully read all data from it.
* If an error occurs, we set the pipe's error status, using `p.SetError()`, and return the pipe.

Filters must not log anything, terminate the program, or return anything but `*Pipe`.

As you can see from the example, the pipe's reader is available to you as `p.Reader`. You can do anything with that that you can with an `io.Reader`.

If your method modifies the pipe (for example if it can set an error on the pipe), it must take a pointer receiver, as in this example. Otherwise, it can take a value receiver.

### Writing a sink

Any method on a pipe which returns something other than a pipe is a sink. For example, here's an implementation of `String()`:

```go
func (p *Pipe) String() (string, error) {
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

* `OpenURL()` makes a web request, like `curl`, and pipes the result
* `Exec()` runs an external program, and pipes its output
* `Stdin()` pipes the program's standard input
* `ListFiles()` takes a filesystem path or glob, and pipes the list of matching files
* `Find()` pipes a list of files matching various criteria (name, modified time, and so on)
* `Processes()` pipes the list of running processes, like `ps`.

### Filters

* `MatchRegex` / `RejectRegex`. You can probably guess what these do.

### Sinks

* `WriteFile` writes the contents of the pipe to a file.
* `AppendFile`... well, you get the idea.

### Examples

Since `script` is designed to help you write system administration programs, it would be great to have some examples of such programs. We could implement familiar utilities like `cat`, `grep`, and `wc`.
