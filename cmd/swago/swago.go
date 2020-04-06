package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/javiercbk/swago/criteria"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-git/go-git/v5"
	"github.com/javiercbk/swago"
	"github.com/javiercbk/swago/encoding/swagger"
)

func main() {
	var dir, file, outfile string
	log := &log.Logger{}
	log.SetOutput(os.Stdout)
	flag.StringVar(&dir, "dir", "./", "Project's directory")
	flag.StringVar(&file, "conf", "./swago.yaml", "Swago's config file")
	flag.StringVar(&outfile, "outfile", "./swagger.yaml", "Swagger's file output")
	flag.Parse()
	projectPath, err := filepath.Abs(dir)
	if err != nil {
		log.Printf("error getting absolute path of %s: %v", dir, err)
		os.Exit(1)
	}
	filePath, err := filepath.Abs(file)
	if err != nil {
		log.Printf("error getting absolute path of %s: %v", file, err)
		os.Exit(1)
	}
	outfilePath, err := filepath.Abs(outfile)
	if err != nil {
		log.Printf("error getting absolute path of %s: %v", outfile, err)
		os.Exit(1)
	}
	swagoFile, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("error reading project criteria from file %s: %v", filePath, err)
		os.Exit(1)
	}
	projectCriteria := criteria.Criteria{}
	criteriaDecorer := criteria.NewCriteriaDecoder(log)
	err = criteriaDecorer.ParseCriteriaFromYAML(swagoFile, &projectCriteria)
	if err != nil {
		log.Printf("error parsing project criteria from file %s: %v", filePath, err)
		os.Exit(1)
	}
	defer swagoFile.Close()
	swaggerDoc := openapi2.Swagger{
		Info: openapi3.Info{
			Title: projectCriteria.Info.Title,
		},
	}
	if len(projectCriteria.Info.Version) == 0 {
		r, err := git.PlainOpen(projectPath)
		if err != nil {
			log.Printf("error reading git repository at '%s': %v", projectPath, err)
			os.Exit(1)
		}
		cIter, err := r.Log(&git.LogOptions{})
		if err != nil {
			log.Printf("error reading commit logs from repository at '%s': %v", projectPath, err)
			os.Exit(1)
		}
		defer cIter.Close()
		commit, err := cIter.Next()
		if err != nil {
			log.Printf("error reading commit logs from repository at '%s': %v", projectPath, err)
			os.Exit(1)
		}
		swaggerDoc.Info.Version = commit.Hash.String()
	} else {
		swaggerDoc.Info.Version = projectCriteria.Info.Version
	}
	sg, err := swago.NewSwaggerGenerator(projectPath, projectPath, projectCriteria.VendorFolders, log)
	if err != nil {
		log.Printf("error creating a swagger generator: %v", err)
		os.Exit(1)
	}
	err = sg.GenerateSwaggerDoc(projectCriteria, &swaggerDoc)
	if err != nil {
		log.Printf("error generating swagger doc: %v", err)
		os.Exit(1)
	}
	outFile, err := os.OpenFile(outfilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Printf("error opening outfile at path %s: %v", outfilePath, err)
		os.Exit(1)
	}
	defer outFile.Close()
	err = swagger.MarshalYAML(swaggerDoc, outFile)
	if err != nil {
		log.Printf("error marshaling swagger yaml to path %s: %v", outfilePath, err)
		os.Exit(1)
	}

}
