package swago

import (
	"bufio"
	"go/token"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/javiercbk/swago/criteria"
	swagoErrors "github.com/javiercbk/swago/errors"
	"github.com/javiercbk/swago/pkg"
)

type modelType string

const (
	modFile = "go.mod"
)

var (
	defaultURLPathExtractor = regexp.MustCompile("\\{([a-zA-Z0-9]+)\\}")
	defaultBlacklist        = []*regexp.Regexp{
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
				err := s.findReqModel(pkgName, funcName, rc, &requestModel)
				if err != nil {
					return err
				}
				requestModel.CallCriteria = rc
				s.routes[i].RequestModel = requestModel
				// when a route criteria matches, do not process following callCriteria
				break
			}
			serviceResponses := make([]pkg.ServiceResponse, 0)
			for _, rc := range projectCriterias.Response {
				responses, err := s.findResModels(pkgName, funcName, rc)
				serviceResponses = append(serviceResponses, responses...)
				if err != nil {
					return err
				}
			}
			s.routes[i].ServiceResponses = serviceResponses
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
		produces := r.RequestModel.CallCriteria.Consumes
		for _, s := range r.ServiceResponses {
			if len(s.Model.CallCriteria.Produces) > 0 {
				produces = s.Model.CallCriteria.Produces
				break
			}
		}
		swaggerResponses := make(map[string]*openapi2.Response)
		for _, sResp := range r.ServiceResponses {
			err := sResp.Model.ToSwaggerSchema(nil)
			if err != nil {
				return err
			}
			httpStatusCode, err := strconv.Atoi(sResp.Code)
			if err == nil {
				// ignoring unknown codes
				swaggerResponses[sResp.Code] = &openapi2.Response{
					Description: http.StatusText(httpStatusCode),
					Schema: &openapi3.SchemaRef{
						Value: sResp.Model.Schema,
					},
				}
			}
		}
		parameters := make([]*openapi2.Parameter, 0, 2)
		urlParameters := extractURLParameters(r.Path, defaultURLPathExtractor)
		parameters = append(parameters, urlParameters...)
		parameters = append(parameters, parameter)
		swagger.AddOperation(r.Path, r.HTTPMethod, &openapi2.Operation{
			Consumes:   []string{r.RequestModel.CallCriteria.Consumes},
			Produces:   []string{produces},
			Parameters: parameters,
			Responses:  swaggerResponses,
		})
	}
	return nil
}

func (s *SwaggerGenerator) findReqModel(pkgName, funcName string, callCriteria criteria.CallCriteria, model *pkg.Struct) error {
	err := s.findReqModelInFunc(pkgName, funcName, callCriteria, model)
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

func (s *SwaggerGenerator) findResModels(pkgName, funcName string, callCriteria criteria.ResponseCallCriteria) ([]pkg.ServiceResponse, error) {
	serviceResponses, err := s.findServiceResponsesInFunc(pkgName, funcName, callCriteria)
	if err != nil && err != swagoErrors.ErrNotFound {
		return serviceResponses, err
	}
	for i := range serviceResponses {
		pkgFound := s.getPkg(serviceResponses[i].Model.PkgName)
		if pkgFound == nil {
			return serviceResponses, swagoErrors.ErrNotFound
		}
		err = pkgFound.FindStruct(&serviceResponses[i].Model)
		if err != nil {
			return serviceResponses, err
		}
	}
	return serviceResponses, nil
}

func (s *SwaggerGenerator) findStructRoutes(structRoute criteria.StructRoute) []pkg.Route {
	s.routes = make([]pkg.Route, 0)
	for _, p := range s.Pkgs {
		foundRoutes := p.SearchForStructRoutes(structRoute)
		s.routes = append(s.routes, foundRoutes...)
	}
	return s.routes
}

func (s *SwaggerGenerator) findReqModelInFunc(pkgName, funcName string, rc criteria.CallCriteria, requestModel *pkg.Struct) error {
	fun := pkg.Function{
		Name: funcName,
	}
	err := s.findFunc(pkgName, funcName, &fun)
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

func (s *SwaggerGenerator) findServiceResponsesInFunc(pkgName, funcName string, rc criteria.ResponseCallCriteria) ([]pkg.ServiceResponse, error) {
	serviceResponses := make([]pkg.ServiceResponse, 0)
	fun := pkg.Function{
		Name: funcName,
	}
	err := s.findFunc(pkgName, funcName, &fun)
	if err != nil {
		return serviceResponses, err
	}
	var lastPos token.Pos = -1
	for {
		modelResponse := pkg.ModelResponse{}
		err = fun.FindResponseCallExpressionAfter(rc, &lastPos, &modelResponse)
		if err != nil {
			break
		}
		pkgName, structName := pkg.TypeParts(modelResponse.Type)
		serviceResponses = append(serviceResponses, pkg.ServiceResponse{
			Model: pkg.Struct{
				PkgName: pkgName,
				Name:    structName,
			},
			Code: modelResponse.Code,
		})
	}
	if err != swagoErrors.ErrNotFound {
		return serviceResponses, err
	}
	return serviceResponses, nil
}

func (s *SwaggerGenerator) findFunc(pkgName, funcName string, fun *pkg.Function) error {
	pkgFound := s.getPkg(pkgName)
	if pkgFound == nil {
		return swagoErrors.ErrNotFound
	}
	err := pkgFound.FindFunc(fun)
	if err != nil {
		return err
	}
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

func extractURLParameters(path string, r *regexp.Regexp) []*openapi2.Parameter {
	foundPathParameters := make([]*openapi2.Parameter, 0)
	found := r.FindAllStringSubmatch(path, -1)
	for _, paramArr := range found {
		foundPathParameters = append(foundPathParameters, &openapi2.Parameter{
			In:   "path",
			Name: paramArr[1],
			// FIXME: detect the type somehow....maybe if the name has ID it should be an number
			// or maybe let the user pass another regexp.
			Type:     "string",
			Required: true,
		})
	}
	return foundPathParameters
}
