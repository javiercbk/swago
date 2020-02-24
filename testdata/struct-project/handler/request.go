package handler

type (
	V1Request1 struct {
		Cool string `json:"handler1Str"`
	}
	V1Request2 struct {
		Int int `json:"handler2Int"`
	}
	V1Request3 struct {
		Name string `json:"name"`
	}
)
