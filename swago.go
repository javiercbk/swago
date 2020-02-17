package swago

import (
	"bufio"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/javiercbk/swago/ast"
	"github.com/javiercbk/swago/criteria"
)

type modelType string

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

// CodeExplorer is able to navigate a project's code
type CodeExplorer struct {
	moduleName string
	rootPath   string
	goPath     string
	logger     *log.Logger
	astManager ast.Manager
}

// FindRoutes attempts to find all the routes in a project folder
func (e CodeExplorer) FindRoutes(criterias []criteria.RouteCriteria) ([]ast.Route, error) {
	routesFound := make([]ast.Route, 0)
	e.logger.Printf("searching all go files in directory %s recursively\n", e.rootPath)
	projectGoFiles, err := listGoFiles(e.rootPath)
	if err != nil {
		return routesFound, err
	}
	for i := range projectGoFiles {
		goFile := projectGoFiles[i]
		e.logger.Printf("searching for routes in file %s\n", goFile)
		// routes, err := ast.searchFileForRouteCriteria(goFile, criterias)
		// if err != nil {
		// 	e.logger.Printf("error searching for route criteria in file %s: %v\n", goFile, err)
		// 	return routesFound, err
		// }
		// if len(routes) > 0 {
		// 	routesFound = append(routesFound, routes...)
		// }
	}
	return routesFound, nil
}

// func (e CodeExplorer) findRequestModel(r *Route, criterias []RouteCriteria) error {
// 	return nil
// }

// NewCodeExplorer creates a code navigator that scans a whole project
func NewCodeExplorer(rootPath, goPath string, logger *log.Logger) (CodeExplorer, error) {
	navigator := CodeExplorer{
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
