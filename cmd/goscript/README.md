# Script runner

This is a Go rewrite of
the [previous iteration](https://til.simonwillison.net/bash/go-script) writen by
from Simon Willison in Bash.

## Installation

System wide

```shell
go install github.com/bitfield/script/cmd/goscript@latest
```

As a module tool

```shell
go get -tool github.com/bitfield/script/cmd/goscript@latest
```

## Shebang arguments

If you ever use multiple arguments for `goscript` from shebang, you can use
`env` command with `-S` flag. It splits the argument into separate strings.

It's because shebang recognizes only two positions:

```
#!first second, stil part of second
```

leading to: `first "second, stil part of second"`, using `env` command like
this:

```shell
#!/usr/bin/env -S cmd -foo -bar
```

you run `cmd "-foo" "-bar"` with two arguments.

## Example

Run as a file,

```shell
#!goscript -i=fmt,os

fmt.Println(os.Getwd())

script.
    File("/etc/passwd").
    Stdout()
```

from your shell,

```shell
cat file.txt | goscript -c 'script.Stdin().Column(1).Freq().First(10).Stdout()'
```

or as a `go tool`.

Define it in your `go.mod` manually,

```
//go.mod
module your.module

go 1.23

tool (
	github.com/bitfield/script/cmd/goscript
)
```

or install it by.

```shell
go get -tool github.com/bitfield/script/cmd/goscript@latest
```

Use by

```shell
go tool goscript
```

or as part of your project flow.

```shell
//go:generate go tool goscript 
```