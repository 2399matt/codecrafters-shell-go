package main

import (
	"os"
	"slices"
	"sort"
	"strings"
)

type TabAction int

const (
	NoMatch TabAction = iota
	SingleMatch
	PartialMatch
	ShowSuggestions
)

type TabResult struct {
	action              TabAction
	suffix              string
	suggestions         []string
	awaitingSuggestions bool
}

func autocomplete(input string, multiple bool) TabResult {
	if keyword, ok := searchKeyword(input); ok {
		suffix := keyword[len(input):]
		return TabResult{
			action:              SingleMatch,
			suffix:              suffix,
			awaitingSuggestions: false,
		}
	}
	parts := strings.Split(input, " ")
	if idx := strings.LastIndex(parts[len(parts)-1], "/"); idx != -1 {
		target := parts[len(parts)-1][idx+1:]
		path := parts[len(parts)-1][:idx]
		fileName, isDir := searchForPath(path, target)
		if fileName != "" {
			if isDir {
				fileName += "/"
			}
			return TabResult{action: SingleMatch, suffix: fileName, awaitingSuggestions: false}
		} else {
			return TabResult{action: NoMatch, awaitingSuggestions: false}
		}
	}
	matches := make([]string, 0)
	target := input
	if len(parts) > 1 {
		target = parts[len(parts)-1]
		matches = searchWorkingDir(target)
	} else {
		matches = searchForExec(target)
	}
	return handleCompletion(target, matches, multiple)
}

func handleCompletion(input string, matches []string, multiple bool) TabResult {
	if len(matches) == 0 {
		return TabResult{action: NoMatch, awaitingSuggestions: false}
	}
	if len(matches) == 1 {
		return TabResult{
			action:              SingleMatch,
			suffix:              matches[0][len(input):],
			awaitingSuggestions: false,
		}
	}
	lcp := longestCommonPrefix(matches)
	if len(lcp) > len(input) {
		return TabResult{
			action:              PartialMatch,
			suffix:              lcp[len(input):],
			awaitingSuggestions: false,
		}
	}
	if multiple {
		return TabResult{
			action:              ShowSuggestions,
			suggestions:         matches,
			awaitingSuggestions: false,
		}
	}
	return TabResult{action: NoMatch, awaitingSuggestions: true}
}

func longestCommonPrefix(files []string) string {
	copied := make([]string, len(files))
	copy(copied, files)
	sort.Slice(copied, func(i, j int) bool {
		return len(copied[i]) < len(copied[j])
	})
	shortest := copied[0]
	res := ""
	for _, str := range files {
		if len(str) < len(shortest) {
			shortest = str
		}
	}
	for i := 0; i < len(shortest); i++ {
		ch := shortest[i]
		for _, str := range files {
			if i < len(str) && str[i] != ch {
				return res
			}
		}
		res += string(ch)
	}
	return res
}

func searchKeyword(prefix string) (string, bool) {
	for _, kw := range keywords {
		if strings.HasPrefix(kw, prefix) {
			return kw, true
		}
	}
	return "", false
}

func searchWorkingDir(prefix string) []string {
	results := make([]string, 0)
	files, err := os.ReadDir("./")
	if err != nil {
		return results
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), prefix) {
			continue
		}
		results = append(results, file.Name())
	}
	slices.Sort(results)
	return results
}

func searchForExec(prefix string) []string {
	results := make([]string, 0)
	for _, path := range paths {
		files, err := os.ReadDir(path)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.HasPrefix(file.Name(), prefix) {
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.Mode().Perm()&0o111 != 0 {
				results = append(results, file.Name())
			}
		}
	}
	slices.Sort(results)
	return results
}

func searchForPath(path, prefix string) (string, bool) {
	//results := make([]string, 0)
	files, err := os.ReadDir(path)
	if err != nil {
		return "", false
	}
	if prefix == "" && len(files) == 1 {
		return files[0].Name(), files[0].IsDir()
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			return file.Name(), file.IsDir()
		}
	}
	return "", false
}
