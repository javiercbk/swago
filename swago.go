package swago

import (
	"go/token"
	"io/ioutil"
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
	"golang.org/x/mod/modfile"
)

type modelType string

const (
	modFileName = "go.mod"
)

var (
	defaultBlacklist = []*regexp.Regexp{
		regexp.MustCompile(".*_test\\.go"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "testdata" + string(os.PathSeparator) + ".*"),
		regexp.MustCompile(".*" + string(os.PathSeparator) + "vendor" + string(os.PathSeparator) + ".*"),
	}
)

// SwaggerGenerator is able to navigate a project's code
type SwaggerGenerator struct {
	Blacklist     []*regexp.Regexp
	Pkgs          []*pkg.Pkg
	ModuleName    string
	VendorFolders []string
	module        *modfile.File
	RootPath      string
	GoPath        string
	logger        *log.Logger
	routes        []pkg.Route
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
				if err != nil {
					return err
				}
				serviceResponses = append(serviceResponses, responses...)
			}
			s.routes[i].ServiceResponses = serviceResponses
		}
	}
	s.completeSwagger(projectCriterias, swagger)
	return nil
}

func (s *SwaggerGenerator) loadExternalPkg(location string) error {
	realPath := s.externalPkgPath(location)
	if len(realPath) == 0 {
		return swagoErrors.ErrNotFound
	}
	externalPkgs, err := pkg.AnalizeProjectWithBlacklist(realPath, s.logger, s.Blacklist)
	if err != nil {
		return err
	}
	s.Pkgs = append(s.Pkgs, externalPkgs...)
	return nil
}

func (s *SwaggerGenerator) externalPkgPath(location string) string {
	var basePath string
	if len(s.ModuleName) > 0 {
		for _, r := range s.module.Require {
			modPathStr := r.Mod.Path
			if strings.Contains(location, modPathStr) {
				pkgName := ""
				if len(location) > len(modPathStr) {
					pkgName = location[len(modPathStr)+1:]
				}
				basePath = path.Join(s.GoPath, "pkg/mod", r.Mod.String(), pkgName)
				break
			}
		}
	} else {
		if s.VendorFolders != nil {
			for _, f := range s.VendorFolders {
				vendorPkg := path.Join(f, location)
				if folderExists(vendorPkg) {
					basePath = vendorPkg
				}
				break
			}
		} else {
			vendorPkg := path.Join(s.RootPath, "vendor", location)
			if folderExists(vendorPkg) {
				basePath = vendorPkg
			} else {
				goPathPkg := path.Join(s.GoPath, "src", location)
				if folderExists(goPathPkg) {
					basePath = goPathPkg
				}
			}
		}

	}
	return basePath
}

func folderExists(folderPath string) bool {
	fileStat, err := os.Stat(folderPath)
	if err != nil {
		return false
	}
	return fileStat.IsDir()
}

