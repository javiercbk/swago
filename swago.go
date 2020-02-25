package swago

import (
	"bufio"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-swagger/go-swagger/spec"
	"github.com/javiercbk/swago/ast"
	"github.com/javiercbk/swago/criteria"
)

type modelType string

type generatorErr string

type analisisFunc = func(f ast.Function) error

func (m generatorErr) Error() string {
	return string(m)
}

const (
	modFile = "go.mod"
	// ErrMainNotFound is returned when a main function is not found
	ErrFuncNotFound generatorErr = "starting point not found"
)

var (
	defaultIgnoreList = []*regexp.Regexp{
		regexp.MustCompile(".*_test\\.go"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "testdata" + string(os.PathSeparator) + ".*"),
	}
)

func isGoFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".go")
}

func listGoFiles(dir string, ignoreList []*regexp.Regexp) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err == nil && isGoFile(info.Name()) && !shouldIgnore(filePath, ignoreList) {
			files = append(files, filePath)
		}
		return err
	})
	return files, err
}

func shouldIgnore(filePath string, ignoreList []*regexp.Regexp) bool {
	for _, r := range ignoreList {
		return r.MatchString(filePath)
	}
	return false
}

// SwaggerGenerator is able to navigate a project's code
type SwaggerGenerator struct {
	moduleName     string
	rootPath       string
	goPath         string
	projectGoFiles []string
	ignoreList     []*regexp.Regexp
	logger         *log.Logger
	astManager     ast.Manager
	routes         []ast.Route
	structs        map[string]ast.StructDef
}

// GenerateSwaggerDoc generates the swagger documentation
func (e *SwaggerGenerator) GenerateSwaggerDoc(projectCriterias criteria.Criteria) (*spec.Swagger, error) {
	for i := range projectCriterias.Routes {
		if projectCriterias.Routes[i].StructRoute != nil {
			err := e.initStructRoute(projectCriterias.Routes[i].StructRoute)
			if err != nil {
				e.logger.Printf("error finding structs: %v\n", err)
				return nil, err
			}
			err = e.findStructCompositeLiteral(projectCriterias.Routes[i].StructRoute)
			if err != nil {
				e.logger.Printf("error searching for struct route: %v\n", err)
				return nil, err
			}
		}
	}
	for i := range projectCriterias.Routes {
		if projectCriterias.Routes[i].FuncRoute != nil {
			f := ast.Function{
				Name: "main",
			}
			err := e.findFunction(&f)
			if err != nil {
				return nil, ErrFuncNotFound
			}
			err = e.breadthFirstExplore(&f, e.findRouteFactory(projectCriterias.Routes))
			if err != nil {
				e.logger.Printf("error finding routes: %v\n", err)
				return nil, err
			}
		}
	}
	for i := range e.routes {
		err := e.resolvePathValue(&e.routes[i])
		if err != nil {
			e.logger.Printf("error resolving path values: %v\n", err)
			return err
		}
		err = e.findHandlerDeclaration(&e.routes[i])
		if err != nil {
			e.logger.Printf("error finding handler declaration: %v\n", err)
			return err
		}

	}
	return nil, nil
}

func (e *SwaggerGenerator) resolveRoute(structRoute *criteria.StructRoute, route *ast.Route) error {
	for k, v := range route.Struct.Values {
		if structRoute.PathField == k {
			route.Path = v
		} else if structRoute.HandlerField == k {
			route.Handler = ast.Function{
				Name:      v.Name,
				Hierarchy: v.Hierarchy,
			}
		} else if structRoute.HTTPMethodField == k {
			methodName := v.Name
			if v.Value != "" {
				methodName = v.Value
			}
			route.HTTPMethod = criteria.MatchHTTPMethod(methodName)
		}
	}
	return nil
}

func (e *SwaggerGenerator) breadthFirstExplore(f *ast.Function, analyze analisisFunc) error {
	err := analyze(*f)
	if err != nil {
		e.logger.Printf("error analyzing function %s: %v\n", f.Name, err)
		return err
	}
	functions := e.astManager.ExtractFuncCalls(*f)
	for i := range functions {
		err := e.findFunction(&functions[i])
		if err != nil {
			return err
		}
		e.breadthFirstExplore(&functions[i], analyze)
	}
	return nil
}

func (e *SwaggerGenerator) findFunction(f *ast.Function) error {
	for _, goFile := range e.projectGoFiles {
		err := e.astManager.FindFuncDeclaration(goFile, f)
		if err == nil {
			return nil
		} else if err != ast.ErrNotFound {
			return err
		}
	}
	return ast.ErrNotFound
}

func (e *SwaggerGenerator) resolvePathValue(r *ast.Route) error {
	if len(r.Path.Value) == 0 {
		foldersToCheck := make([]string, 1, 2)
		// Path is a variable, we need to get the actual value
		if len(r.Path.Hierarchy) == 0 {
			foldersToCheck[0] = e.resolveImportPath(r.Path.Hierarchy)
		}
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
			goFiles, err := listGoFiles(folderToCheck, e.ignoreList)
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
	}
	return nil
}

func (e *SwaggerGenerator) findHandlerDeclaration(r *ast.Route) error {
	if len(r.Handler.Hierarchy) == 0 {
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
			goFiles, err := listGoFiles(d, e.ignoreList)
			if err != nil {
				e.logger.Printf("error listing go files for directory %s: %v\n", d, err)
				return err
			}
			for _, goFile := range goFiles {
				e.astManager.FindFuncDeclaration(goFile, &r.Handler)
			}
		}
	} else {
		//TODO: find handler in this package
	}
	return nil
}

func (e *SwaggerGenerator) findRouteFactory(criterias []criteria.RouteCriteria) analisisFunc {
	return func(f ast.Function) error {
		return e.findRoutes(f, criterias)
	}
}

// findRoutes attempts to find all the routes in a project folder
func (e *SwaggerGenerator) findRoutes(f ast.Function, criterias []criteria.RouteCriteria) error {
	e.logger.Printf("searching all go files in directory %s recursively\n", e.rootPath)
	routesForFile := e.astManager.ExtractRoutes(&f, criterias)
	e.routes = append(e.routes, routesForFile...)
	return nil
}

func (e *SwaggerGenerator) findRequestModel(r *ast.Route, callCriterias []criteria.CallCriteria) error {
	id := ast.Variable{}
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

func (e *SwaggerGenerator) findStructDeclaration() {}

func (e *SwaggerGenerator) dotImportsForFile(filePath string) ([]string, error) {
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

func (e *SwaggerGenerator) resolveImportPath(pkgName string) string {
	if strings.HasPrefix(pkgName, e.moduleName) {
		return path.Join(e.rootPath, pkgName[len(e.moduleName):])
	}
	return path.Join(e.goPath, pkgName)
}

// NewSwaggerGenerator creates a swagger generator that scans a whole project
func NewSwaggerGenerator(rootPath, goPath string, logger *log.Logger) (*SwaggerGenerator, error) {
	var err error
	ignoreList := defaultIgnoreList
	generator := &SwaggerGenerator{
		rootPath:   rootPath,
		goPath:     goPath,
		ignoreList: ignoreList,
		logger:     logger,
		structs:    make(map[string]ast.StructDef),
	}
	generator.projectGoFiles, err = listGoFiles(rootPath, generator.ignoreList)
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
