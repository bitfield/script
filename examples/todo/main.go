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
	// filter hidden directories and files
	filterFiles := regexp.MustCompile(`^\..*|/\.`)
	files := script.FindFiles(listPath).RejectRegexp(filterFiles)
	content := files.EachLine(func(filePath string, builderFile *strings.Builder) {
		p := script.File(filePath)
		lineNumber := 1
		p.EachLine(func(str string, build *strings.Builder) {
			findTodo, err := hasTodo([]byte(str))
			if err != nil {
				log.Fatal(err)
			}
			if findTodo {
				builderFile.WriteString(fmt.Sprintf("%s:%d %s \n", filePath, lineNumber, strings.TrimSpace(str)))
			}
			lineNumber++
		})
	})
	content.Stdout()
}

// hasTodo finds whether content passed contains a todo
func hasTodo(content []byte) (bool, error) {
	// find todo or TODO existence as an independent word
	regex := `(?i)\stodo\s.*`
	findTodo, err := regexp.Match(regex, content)
	if err != nil {
		return false, fmt.Errorf("An error occurred while finding todo: %w", err)
	}
	return findTodo, nil
}
