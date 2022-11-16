So you'd like to contribute to the `script` library? Excellent! Thank you very much. I can absolutely use your help.

# Getting started

Here are some hints on a good workflow for contributing to the project.

## Look for existing issues

First of all, check the [issues](https://github.com/bitfield/script/issues) list. If you see an outstanding issue that you would like to tackle, by all means comment on the issue and let me know.

If you already have an idea for a feature you want to add, check the issues list anyway, just to make sure it hasn't already been discussed.

## Open a new issue before making a PR

I _don't_ recommend just making a pull request for some new feature—it probably won't be accepted! Usually it's better to [open an issue](https://github.com/bitfield/script/issues/new) first, and we can discuss what the feature is about, how best to design it, other people can weigh in with contributions, and so forth. Design is, in fact, the hard part. Once we have a solid, well-thought-out design, implementing it is usually fairly easy. (Implementing a bad design may be easy too, but it's a waste of effort.)

## Write a use case

This is probably the most important thing to bear in mind. A great design principle for software libraries is to start with a real-world use case, and try to implement it using the feature you have in mind. _No issues or PRs will be accepted into `script` without an accompanying use case_. And I hold myself to that rule just as much as anybody else.

What do I mean by "use case"? I mean a real problem that you or someone else actually has, that could be solved using the feature. For example, you might think it's a very cool idea to add a `Frobnicate()` method to `script`. Maybe it is, but what's it for? Where would this be used in the real world? Can you give an example of a problem that could be solved by a `script` program using `Frobnicate()`? If so, what would the program look like?

The reason for insisting on this up front is that it's much easier to design a feature the right way if you start with its usage in mind. It's all too easy to design something in the abstract, and then find later that when you try to use it in a program, the API is completely unsuitable.

A concrete use case also provides a helpful example program that can be included with the library to show how the feature is used.

The final reason is that it's tempting to over-elaborate a design and add all sorts of bells and whistles that nobody actually wants. Simple APIs are best. If you think of an enhancement, but it's not needed for your use case, leave it out. Things can always be enhanced later if necessary.

# Coding standards

A library is easier to use, and easier for contributors to work on, if it has a consistent, unified style, approach, and layout. Here are a few hints on how to make a `script` PR that will be accepted right away.

## Tests

