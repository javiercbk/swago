package swago

import (
	"bufio"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	modFile = "go.mod"
)

func isGoFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".go")
}

func listGoFiles(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err == nil && isGoFile(info.Name()) {
			files = append(files, filePath)
		}
		return err
	})
	return files, err
}

// CodeNavigator is able to navigate a project's code
type CodeNavigator struct {
	moduleName string
	rootPath   string
	goPath     string
	logger     *log.Logger
}

// NewCodeNavigator creates a code navigator that scans a whole project
func NewCodeNavigator(rootPath, goPath string, logger *log.Logger) (CodeNavigator, error) {
	navigator := CodeNavigator{
		rootPath: rootPath,
		goPath:   goPath,
		logger:   logger,
	}
	goModFilePath := path.Join(rootPath, modFile)
	logger.Printf("looking for module declaration in file %s\n", goModFilePath)
	goModFile, err := os.Open(goModFilePath)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			logger.Printf("error opening file %s: %v\n", goModFilePath, err)
			return navigator, err
		}
		// if file does not exist then is not a module
		return navigator, nil
	}
	defer goModFile.Close()
	scanner := bufio.NewScanner(goModFile)
	scanner.Split(bufio.ScanLines)
	moduleName := ""
	logger.Printf("reading file %s\n", goModFilePath)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module") {
			moduleName = line[7:]
			break
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Printf("error reading file %s: %v\n", goModFilePath, err)
	}
	navigator.moduleName = moduleName
	return navigator, nil
}
