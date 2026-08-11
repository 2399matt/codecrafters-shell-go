package main

import (
	"os"
	"slices"
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
	addSpace            bool
}

type FileMatch struct {
	name  string
	isDir bool
}

func (f FileMatch) displayName() string {
	if f.isDir {
		return f.name + "/"
	}
	return f.name
}

func autocomplete(input string, multiple bool) TabResult {
	if keyword, ok := searchKeyword(input); ok {
		suffix := keyword[len(input):]
		return TabResult{
			action:              SingleMatch,
			suffix:              suffix,
			awaitingSuggestions: false,
			addSpace:            true,
		}
	}
	parts := strings.Split(input, " ")
	lastEntry := parts[len(parts)-1]
	target := lastEntry
	path := "."
	if idx := strings.LastIndex(lastEntry, "/"); idx != -1 {
		target = lastEntry[idx+1:]
		path = lastEntry[:idx+1]
	}
	var matches []FileMatch
	if entry, ok := completion.table[parts[0]]; ok {
		prevArg := ""
		if len(parts) >= 2 {
			prevArg = parts[len(parts)-2]
		}
		matches = getCompleteSuggestions(input, parts[0], prevArg, target, entry)
		return handleCompletion(target, matches, multiple)
	}
	// shouldn't need to branch off anymore, if path is working dir already it should be fine.
	if path != "." || len(parts) > 1 {
		matches = searchForPath(path, target)
	} else {
		matches = searchForExec(target)
	}
	return handleCompletion(target, matches, multiple)
}

func handleCompletion(input string, matches []FileMatch, multiple bool) TabResult {
	if len(matches) == 0 {
		return TabResult{action: NoMatch, awaitingSuggestions: false}
	}
	if len(matches) == 1 {
		return TabResult{
			action:              SingleMatch,
			suffix:              matches[0].displayName()[len(input):],
			awaitingSuggestions: false,
			addSpace:            !matches[0].isDir,
		}
	}
	lcp := longestCommonPrefix(matches)
	if len(lcp) > len(input) {
		return TabResult{
			action:              PartialMatch,
			suffix:              lcp[len(input):],
			awaitingSuggestions: false,
			addSpace:            false,
		}
	}
	if multiple {
		return TabResult{
			action:              ShowSuggestions,
			suggestions:         displayNames(matches),
			awaitingSuggestions: false,
			addSpace:            false,
		}
	}
	return TabResult{action: NoMatch, awaitingSuggestions: true, addSpace: false}
}

func displayNames(matches []FileMatch) []string {
	res := make([]string, 0, len(matches))
	for _, match := range matches {
		res = append(res, match.displayName())
	}
	return res
}

func getCompleteSuggestions(rawInput, cmd, prevArg, prefix string, entry CompletionSpec) []FileMatch {
	matches := make([]FileMatch, 0)
	suggestions := completion.extractSuggestions(rawInput, cmd, prevArg, entry.path, prefix)
	for _, sugg := range suggestions {
		if strings.HasPrefix(sugg, prefix) {
			matches = append(matches, FileMatch{name: sugg, isDir: false})
		}
	}
	return matches
}

func longestCommonPrefix(files []FileMatch) string {
	shortest := files[0].name
	res := ""
	for _, str := range files {
		if len(str.name) < len(shortest) {
			shortest = str.name
		}
	}
	for i := 0; i < len(shortest); i++ {
		ch := shortest[i]
		for _, str := range files {
			if i < len(str.name) && str.name[i] != ch {
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

func searchForExec(prefix string) []FileMatch {
	results := make([]FileMatch, 0)
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
				results = append(results, FileMatch{name: file.Name(), isDir: false})
			}
		}
	}
	slices.SortFunc(results, func(a, b FileMatch) int {
		return strings.Compare(a.name, b.name)
	})
	return results
}

func searchForPath(path, prefix string) []FileMatch {
	results := make([]FileMatch, 0)
	files, err := os.ReadDir(path)
	if err != nil {
		return results
	}
	if len(files) == 1 && strings.HasPrefix(files[0].Name(), prefix) {
		results = append(results, FileMatch{name: files[0].Name(), isDir: files[0].IsDir()})
		return results
	}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), prefix) {
			results = append(results, FileMatch{name: file.Name(), isDir: file.IsDir()})
		}
	}
	return results
}
