package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	idPattern    = regexp.MustCompile(`lison:(?:id|field)=([a-z0-9_]+)`)
	wordPattern  = regexp.MustCompile(`^\s*(.+?)\s+::\s+(.+?)\s+<!--\s*lison:id=([a-z0-9_]+)\s*-->\s*$`)
	validID      = regexp.MustCompile(`^[a-z0-9_]+$`)
	requiredMeta = []string{"schema_version", "id", "slug", "title", "summary", "level", "learning_language", "translation_language", "status", "order"}
)

func main() {
	root := flag.String("root", "./contents/topics", "root directory containing topic Markdown files")
	flag.Parse()

	files := 0
	errorsFound := 0
	err := filepath.WalkDir(*root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) != "topic.md" {
			return nil
		}
		files++
		issues, wordCount := validate(path)
		if len(issues) == 0 {
			fmt.Printf("OK %s (%d vocabulary items)\n", path, wordCount)
			return nil
		}
		errorsFound += len(issues)
		for _, issue := range issues {
			fmt.Printf("ERROR %s\n", issue)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if files == 0 {
		fmt.Fprintln(os.Stderr, "no topic.md files found")
		os.Exit(1)
	}
	if errorsFound > 0 {
		os.Exit(1)
	}
}

func validate(path string) ([]string, int) {
	body, err := os.ReadFile(path)
	if err != nil {
		return []string{path + ": " + err.Error()}, 0
	}
	text := string(body)
	issues := make([]string, 0)
	if strings.Contains(strings.ToLower(text), "<script") {
		issues = append(issues, path+": executable script markup is not allowed")
	}

	metadata, ok := frontMatter(text)
	if !ok {
		issues = append(issues, path+": missing YAML front matter")
	} else {
		for _, key := range requiredMeta {
			if metadata[key] == "" {
				issues = append(issues, fmt.Sprintf("%s: missing metadata %q", path, key))
			}
		}
		if id := metadata["id"]; id != "" && !validID.MatchString(id) {
			issues = append(issues, path+": topic id must contain lowercase Latin letters, digits or underscores")
		}
	}

	seen := map[string]int{}
	wordCount := 0
	scanner := bufio.NewScanner(strings.NewReader(text))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		for _, match := range idPattern.FindAllStringSubmatch(line, -1) {
			id := match[1]
			if previous, exists := seen[id]; exists {
				issues = append(issues, fmt.Sprintf("%s:%d duplicate id %q (first used on line %d)", path, lineNumber, id, previous))
			} else {
				seen[id] = lineNumber
			}
		}
		if strings.Contains(line, " :: ") {
			match := wordPattern.FindStringSubmatch(line)
			if match == nil || strings.TrimSpace(match[1]) == "" || strings.TrimSpace(match[2]) == "" {
				issues = append(issues, fmt.Sprintf("%s:%d invalid vocabulary entry", path, lineNumber))
			} else {
				wordCount++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		issues = append(issues, path+": "+err.Error())
	}
	if wordCount == 0 {
		issues = append(issues, path+": topic has no vocabulary entries")
	}
	return issues, wordCount
}

func frontMatter(text string) (map[string]string, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	result := map[string]string{}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return result, true
		}
		key, value, found := strings.Cut(line, ":")
		if found {
			result[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return nil, false
}
