package pipeline

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// SliceToKeyMap returns a map with the given keys and a true value.
func SliceToKeyMap(keys []string) map[string]bool {
	keyMap := make(map[string]bool, len(keys))

	for _, name := range keys {
		keyMap[name] = true
	}

	return keyMap
}

func GetLinesFromFile(p string) []string {
	f, err := os.ReadFile("test/python-versions")
	if err != nil {
		log.Fatalf("ERROR: cannot get lines from file: %s", err)
	}
	fLines := strings.Split(string(f), "\n")
	// Truncate trailing newline
	if fLines[len(fLines)] == "\n" {
		fLines = fLines[:len(fLines)-1]
	}
	return fLines
}

func WalkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}
