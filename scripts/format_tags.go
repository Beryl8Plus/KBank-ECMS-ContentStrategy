package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Tag struct {
	Key   string
	Value string
}

func (t Tag) String() string {
	return fmt.Sprintf("%s:%q", t.Key, t.Value)
}

func parseTags(tagStr string) []Tag {
	tagStr = strings.Trim(tagStr, "`")
	if tagStr == "" {
		return nil
	}

	var tags []Tag
	for i := 0; i < len(tagStr); {
		for i < len(tagStr) && tagStr[i] == ' ' {
			i++
		}
		if i >= len(tagStr) {
			break
		}

		keyStart := i
		for i < len(tagStr) && tagStr[i] != ':' && tagStr[i] != ' ' {
			i++
		}
		if i >= len(tagStr) || tagStr[i] != ':' {
			return tags
		}

		key := tagStr[keyStart:i]
		i++
		if i >= len(tagStr) || tagStr[i] != '"' {
			return tags
		}

		valueStart := i
		i++
		escaped := false
		for i < len(tagStr) {
			switch {
			case escaped:
				escaped = false
			case tagStr[i] == '\\':
				escaped = true
			case tagStr[i] == '"':
				rawValue := tagStr[valueStart : i+1]
				value, err := strconv.Unquote(rawValue)
				if err != nil {
					return tags
				}
				tags = append(tags, Tag{Key: key, Value: value})
				i++
				goto nextTag
			}
			i++
		}

		return tags

	nextTag:
	}
	return tags
}

func formatTags(tagStr string, maxGormLen int) string {
	if tagStr == "" {
		return ""
	}

	tags := parseTags(tagStr)
	if len(tags) == 0 {
		return tagStr
	}

	var resultParts []string
	for i, t := range tags {
		part := t.String()
		if t.Key == "gorm" && i < len(tags)-1 {
			if maxGormLen > len(part) {
				part += strings.Repeat(" ", maxGormLen-len(part))
			}
		}
		resultParts = append(resultParts, part)
	}

	return "`" + strings.Join(resultParts, " ") + "`"
}

func processFile(path string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	modified := false
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			// First pass: find max GORM length for this struct
			maxGormLen := 0
			for _, field := range x.Fields.List {
				if len(field.Names) > 0 {
					fieldName := field.Names[0].Name
					if !ast.IsExported(fieldName) {
						continue
					}
					var currentTag string
					if field.Tag != nil {
						currentTag = field.Tag.Value
					}
					tags := parseTags(currentTag)
					for _, t := range tags {
						if t.Key == "gorm" {
							gormPart := fmt.Sprintf("gorm:%q", t.Value)
							if len(gormPart) > maxGormLen {
								maxGormLen = len(gormPart)
							}
						}
					}
				}
			}

			// Second pass: apply formatting
			for _, field := range x.Fields.List {
				if len(field.Names) > 0 {
					fieldName := field.Names[0].Name
					if !ast.IsExported(fieldName) {
						continue
					}

					var currentTag string
					if field.Tag != nil {
						currentTag = field.Tag.Value
					}

					newTag := formatTags(currentTag, maxGormLen)
					if newTag == "" {
						if field.Tag != nil {
							field.Tag = nil
							modified = true
						}
					} else if field.Tag == nil || field.Tag.Value != newTag {
						field.Tag = &ast.BasicLit{
							Kind:  token.STRING,
							Value: newTag,
						}
						modified = true
					}
				}
			}
		}
		return true
	})

	if modified {
		out, err := os.Create(path)
		if err != nil {
			return err
		}
		defer out.Close()
		return format.Node(out, fset, f)
	}

	return nil
}

type Config struct {
	TargetDir string
}

func main() {
	cfg := Config{
		TargetDir: "internal/domain/entity",
	}

	err := filepath.Walk(cfg.TargetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			fmt.Printf("Processing %s...\n", path)
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
	fmt.Println("Tag formatting complete.")
}
