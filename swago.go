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

// SwaggerGenerator is able to navigate a project's code
type SwaggerGenerator struct {
	moduleName     string
	rootPath       string
	goPath         string
	projectGoFiles []string
	logger         *log.Logger
	astManager     ast.Manager
}

// GenerateSwaggerDoc generates the swagger documentation
func (e SwaggerGenerator) GenerateSwaggerDoc(goFilePath string, projectCriterias criteria.Criteria) error {
	routes, err := e.findRoutes(projectCriterias.Routes)
	if err != nil {
		e.logger.Printf("error finding criterias: %v\n", err)
		return err
	}
	for i := range routes {
		err = e.resolvePathValue(&routes[i])
		if err != nil {
			e.logger.Printf("error resolving path values: %v\n", err)
			return err
		}
		err = e.findHandlerDeclaration(&routes[i])
		if err != nil {
			e.logger.Printf("error finding handler declaration: %v\n", err)
			return err
		}

	}
	return nil
}

func (e SwaggerGenerator) resolvePathValue(r *ast.Route) error {
	if len(r.Path.Value) == 0 {
		// Path is a variable, we need to get the actual value
		if len(r.Path.Pkg) == 0 {
			foldersToCheck := make([]string, 1, 2)
			foldersToCheck[0] = e.resolveImportPath(r.Path.Pkg)
			imports, err := e.astManager.GetFileImports(r.File)
			if err != nil {
				e.logger.Printf("error getting file imports for file %s: %v\n", r.File, err)
				return err
			}
			if len(imports) > 0 {
				for _, imp := range imports {
					foldersToCheck = append(foldersToCheck, e.resolveImportPath(imp.Pkg))
				}
			}
			dotImportsFolders, err := e.dotImportsForFile(r.File)
			if err != nil {
				e.logger.Printf("error retrieving dot imports folders for file %s: %v\n", r.File, err)
				return err
			}
			if len(dotImportsFolders) > 0 {
				foldersToCheck = append(foldersToCheck, dotImportsFolders...)
			}
			for _, folderToCheck := range foldersToCheck {
				goFiles, err := listGoFiles(folderToCheck)
				if err != nil {
					e.logger.Printf("error listing files in directory %s: %v\n", folderToCheck, err)
					return err
				}
				for _, goFile := range goFiles {
					err = e.astManager.FindValue(goFile, &r.Path)
					if err == nil {
						return nil
					} else if err != ast.ErrNotFound {
						e.logger.Printf("error findinding value on file %s: %v\n", goFile, err)
						return err
					}
				}
			}
		} else {

		}
	}
	return nil
}

func (e SwaggerGenerator) findHandlerDeclaration(r *ast.Route) error {
	if len(r.Handler.Pkg) == 0 {
		dirsToCheck := []string{path.Dir(r.File)}
		dotImportsFolders, err := e.dotImportsForFile(r.File)
		if err != nil {
			e.logger.Printf("error retrieving dot imports folders for file %s: %v\n", r.File, err)
			return err
		}
		for len(dotImportsFolders) > 0 {
			dirsToCheck = append(dirsToCheck, dotImportsFolders...)
		}
		for _, d := range dirsToCheck {
			goFiles, err := listGoFiles(d)
			if err != nil {
				e.logger.Printf("error listing go files for directory %s: %v\n", d, err)
				return err
			}
			for _, goFile := range goFiles {
				e.astManager.FindFuncDeclaration(goFile, &r.Handler)
			}
		}
	} else {

	}
	return nil
}

// findRoutes attempts to find all the routes in a project folder
func (e SwaggerGenerator) findRoutes(criterias []criteria.RouteCriteria) ([]ast.Route, error) {
	routesFound := make([]ast.Route, 0)
	e.logger.Printf("searching all go files in directory %s recursively\n", e.rootPath)
	for _, goFile := range e.projectGoFiles {
		e.logger.Printf("searching for routes in file %s\n", goFile)
		routesForFile, err := e.astManager.ExtractRoutesFromFile(goFile, criterias)
		if err != nil {
			e.logger.Printf("error extracting routes from file %s: %v\n", goFile, err)
			return routesFound, err
		}
		routesFound = append(routesFound, routesForFile...)
	}
	return routesFound, nil
}

func (e SwaggerGenerator) findRequestModel(r *ast.Route, callCriterias []criteria.CallCriteria) error {
	id := ast.Identifier{}
	for _, c := range callCriterias {
		err := e.astManager.FindCallCriteria(r.Handler, c, &id)
		if err != nil {
			if err != ast.ErrNotFound {
				e.logger.Printf("error finding call criteria in function: %v\n", err)
				return err
			}
		} else {
			break
		}
	}
	if len(id.Name) > 0 {

	}
	return nil
}

func (e SwaggerGenerator) findStructDeclaration() {}

func (e SwaggerGenerator) dotImportsForFile(filePath string) ([]string, error) {
	dotImportsDir := make([]string, 0)
	imports, err := e.astManager.GetFileImports(filePath)
	if err != nil {
		e.logger.Printf("error getting file imports for file %s: %v\n", filePath, err)
		return dotImportsDir, err
	}
	for importIndex := range imports {
		imp := imports[importIndex]
		if imp.Name == "." {
			dotImportsDir = append(dotImportsDir, e.resolveImportPath(imp.Pkg))
			break
		}
	}
	return dotImportsDir, nil
}

func (e SwaggerGenerator) resolveImportPath(pkgName string) string {
	if strings.HasPrefix(pkgName, e.moduleName) {
		return path.Join(e.rootPath, pkgName[len(e.moduleName):])
	}
	return path.Join(e.goPath, pkgName)
}

// func (e CodeExplorer) findRequestModel(r *Route, criterias []RouteCriteria) error {
// 	return nil
// }

// NewSwaggerGenerator creates a swagger generator that scans a whole project
func NewSwaggerGenerator(rootPath, goPath string, logger *log.Logger) (SwaggerGenerator, error) {
	var err error
	generator := SwaggerGenerator{
		rootPath: rootPath,
		goPath:   goPath,
		logger:   logger,
	}
	generator.projectGoFiles, err = listGoFiles(rootPath)
	if err != nil {
		logger.Printf("error getting go file list for folder %s: %v\n", rootPath, err)
		return generator, err
	}
	goModFilePath := path.Join(rootPath, modFile)
	logger.Printf("looking for module declaration in file %s\n", goModFilePath)
	goModFile, err := os.Open(goModFilePath)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			logger.Printf("error opening file %s: %v\n", goModFilePath, err)
			return generator, err
		}
		// if file does not exist then is not a module
		return generator, nil
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
		return generator, err
	}
	generator.moduleName = moduleName
	generator.astManager = ast.NewManager(logger)
	return generator, nil
}
