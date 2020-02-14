package main

import (
	"fmt"
	"strings"

	"github.com/javiercbk/swago"
)

func main() {
	// swago.AnalyzeDirectory("/home/javier/go/src/github.com/foxbroadcasting/cpe-commerce-paypal")
	// swago.AnalyzeDirectory("/home/javier/workspace/fox/ppv-crypto/server")
	goCode := `package event

	import (
		"github.com/labstack/echo/v4"
	)
	
	func Routes(e *echo.Group) {
		get := e.GET
		get("", func(c echo.Context) error {
			return nil
		})
	}
	

	`
	r := strings.NewReader(goCode)
	err := swago.AnalyzeReader("test.go", r)
	if err != nil {
		fmt.Printf("error analyzing reader %v\n", err)
	}
}
