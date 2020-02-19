package user

import (
	"log"
	"modproj/models"
	"modproj/response"
	"strconv"

	"github.com/labstack/echo/v4"
)

const (
	userIDParamKey = "userID"
)

// QueryInput is the incoming request data
type QueryInput struct {
	UserName string `json:"username"`
}

// NewUserInput is the incoming request data
type NewUserInput struct {
	UserName string `json:"username"`
	Email    string `json:"email"`
}

// NewUserOutput is the outgoing request data
type NewUserOutput struct {
	ID       int64
	UserName string `json:"username"`
	Email    string `json:"email"`
}

// Handler is able to initialize the routes and the handlers
type Handler interface {
	Routes(e *echo.Group, middleware echo.MiddlewareFunc)
}

type handler struct {
	logger *log.Logger
}

// NewHandler creates a new handler
func NewHandler(logger *log.Logger) Handler {
	return handler{
		logger: logger,
	}
}

func (h handler) Routes(e *echo.Group, jwtMiddleware echo.MiddlewareFunc) {
	e.GET("", h.retrieveUsers, jwtMiddleware)
	e.POST("", h.createUser, jwtMiddleware)
	e.GET("/:userID", h.retrieveUser, jwtMiddleware)
	e.PUT("/:userID", h.updateUser, jwtMiddleware)
}

func (h handler) retrieveUsers(c echo.Context) error {
	queryInput := QueryInput{}
	err := c.Bind(&queryInput)
	if err != nil {
		h.logger.Printf("could not bind request data%v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	if err = c.Validate(queryInput); err != nil {
		h.logger.Printf("validation error %v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	users := make([]models.User, 0)
	return response.NewSuccessResponse(c, users)
}
func (h handler) createUser(c echo.Context) error {
	newUserInput := NewUserInput{}
	err := c.Bind(&newUserInput)
	if err != nil {
		h.logger.Printf("could not bind request data%v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	if err = c.Validate(newUserInput); err != nil {
		h.logger.Printf("validation error %v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	newUser := createUser()
	return response.NewSuccessResponse(c, newUser)
}
func (h handler) retrieveUser(c echo.Context) error {
	userID, err := parseUserID(c)
	if err != nil {
		h.logger.Printf("userID is invalid: %v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	h.logger.Printf("userID = %d\n", userID)
	newUser := createUser()
	return response.NewSuccessResponse(c, newUser)
}
func (h handler) updateUser(c echo.Context) error {
	userID, err := parseUserID(c)
	if err != nil {
		h.logger.Printf("userID is invalid: %v\n", err)
		return response.NewBadRequestResponse(c, err.Error())
	}
	h.logger.Printf("userID = %d\n", userID)
	newUser := createUser()
	return response.NewSuccessResponse(c, newUser)
}

func createUser() *models.User {
	return &models.User{}
}

func parseUserID(c echo.Context) (int64, error) {
	userIDStr := c.Param(userIDParamKey)
	eventID, err := strconv.ParseInt(userIDStr, 10, 64)
	return eventID, err
}
