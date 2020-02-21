package security

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// JWTUser is the data being encoded in the JWT token
type JWTUser struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Expiry    time.Time `json:"expiry"`
}

// JWTMiddlewareFactory creates a JWTMiddleware
func JWTMiddlewareFactory(jwtSecret string, optional bool) echo.MiddlewareFunc {
	jwtMiddleware := middleware.JWTWithConfig(middleware.JWTConfig{
		SigningKey: []byte(jwtSecret),
	})
	jwt.TimeFunc = time.Now().UTC
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := jwtMiddleware(next)(c)
			// only if the error was ErrJWTMissing retry the request
			if errors.Is(err, middleware.ErrJWTMissing) && optional {
				// if it failed to find the JWTToken, then continue
				// if and only if the user is optional
				return next(c)
			}
			return err
		}
	}
}

// JWTEncode encodes a user into a jwt.MapClaims
func JWTEncode(user JWTUser, d time.Duration) jwt.MapClaims {
	claims := jwt.MapClaims{}
	return claims
}

// JWTDecode attempt to decode a user
func JWTDecode(c echo.Context, jwtUser *JWTUser) error {
	return nil
}
