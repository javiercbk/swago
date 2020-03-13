package swago

import (
	"bufio"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/javiercbk/swago/criteria"
	swagoErrors "github.com/javiercbk/swago/errors"
	"github.com/javiercbk/swago/pkg"
)

type modelType string

const (
	modFile = "go.mod"
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
func (s *SwaggerGenerator) GenerateSwaggerDoc(projectCriterias criteria.Criteria, swagger *openapi2.Swagger) error {
	for _, r := range projectCriterias.Routes {
		if r.StructRoute != nil {
			s.findStructRoutes(*r.StructRoute)
		}
	}
	for i := range s.routes {
		if len(s.routes[i].HandlerType) > 0 {
			pkgName, funcName := pkg.TypeParts(s.routes[i].HandlerType)
			for _, rc := range projectCriterias.Request {
				requestModel := pkg.Struct{}
				err := s.findStructModel(pkgName, funcName, rc, &requestModel)
				if err != nil {
					return err
				}
				requestModel.CallCriteria = rc
				s.routes[i].RequestModel = requestModel
				// when a route criteria matches, do not process following callCriteria
				break
			}
			for _, rc := range projectCriterias.Response {
				responseModel := pkg.Struct{}
				err := s.findStructModel(pkgName, funcName, rc, &responseModel)
				if err != nil {
					return err
				}
				responseModel.CallCriteria = rc
				s.routes[i].ResponseModel = responseModel
				// when a route criteria matches, do not process following callCriteria
				break
			}
		}
	}
	s.completeSwagger(projectCriterias, swagger)
	return nil
}

func (s *SwaggerGenerator) completeSwagger(projectCriterias criteria.Criteria, swagger *openapi2.Swagger) error {
	swagger.BasePath = projectCriterias.BasePath
	swagger.Host = projectCriterias.Host
	for _, r := range s.routes {
		parameter := &openapi2.Parameter{
			In: "body",
		}
		err := r.RequestModel.ToSwaggerSchema(parameter)
		if err != nil {
			return err
		}
		swagger.AddOperation(r.Path, r.HTTPMethod, &openapi2.Operation{
			Consumes: []string{r.RequestModel.CallCriteria.Consumes},
			Produces: []string{r.ResponseModel.CallCriteria.Produces},
			Parameters: []*openapi2.Parameter{
				parameter,
			},
		})
	}
	return nil
}

func (s *SwaggerGenerator) findStructModel(pkgName, funcName string, callCriteria criteria.CallCriteria, model *pkg.Struct) error {
	err := s.findModelInFunc(pkgName, funcName, callCriteria, model)
	if err != nil && err != swagoErrors.ErrNotFound {
		return err
	}
	pkgFound := s.getPkg(model.PkgName)
	if pkgFound == nil {
		return swagoErrors.ErrNotFound
	}
	err = pkgFound.FindStruct(model)
	if err != nil {
		return err
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

func (s *SwaggerGenerator) findModelInFunc(pkgName, funcName string, rc criteria.CallCriteria, requestModel *pkg.Struct) error {
	pkgFound := s.getPkg(pkgName)
	if pkgFound == nil {
		return swagoErrors.ErrNotFound
	}
	fun := pkg.Function{
		Name: funcName,
	}
	err := pkgFound.FindFunc(&fun)
	if err != nil {
		return err
	}
	varType, err := fun.FindArgTypeCallExpression(rc)
	if err != nil {
		return err
	}
	pkgName, structName := pkg.TypeParts(varType)
	requestModel.PkgName = pkgName
	requestModel.Name = structName
	return nil
}

func (s *SwaggerGenerator) getPkg(name string) *pkg.Pkg {
	for _, p := range s.Pkgs {
		if p.Name == name {
			return p
		}
	}
	return nil
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