It goes without saying, but I'll say it anyway, that you must provide comprehensive tests for your feature. Code coverage doesn't need to be 100% (that's a waste of time and effort), but it does need to be very good. The [awesome-go](https://github.com/avelino/awesome-go) collection (which `script` is part of) mandates at least 80% coverage, and I'd rather it were 90% or better.

Test data should go in the `testdata` directory. If you create a file of data for input to your method, name it `method_name.input.txt`. If you create a 'golden' file (of correct output, to compare with the output from your method) name it `method_name.golden.txt`. This will help keep things organised.

### Use the standard library

All `script` tests use the standard Go `testing` library; they don't use `testify` or `gock` or any of the other tempting and shiny test libraries. There's nothing wrong with those libraries, but it's good to keep things consistent, and not import any libraries we don't absolutely need.

You'll get the feel of things by reading the existing tests, and maybe copying and adapting them for your own feature.

All tests should call `t.Parallel()`. If there is some really good reason why your test can't be run in parallel, we'll talk about it.

### Spend time on your test cases

Add lots of test cases; they're cheap. Don't just test the obvious happy-path cases; test the null case, where your feature does nothing (make sure it does!). Test edge cases, strange inputs, missing inputs, non-ASCII characters, zeroes, and nils. Knowing what you know about your implementation, what inputs and cases might possibly cause it to break? Test those.

Remember people are using `script` to write mission-critical system administration programs where their data, their privacy, and even their business could be at stake. Now, of course it's up to them to make sure that their programs are safe and correct; library maintainers bear no responsibility for that. But we can at least ensure that the code is as reliable and trustworthy as we can make it.

### Add your method to `doMethodsOnPipe` for stress testing

One final point: a common source of errors in Go programs is methods being called on zero or nil values. All `script` pipe methods should handle this situation, as well as being called on a valid pipe that just happens to have no contents (such as a newly-created pipe).

To ensure this, we call every possible method on (in turn) a nil pipe, a zero pipe, and an empty pipe, using the `doMethodsOnPipe` helper function. If you add a new method to `script`, add a call to your method to this helper function, and it will automatically be stress tested.

Methods on a nil, zero, or empty pipe should not necessarily do nothing; that depends on the method semantics. For example, `WriteFile()` on an empty pipe creates the required file, writes nothing to it, and closes it. This is correct behaviour.

## Dealing with errors

Runtime errors (as opposed to test failures or compilation errors) are handled in a special way in `script`.

### Don't panic

Methods should not, in any situation, panic. In fact, no `script` method panics, nor should any library method. Because calling `panic()` ends the program, this decision should be reserved for the `main()` function. In other words, it's up to the user, not us, when to crash the program. This is a good design principle for Go libraries in general, but especially here because we have a better way of dealing with errors.

### Set the pipe's error status

Normally, Go library code that encounters a problem would return an error to the caller, but `script` methods are specifically designed not to do this (see [Handling errors](README.md#Handling-errors)). Instead, set the error status on the pipe and return. Before you do anything at all in your method, you should check whether the pipe is nil, or the error status is set, and if so, return immediately.

Here's an example:

```go
func (p *Pipe) Frobnicate() *Pipe {
	// If the pipe has an error, or is nil, this is a no-op
	if p == nil || p.Error() != nil {
		return p
	}
	output, err := doSomething()
	if err != nil {
		// Something went wrong, so save the error in the pipe. The user can
		// check it afterwards.
		p.SetError(err)
		return p
	}
	return NewPipe().WithReader(bytes.NewReader(output))
}
```

## Style and formatting

This is easy in Go. Just use `gofmt`. End of.

Your code should also pass `golint` and `go vet` without errors (and if you want to run other linters too, that would be excellent). Very, very occasionally there are situations where `golint` incorrectly detects a problem, and the workaround is awkward or annoying. In that situation, comment on the PR and we'll work out how best to handle it.

# Documentation

It doesn't matter if you write the greatest piece of code in the history of the world, if no one knows it exists, or how to use it.

## Write doc comments

Any functions or methods you write should have useful documentation comments in the standard `go doc` format. Specifically, they should say what inputs the function takes, what it does (in detail), and what outputs it returns. If it returns an error value, explain under what circumstances this happens.

For example:

```go
// WriteFile writes the contents of the pipe to the specified file, and closes
// the pipe after reading. If the file already exists, it is truncated and the
// new data will replace the old. It returns the number of bytes successfully
// written, or an error.
func (p *Pipe) WriteFile(fileName string) (int64, error) {
```

This is the _whole_ user manual for your code. It will be included in the autogenerated documentation for the whole package. Remember that readers will often see it _without_ the accompanying code, so it needs to make sense on its own.

## Update the README

Any change to the `script` API should also be accompanied by an update to the README. If you add a new method, add it to the appropriate table (sources, filters, or sinks), and if it's the equivalent of a command Unix command, add it to the table of Unix equivalents too.

# Before submitting your pull request

Here's a handy checklist for making sure your PR will be accepted as quickly as possible.

- [ ] Have you opened an issue to discuss the feature and agree its general design?
- [ ] Do you have a use case and, ideally, an example program using the feature?
- [ ] Do you have tests covering 90%+ of the feature code (and, of course passing)
- [ ] Have you added your method to the `doMethodsOnPipe` stress tests?
- [ ] Have you written complete and accurate doc comments?
- [ ] Have you updated the README and its table of contents?
- [ ] You rock. Thanks a lot.

# After submitting your PR

Here's a nice tip for PR-driven development in general. After you've submitted the PR, do a 'pre-code-review'. Go through the diffs, line by line, and be your own code reviewer. Does something look weird? Is something not quite straightforward? It's quite likely that you'll spot errors at this stage that you missed before, simply because you're looking at the code with a reviewer's mindset.

If so, fix them! But if you can foresee a question from a code reviewer, comment on the code to answer it in advance. (Even better, improve the code so that the question doesn't arise.)

# The code review process

If you've completed all these steps, I _will_ invest significant time and energy in giving your PR a detailed code review. This is a powerful and beneficial process that can not only improve the code, but can also help you learn to be a better engineer and a better Go programmer—and the same goes for me!

## Expect to be taken seriously

Don't think of code review as a "you got this wrong, fix it" kind of conversation (this isn't a helpful review comment). Instead, think of it as a discussion where both sides can ask questions, make suggestions, clarify problems and misunderstandings, catch mistakes, and add improvements.

You shouldn't be disappointed if you don't get a simple 'LGTM' and an instant merge. If this is what you're used to, then your team isn't really doing code review to its full potential. Instead, the more comments you get, the more seriously it means I'm taking your work. Where appropriate, I'll say what I liked as well as what I'd like to see improved.

## Dealing with comments

Now comes the tricky bit. You may not agree with some of the code review comments. Reviewing code is a delicate business in the first place, requiring diplomacy as well as discretion, but responding to code reviews is also a skilled task.

If you find yourself reacting emotionally, take a break. Go walk in the woods for a while, or play with a laughing child. When you come back to the code, approach it as though it were someone else's, not your own, and ask yourself seriously whether or not the reviewer _has a point_.

If you genuinely think the reviewer has just misunderstood something, or made a mistake, try to clarify the issue. Ask questions, don't make accusations. Remember that every project has a certain way of doing things that may not be _your_ way. It's polite to go along with these practices and conventions.

You may feel as though you're doing the project maintainer a favour by contributing, as indeed you are, but an open source project is like somebody's home. They're used to living there, they probably like it the way it is, and they don't always respond well to strangers marching in and rearranging the furniture. Be considerate, and be willing to listen and make changes.

## This may take a while

Don't be impatient. We've all had the experience of sending in our beautifully-crafted PR and then waiting, waiting, waiting. Why won't those idiots just merge it? How come other issues and PRs are getting dealt with ahead of mine? Am I invisible?

In fact, doing a _proper_ and serious code review is a time-consuming business. It's not just a case of skim-reading the diffs. The reviewer will need to check out your branch, run the tests, think carefully about what you've done, make suggestions, test alternatives. It's almost as much work as writing the PR in the first place.

Open source maintainers are just regular folk with jobs, kids, and zero free time or energy. They may not be able to drop everything and put in several hours on your PR. The task may have to wait a week or two until they can get sufficient time and peace and quiet to work on it. Don't pester them. It's fine to add a comment on the PR if you haven't heard anything for a while, asking if the reviewer's been able to look at it and whether there's anything you can do to help speed things up. Comments like 'Y U NO MERGE' are unlikely to elicit a positive response.

Thanks again for helping out!

## Code of Conduct

As a contributor you can help keep the `script` community inclusive and open to everyone. Please read and adhere to our [Code of Conduct](CODE_OF_CONDUCT.md).
