package pipeline

import (
	"log"
	"os"
	"strings"
)

// sliceToKeyMap returns a map with the given keys and a true value.
func sliceToKeyMap(keys []string) map[string]bool {
	keyMap := make(map[string]bool, len(keys))

	for _, name := range keys {
		keyMap[name] = true
	}

	return keyMap
}

func getLinesFromFile(p string) []string {
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
