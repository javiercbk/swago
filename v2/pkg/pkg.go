package pkg

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

var (
	ErrNoPackage = errors.New("no package name")
)

// getPackageName finds reads any non test go file from the package
// and extracts the package name out of it
func GetNameFromModule(modCacheDir, module, version, packageName string) (string, error) {
	moduleDir := path.Join(modCacheDir, module+"@"+version, packageName)
	files, err := ioutil.ReadDir(moduleDir)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") {
			fullGoFilePath := path.Join(moduleDir, f.Name())
			packageName, err := readPackageName(fullGoFilePath)
			if err == nil {
				return packageName, nil
			}
		}
	}
	return "", ErrNoPackage
}

func readPackageName(fullFilePath string) (string, error) {
	file, err := os.Open(fullFilePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSpace(line[8:]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", ErrNoPackage
}
