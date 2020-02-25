package folder

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// IsGoFile returns true if a file name is a go file
func IsGoFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".go")
}

// ListGoFilesRecursively returns a list of go files in a directory recursively
func ListGoFilesRecursively(dir string, blacklist []*regexp.Regexp) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && IsGoFile(info.Name()) && !shouldIgnore(filePath, blacklist) {
			files = append(files, filePath)
		}
		return err
	})
	return files, err
}

// ListGoFiles list all files in directory
func ListGoFiles(dir string, blacklist []*regexp.Regexp) ([]string, error) {
	files := make([]string, 0)
	filesInDir, err := ioutil.ReadDir(dir)
	if err != nil {
		return files, err
	}
	for _, info := range filesInDir {
		if !info.IsDir() && IsGoFile(info.Name()) {
			files = append(files, info.Name())
		}
	}
	return files, err
}

func shouldIgnore(filePath string, blacklist []*regexp.Regexp) bool {
	for _, r := range blacklist {
		if r.MatchString(filePath) {
			return true
		}
	}
	return false
}
