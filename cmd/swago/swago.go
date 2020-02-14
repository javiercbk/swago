package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/javiercbk/swago"
)

func main() {
	logger := log.New(os.Stdout, "", log.Llongfile)
	projectDirPath := "../../testdata"
	absProjectPath, err := filepath.Abs(projectDirPath)
	if err != nil {
		logger.Fatalf("error getting absolute path for dir %s: %v\n", projectDirPath, err)
	}
	codeExplorer, err := swago.NewCodeExplorer(absProjectPath, "", logger)
	if err != nil {
		logger.Fatalf("error creating code explorer %v\n", err)
	}
	routeCriterias := make([]swago.RouteCriteria, 0)
	_, err = codeExplorer.FindRoutes(routeCriterias)
	if err != nil {
		logger.Fatalf("error finding routes %v\n", err)
	}
}
