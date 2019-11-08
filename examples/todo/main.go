package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bitfield/script"
)

func main() {
	listPath := "."
	if len(os.Args) > 1 {
		listPath = os.Args[1]
	}
	filterFiles := regexp.MustCompile(`^\..*|/\.`)
	files := script.FindFiles(listPath).RejectRegexp(filterFiles)
	content := files.EachLine(func(filePath string, builderFile *strings.Builder) {
		p := script.File(filePath)
		lineNumber := 1
		// keep track of comments0
		isInsideACommentBlock := false // track for multiline comments
		previousLineHasTodo := false   // if previous line has todo, and next line has another todo, treat them as separate todos
		p.EachLine(func(str string, build *strings.Builder) {
			if isInsideACommentBlock {
				// in this case, just look for todos until the multiline comment is closed
				hasCommentBlockCloser, err := findCommentBlockCloser([]byte(str))
				if err != nil {
					log.Fatal(err)
				}
				findTodo, err := hasTodo([]byte(str))
				if err != nil {
					log.Fatal(err)
				}
				if findTodo {
					builderFile.WriteString(fmt.Sprintf("%s\t%d\t%s", strings.TrimSpace(str), lineNumber, filePath))
				}
				// probably check for text before closer and append it to be part of todo
				// comment block was just closed
				if hasCommentBlockCloser {
					isInsideACommentBlock = false
					lineNumber++
					return
				}
				// too repetitive, find a solution for this
				lineNumber++
				return
			}
			hasComment, isBlockComment, err := lineHasComment([]byte(str))
			if err != nil {
				log.Fatal(err)
			}
			// notify next line of block comment section
			isInsideACommentBlock = isBlockComment
			// no comment continue
			if !hasComment {
				previousLineHasTodo = false
				lineNumber++
				return
			}
			findTodo, err := hasTodo([]byte(str))
			if err != nil {
				log.Fatal(err)
			}
			if previousLineHasTodo && !findTodo {
				// append content of this comment to the last todo created
				// previousLineHasTodo = findTodo
				// index := len(foundTodos) - 1
				// foundTodos[index].todo = fmt.Sprintf("%s\n%s", foundTodos[index].todo, strings.TrimSpace(str))
				// lineNumber++
				return
			}
			if findTodo {
				builderFile.WriteString(fmt.Sprintf("%s\t%d\t%s\n", strings.TrimSpace(str), lineNumber, filePath))
			}
			previousLineHasTodo = findTodo
			lineNumber++
		})
		// for each file do something like finding todos
		// build.WriteString(content)
	})
	content.Stdout()
}

// lineHasComment finds out whether contains comment or not
// it also reports whether a comment is multiline i.e /* */ or
// single line comment.
func lineHasComment(content []byte) (bool, bool, error) {
	// check for multiline comments also closed within the same line /* */
	// comments to look for include:
	// - /* ... */
	// - //
	// - #
	hasOneLineComment, err := regexp.Match(`\/\/.*|\#.*|\/\*.*\*\/`, content)
	if err != nil {
		return false, false, fmt.Errorf("An error occurred while finding todo: %v", err)
	}
	if hasOneLineComment {
		return true, false, nil
	}
	hasCommentBlockOpener, err := regexp.Match(`\/\*.*`, content)
	if err != nil {
		return false, false, fmt.Errorf("An error occurred while finding todo: %v", err)
	}
	if hasCommentBlockOpener {
		return true, true, nil
	}
	return false, false, nil
}

// findCommentBlockCloser looks for the end of an open comment block - `*/`
// this should only be called if an open comment block is already found
func findCommentBlockCloser(content []byte) (bool, error) {
	hasCommentBlockCloser, err := regexp.Match(`(?i)\*\/.*`, content)
	if err != nil {
		return false, fmt.Errorf("An error occurred while finding todo: %v", err)
	}
	return hasCommentBlockCloser, nil
}

// hasTodo finds whether content passed contains a todo
func hasTodo(content []byte) (bool, error) {
	findTodo, err := regexp.Match(`(?i)todo.*`, content)
	if err != nil {
		return false, fmt.Errorf("An error occurred while finding todo: %v", err)
	}
	return findTodo, nil
}