func (s *SwaggerGenerator) completeSwagger(projectCriterias criteria.Criteria, swagger *openapi2.Swagger) error {
	swagger.BasePath = projectCriterias.BasePath
	swagger.Host = projectCriterias.Host
	for _, r := range s.routes {
		if len(r.HandlerType) == 0 {
			// ignore routes with no handler
			continue
		}
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
		for i := range r.ServiceResponses {
			sResp := r.ServiceResponses[i]
			httpStatusCode, success := parseCode(sResp.Code)
			httpStatusCodeStr := strconv.Itoa(httpStatusCode)
			if httpStatusCode > 0 {
				if success {
					err := sResp.Model.ToSwaggerSchema(nil)
					if err != nil {
						return err
					}
					if err == nil {
						// ignoring unknown codes
						swaggerResponses[httpStatusCodeStr] = &openapi2.Response{
							Description: http.StatusText(httpStatusCode),
							Schema: &openapi3.SchemaRef{
								Value: sResp.Model.Schema,
							},
						}
					}

				} else {
					swaggerResponses[httpStatusCodeStr] = &openapi2.Response{
						Description: http.StatusText(httpStatusCode),
						Schema: &openapi3.SchemaRef{
							Value: projectCriterias.ErrorResponse,
						},
					}
				}
			}
		}
		parameters := make([]*openapi2.Parameter, 0, 2)
		urlParameters := extractNamedPathVarParameters(r.Path, r.NamedPathVarExtractor)
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
		if len(serviceResponses[i].Model.Name) > 0 {
			pkgFound := s.getPkg(serviceResponses[i].Model.PkgName)
			if pkgFound == nil {
				return serviceResponses, swagoErrors.ErrNotFound
			}
			err = pkgFound.FindStruct(&serviceResponses[i].Model)
			if err != nil {
				return serviceResponses, err
			}
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
	if rc.ParamIndex > 0 {
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
	} else {
		// FIXME: should be done in the same loop as above
		for {
			modelResponse := pkg.ModelResponse{}
			err = fun.FindErrorResponseCallExpressionAfter(rc, &lastPos, &modelResponse)
			if err != nil {
				break
			}
			serviceResponses = append(serviceResponses, pkg.ServiceResponse{
				Code: modelResponse.Code,
			})
		}
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
func NewSwaggerGeneratorWithBlacklist(rootPath, goPath string, vendorFolders []string, logger *log.Logger, blacklist []*regexp.Regexp) (*SwaggerGenerator, error) {
	var err error
	generator := &SwaggerGenerator{
		RootPath:  rootPath,
		GoPath:    goPath,
		Blacklist: blacklist,
		logger:    logger,
	}
	goModFilePath := path.Join(rootPath, modFileName)
	logger.Printf("looking for module declaration in file %s\n", goModFilePath)
	goModFile, err := os.Open(goModFilePath)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			logger.Printf("error opening file %s: %v\n", goModFilePath, err)
			return generator, err
		}
	} else {
		defer goModFile.Close()
		goModBytes, err := ioutil.ReadAll(goModFile)
		if err != nil {
			return generator, err
		}
		module, err := modfile.Parse(goModFilePath, goModBytes, nil)
		if err != nil {
			return generator, err
		}
		generator.ModuleName = module.Syntax.Name
		generator.module = module
	}
	generator.Pkgs, err = pkg.AnalizeProjectWithBlacklist(rootPath, logger, blacklist)
	if err != nil {
		return generator, err
	}
	project := pkg.Project{
		Pkgs:      generator.Pkgs,
		RootPath:  rootPath,
		Blacklist: blacklist,
	}
	for i := range generator.Pkgs {
		generator.Pkgs[i].Project = &project
	}
	generator.VendorFolders = vendorFolders
	return generator, nil
}

// NewSwaggerGenerator creates a swagger generator that scans a whole project
func NewSwaggerGenerator(rootPath, goPath string, vendorFolders []string, logger *log.Logger) (*SwaggerGenerator, error) {
	return NewSwaggerGeneratorWithBlacklist(rootPath, goPath, vendorFolders, logger, defaultBlacklist)
}

func extractNamedPathVarParameters(path string, r *regexp.Regexp) []*openapi2.Parameter {
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

func parseCode(code string) (int, bool) {
	// FIXME: this function should be configurable
	codeInt, err := strconv.Atoi(code)
	if err != nil {
		if strings.Contains(code, "StatusContinue") {
			codeInt = 100
		}
		if strings.Contains(code, "StatusSwitchingProtocols") {
			codeInt = 101
		}
		if strings.Contains(code, "StatusProcessing") {
			codeInt = 102
		}
		if strings.Contains(code, "StatusEarlyHints") {
			codeInt = 103
		}
		if strings.Contains(code, "StatusOK") {
			codeInt = 200
		}
		if strings.Contains(code, "StatusCreated") {
			codeInt = 201
		}
		if strings.Contains(code, "StatusAccepted") {
			codeInt = 202
		}
		if strings.Contains(code, "StatusNonAuthoritativeInfo") {
			codeInt = 203
		}
		if strings.Contains(code, "StatusNoContent") {
			codeInt = 204
		}
		if strings.Contains(code, "StatusResetContent") {
			codeInt = 205
		}
		if strings.Contains(code, "StatusPartialContent") {
			codeInt = 206
		}
		if strings.Contains(code, "StatusMultiStatus") {
			codeInt = 207
		}
		if strings.Contains(code, "StatusAlreadyReported") {
			codeInt = 208
		}
		if strings.Contains(code, "StatusIMUsed") {
			codeInt = 226
		}
		if strings.Contains(code, "StatusMultipleChoices") {
			codeInt = 300
		}
		if strings.Contains(code, "StatusMovedPermanently") {
			codeInt = 301
		}
		if strings.Contains(code, "StatusFound") {
			codeInt = 302
		}
		if strings.Contains(code, "StatusSeeOther") {
			codeInt = 303
		}
		if strings.Contains(code, "StatusNotModified") {
			codeInt = 304
		}
		if strings.Contains(code, "StatusUseProxy") {
			codeInt = 305
		}
		if strings.Contains(code, "StatusTemporaryRedirect") {
			codeInt = 307
		}
		if strings.Contains(code, "StatusPermanentRedirect") {
			codeInt = 308
		}
		if strings.Contains(code, "StatusBadRequest") {
			codeInt = 400
		}
		if strings.Contains(code, "StatusUnauthorized") {
			codeInt = 401
		}
		if strings.Contains(code, "StatusPaymentRequired") {
			codeInt = 402
		}
		if strings.Contains(code, "StatusForbidden") {
			codeInt = 403
		}
		if strings.Contains(code, "StatusNotFound") {
			codeInt = 404
		}
		if strings.Contains(code, "StatusMethodNotAllowed") {
			codeInt = 405
		}
		if strings.Contains(code, "StatusNotAcceptable") {
			codeInt = 406
		}
		if strings.Contains(code, "StatusProxyAuthRequired") {
			codeInt = 407
		}
		if strings.Contains(code, "StatusRequestTimeout") {
			codeInt = 408
		}
		if strings.Contains(code, "StatusConflict") {
			codeInt = 409
		}
		if strings.Contains(code, "StatusGone") {
			codeInt = 410
		}
		if strings.Contains(code, "StatusLengthRequired") {
			codeInt = 411
		}
		if strings.Contains(code, "StatusPreconditionFailed") {
			codeInt = 412
		}
		if strings.Contains(code, "StatusRequestEntityTooLarge") {
			codeInt = 413
		}
		if strings.Contains(code, "StatusRequestURITooLong") {
			codeInt = 414
		}
		if strings.Contains(code, "StatusUnsupportedMediaType") {
			codeInt = 415
		}
		if strings.Contains(code, "StatusRequestedRangeNotSatisfiable") {
			codeInt = 416
		}
		if strings.Contains(code, "StatusExpectationFailed") {
			codeInt = 417
		}
		if strings.Contains(code, "StatusTeapot") {
			codeInt = 418
		}
		if strings.Contains(code, "StatusMisdirectedRequest") {
			codeInt = 421
		}
		if strings.Contains(code, "StatusUnprocessableEntity") {
			codeInt = 422
		}
		if strings.Contains(code, "StatusLocked") {
			codeInt = 423
		}
		if strings.Contains(code, "StatusFailedDependency") {
			codeInt = 424
		}
		if strings.Contains(code, "StatusTooEarly") {
			codeInt = 425
		}
		if strings.Contains(code, "StatusUpgradeRequired") {
			codeInt = 426
		}
		if strings.Contains(code, "StatusPreconditionRequired") {
			codeInt = 428
		}
		if strings.Contains(code, "StatusTooManyRequests") {
			codeInt = 429
		}
		if strings.Contains(code, "StatusRequestHeaderFieldsTooLarge") {
			codeInt = 431
		}
		if strings.Contains(code, "StatusUnavailableForLegalReasons") {
			codeInt = 451
		}
		if strings.Contains(code, "StatusInternalServerError") {
			codeInt = 500
		}
		if strings.Contains(code, "StatusNotImplemented") {
			codeInt = 501
		}
		if strings.Contains(code, "StatusBadGateway") {
			codeInt = 502
		}
		if strings.Contains(code, "StatusServiceUnavailable") {
			codeInt = 503
		}
		if strings.Contains(code, "StatusGatewayTimeout") {
			codeInt = 504
		}
		if strings.Contains(code, "StatusHTTPVersionNotSupported") {
			codeInt = 505
		}
		if strings.Contains(code, "StatusVariantAlsoNegotiates") {
			codeInt = 506
		}
		if strings.Contains(code, "StatusInsufficientStorage") {
			codeInt = 507
		}
		if strings.Contains(code, "StatusLoopDetected") {
			codeInt = 508
		}
		if strings.Contains(code, "StatusNotExtended") {
			codeInt = 510
		}
		if strings.Contains(code, "StatusNetworkAuthenticationRequired") {
			codeInt = 511
		}
	}
	isSuccessful := false
	if codeInt >= 200 && codeInt <= 299 {
		isSuccessful = true
	}
	return codeInt, isSuccessful
}
