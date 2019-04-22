[![GoDoc](https://godoc.org/github.com/bitfield/script?status.png)](http://godoc.org/github.com/bitfield/script)[![Go Report Card](https://goreportcard.com/badge/github.com/bitfield/script)](https://goreportcard.com/report/github.com/bitfield/script)

`script` is a collection of utilities for doing the kind of tasks that shell scripts are good at: reading files, counting lines, matching strings, and so on. Why shouldn't it be as easy to write system administration programs in Go as it is in a typical shell? `script` aims to make it just that easy.

Just as the shell allows you to chain operations together into a pipeline, `script` does the same:

```go
numLines := script.File("test.txt").CountLines()
```

This works because File returns a Pipe object. Most `script` operations can be methods on a pipe, and will return another pipe, so that you can chain operations indefinitely.

Ultimately, you'll want to read the results from the pipe, and you can do that using the `String()` method, for example. If the pipe's original data source was something that needs closing, like a file, it will be automatically closed once all the data has been read. `script` programs will not leak file handles.

If any pipe operation results in an error, the pipe's `Error()` method will return that error, and all pipe operations will henceforth be no-ops. Thus you can safely chain a whole series of operations without having to check the error status at each stage.

When you eventually read the pipe, for convenience, you will also get the pipe's error status as an extra return value:

```go
p := script.File("doesnt_exist.txt")
out, err := p.String() // err is non-nil
res, err := p.CountLines() // err is non-nil
fmt.Println(p.Error())
// Output: open doesnt_exist.txt: no such file or directory
```