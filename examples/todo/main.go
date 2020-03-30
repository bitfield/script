package main

import (
	"fmt"
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
			if strings.Contains(str, "todo") {
				builderFile.WriteString(fmt.Sprintf("%s:%d %s \n", filePath, lineNumber, strings.TrimSpace(str)))
			}
			lineNumber++
		})
	})
	content.Stdout()
}
