package swago

import (
	"bufio"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/javiercbk/swago/criteria"
	"github.com/javiercbk/swago/pkg"
)

type modelType string

type generatorErr string

func (m generatorErr) Error() string {
	return string(m)
}

const (
	modFile = "go.mod"
	// ErrMainNotFound is returned when a main function is not found
	ErrFuncNotFound generatorErr = "starting point not found"
)

var (
	defaultBlacklist = []*regexp.Regexp{
		regexp.MustCompile(".*_test\\.go"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "testdata" + string(os.PathSeparator) + ".*"),
	}
)

// SwaggerGenerator is able to navigate a project's code
type SwaggerGenerator struct {
	Blacklist  []*regexp.Regexp
	Pkgs       []*pkg.Pkg
	ModuleName string
	RootPath   string
	GoPath     string
	logger     *log.Logger
	routes     []pkg.Route
}

// GenerateSwaggerDoc generates the swagger documentation
func (s *SwaggerGenerator) GenerateSwaggerDoc(projectCriterias criteria.Criteria) error {
	for _, r := range projectCriterias.Routes {
		if r.StructRoute != nil {
			s.findStructRoutes(*r.StructRoute)
		}
	}
	for i := range s.routes {
		if len(s.routes[i].HandlerType) > 0 {
			parts := strings.Split(s.routes[i].HandlerType, ".")
			if len(parts) > 1 {
				pkg := parts[0]
				funcName := parts[1]
			}
			s.findRequestModelInFunc()
		}
	}
	return nil
}

func (s *SwaggerGenerator) findStructRoutes(structRoute criteria.StructRoute) []pkg.Route {
	s.routes = make([]pkg.Route, 0)
	for _, p := range s.Pkgs {
		foundRoutes := p.SearchForStructRoutes(structRoute)
		s.routes = append(s.routes, foundRoutes...)
	}
	return s.routes
}

// NewSwaggerGeneratorWithBlacklist creates a swagger generator that scans a whole project except for any matching a given blacklist
func NewSwaggerGeneratorWithBlacklist(rootPath, goPath string, logger *log.Logger, blacklist []*regexp.Regexp) (*SwaggerGenerator, error) {
	var err error
	generator := &SwaggerGenerator{
		RootPath:  rootPath,
		GoPath:    goPath,
		Blacklist: blacklist,
		logger:    logger,
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
	generator.ModuleName = moduleName
	generator.Pkgs, err = pkg.AnalizeProjectWithBlacklist(rootPath, logger, blacklist)
	return generator, err
}

// NewSwaggerGenerator creates a swagger generator that scans a whole project
func NewSwaggerGenerator(rootPath, goPath string, logger *log.Logger) (*SwaggerGenerator, error) {
	return NewSwaggerGeneratorWithBlacklist(rootPath, goPath, logger, defaultBlacklist)
}
