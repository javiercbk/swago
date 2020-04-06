package errors

type swagoError string

func (s swagoError) Error() string {
	return string(s)
}

const (
	// ErrNotFound is an error that is returned when an entity was not found
	ErrNotFound swagoError = "entity not found"
)
