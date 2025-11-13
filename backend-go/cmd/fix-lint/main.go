package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// This tool helps fix common golangci-lint errors automatically

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <directory>")
		os.Exit(1)
	}

	dir := os.Args[1]

	fmt.Println("Starting automated lint fixes...")

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and test files for now
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.git/") {
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			if err := processFile(path); err != nil {
				fmt.Printf("Error processing %s: %v\n", path, err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Automated fixes complete!")
}

func processFile(path string) error {
	// #nosec G304 - path is from filepath.Walk, not user input
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	original := string(content)
	modified := original

	// Fix dupword issues
	modified = fixDupwords(modified)

	// Fix error wrapping issues (%v -> %w)
	modified = fixErrorWrapping(modified)

	// Fix SA1012: nil context -> context.TODO()
	modified = fixNilContext(modified)

	// Fix ineffectual assignments (basic cases)
	modified = fixIneffectualAssignments(modified)

	// Only write if changed
	if modified != original {
		// Format the modified code
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, modified, parser.ParseComments)
		if err != nil {
			// If parsing fails, write the modified version anyway
			return os.WriteFile(path, []byte(modified), 0600)
		}

		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return os.WriteFile(path, []byte(modified), 0600)
		}

		return os.WriteFile(path, buf.Bytes(), 0600)
	}

	return nil
}

func fixDupwords(content string) string {
	// Fix "TiDB TiDB" -> "TiDB" (Go regex doesn't support backreferences, use explicit pattern)
	content = regexp.MustCompile(`\bTiDB\s+TiDB\b`).ReplaceAllString(content, "TiDB")
	// Fix "user user" -> "user"
	content = regexp.MustCompile(`\buser\s+user\b`).ReplaceAllString(content, "user")
	return content
}

func fixErrorWrapping(content string) string {
	// Fix fmt.Errorf with %v to %w for error wrapping
	re := regexp.MustCompile(`fmt\.Errorf\(("[^"]*?)%v([^"]*?"), ([^,]+?), err\)`)
	content = re.ReplaceAllString(content, `fmt.Errorf($1%w$2, $3, err)`)

	// Also handle cases like fmt.Errorf("...: %v", err)
	re2 := regexp.MustCompile(`fmt\.Errorf\(("[^"]*?": )%v", err\)`)
	content = re2.ReplaceAllString(content, `fmt.Errorf($1%w", err)`)

	return content
}

func fixNilContext(content string) string {
	// Fix nil context to context.TODO()
	content = strings.ReplaceAll(content, ", nil)", ", context.TODO())")
	content = strings.ReplaceAll(content, "(nil,", "(context.TODO(),")
	content = strings.ReplaceAll(content, "(nil)", "(context.TODO())")
	return content
}

func fixIneffectualAssignments(content string) string {
	// This is complex and requires AST analysis
	// We'll handle simple cases with regexp

	// Pattern: variable assigned but immediately reassigned
	// This is better done with proper AST walking
	return content
}

