package auth

import (
	"context"
	"log"
	"modproj/security"
	"time"
)

// ErrDuplicatedUser is returned when attempting to create a duplicated user
type ErrDuplicatedUser string

func (e ErrDuplicatedUser) Error() string {
	return string(e)
}

// NameTooLongErr is returned when attempting to store a first or last name that is too long
type NameTooLongErr string

func (e NameTooLongErr) Error() string {
	return string(e)
}

// EmailTooLongErr is returned when attempting to store an email that is too long
type EmailTooLongErr string

func (e EmailTooLongErr) Error() string {
	return string(e)
}

// BadCredentialsErr is returned when attempting to login with invalid credentials
type BadCredentialsErr string

func (e BadCredentialsErr) Error() string {
	return string(e)
}

// UserNotExistErr is thrown when the user does not exist anymore
type UserNotExistErr string

func (e UserNotExistErr) Error() string {
	return string(e)
}

const (
	// ErrBadCredentials is returned when attempting to login with invalid credentials
	ErrBadCredentials BadCredentialsErr = "bad credentials"
	// ErrUserNotExist is thrown when the user does not exist anymore
	ErrUserNotExist UserNotExistErr = "bad credentials"
)

// Credentials has all the data necesary to authenticate an admin
type Credentials struct {
	Email    string `json:"email" validate:"required,gt=0,lte=256"`
	Password string `json:"password" validate:"required,gt=0"`
}

// TokenResponse contains a jwt token
type TokenResponse struct {
	User  VisibleUser `json:"user"`
	Token string      `json:"token"`
}

// VisibleUser is the public data of an admin
type VisibleUser struct {
	ID        int64      `json:"id"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Expiry    time.Time  `json:"validUntil"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// APIFactory is a function capable of creating an Auth API
type APIFactory func(logger *log.Logger, jwtSecret string) API

// API is authentication API interface
type API interface {
	AuthenticateUser(ctx context.Context, credentials Credentials) (TokenResponse, error)
	UserInfo(ctx context.Context, jwtUser security.JWTUser, user *VisibleUser) error
}

type api struct {
	logger    *log.Logger
	jwtSecret string
}

// NewAPI creates a new authentication API
func NewAPI(logger *log.Logger, jwtSecret string) API {
	return api{
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

// AuthenticateUser authenticates a user and returns a token
func (api api) AuthenticateUser(ctx context.Context, credentials Credentials) (TokenResponse, error) {
	tokenResponse := TokenResponse{}
	return tokenResponse, nil
}

// UserInfo returns a visible user from a userID
func (api api) UserInfo(ctx context.Context, jwtUser security.JWTUser, visibleUser *VisibleUser) error {
	return nil
}
