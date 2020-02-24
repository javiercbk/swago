package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/javiercbk/swago"
	"github.com/javiercbk/swago/criteria"
)

func main() {
	logger := log.New(os.Stdout, "", log.Llongfile)
	projectDirPath := "../../testdata"
	absProjectPath, err := filepath.Abs(projectDirPath)
	if err != nil {
		logger.Fatalf("error getting absolute path for dir %s: %v\n", projectDirPath, err)
	}
	swaggerGenerator, err := swago.NewSwaggerGenerator(absProjectPath, "", logger)
	if err != nil {
		logger.Fatalf("error creating code explorer %v\n", err)
	}
	c := criteria.Criteria{
		Routes: make([]criteria.RouteCriteria, 0),
	}
	err = swaggerGenerator.GenerateSwaggerDoc(c)
	if err != nil {
		logger.Fatalf("error finding routes %v\n", err)
	}
}
