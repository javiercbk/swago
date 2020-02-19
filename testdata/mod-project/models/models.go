package models

// User is a user
type User struct {
	ID       int64  `json:"id"`
	Password int64  `json:"-"`
	UserName string `json:"username"`
	Email    string `json:"email,omitempty"`
}
