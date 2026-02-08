package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/ktrysmt/mermaid-ascii/cmd"
	"golang.org/x/term"
)

var pattern = "(?s)```mermaid(.*?)```"
var mermaidBlockRe = regexp.MustCompile(pattern)

func convertMermaid(mermaidSrc string) string {
	// Use mermaid-ascii as a library
	asciiArt, err := cmd.Convert(mermaidSrc)
	if err == nil && asciiArt != "" {
		return strings.TrimSuffix(asciiArt, "\n")
	}

	// Fallback: return original as-is in a code block
	return strings.TrimSuffix(mermaidSrc, "\n")
}

func processMarkdown(content string) (string, bool) {

	var hadMermaid bool

	processed := mermaidBlockRe.ReplaceAllStringFunc(content, func(match string) string {
		hadMermaid = true
		// Extract content between ```mermaid and ```
		captures := mermaidBlockRe.FindStringSubmatch(match)
		if len(captures) < 2 {
			return match
		}
		mermaidSrc := captures[1]
		asciiArt := convertMermaid(mermaidSrc)
		return fmt.Sprintf("```\n%s\n```", asciiArt)
	})

	return processed, hadMermaid
}

func renderWithGlamour(content string, width int) error {
	// Create glamour renderer with width option
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}

	// Render the markdown
	out, err := r.Render(content)
	if err != nil {
		return err
	}

	// Write to stdout
	fmt.Print(out)
	return nil
}

func parseWidth(args []string) int {
	// Default width: use terminal width or 80
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return width
	}
	return 80
}

func main() {
	// Parse arguments: separate file path from options
	var filePath string
	renderArgs := []string{}

	for _, arg := range os.Args[1:] {
		if filePath == "" && !strings.HasPrefix(arg, "-") {
			// Check if it's a file (similar to os.path.isfile in Python)
			if info, err := os.Stat(arg); err == nil && !info.IsDir() {
				filePath = arg
			}
		} else {
			renderArgs = append(renderArgs, arg)
		}
	}

	// Read markdown content
	var content string
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	} else if !isTTY(os.Stdin.Fd()) {
		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		content = string(data)
	} else {
		// No file and no stdin: print usage
		fmt.Fprintln(os.Stderr, "Usage: mmdv [options] <file.md>")
		fmt.Fprintln(os.Stderr, "       cat file.md | mmdv [options]")
		os.Exit(1)
	}

	processed, _ := processMarkdown(content)

	// Parse width from args or use terminal width
	width := parseWidth(renderArgs)
	for i, arg := range renderArgs {
		if arg == "-w" || arg == "--width" {
			if i+1 < len(renderArgs) {
				if w, err := strconv.Atoi(renderArgs[i+1]); err == nil {
					width = w
				}
			}
		}
	}

	// Render with glamour (no temp file needed)
	if err := renderWithGlamour(processed, width); err != nil {
		fmt.Fprintf(os.Stderr, "Render error: %v\n", err)
		os.Exit(1)
	}
}

func isTTY(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}
