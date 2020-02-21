package main

//go:generate sqlboiler psql

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"modproj/http"
)

const (
	defaultJWTSecret = "sample"
)

func main() {
	var logFilePath, address, jwtSecret string
	flag.StringVar(&logFilePath, "l", defaultLogFilePath, "the log file location")
	flag.StringVar(&address, "a", defaultAddress, "the http server address")
	flag.StringVar(&jwtSecret, "jwt", defaultJWTSecret, "the jwt secret")
	flag.Parse()
	var logWritter io.Writer
	if logFilePath != "stdout" {
		logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("error opening lof file: %s", err)
			os.Exit(1)
		}
		defer logFile.Close()
		logWritter = logFile
	} else {
		logWritter = os.Stdout
	}
	logger := log.New(logWritter, "applog: ", log.Lshortfile|log.LstdFlags)
	cnf := http.Config{
		Address:   address,
		JWTSecret: jwtSecret,
	}
	logger.Printf("server is initializing\n")
	err = http.Serve(cnf, logger, db, contractDeployer)
	if err != nil {
		logger.Fatalf("could not start server %s\n", err)
	}
}

func connectPostgres(dbName, dbHost, dbUser, dbPass string, logger *log.Logger) (*sql.DB, error) {
	logger.Printf("connecting to postgres server\n")
	postgresOpts := fmt.Sprintf("dbname=%s host=%s user=%s password=%s sslmode=disable", dbName, dbHost, dbUser, dbPass)
	db, err := sql.Open("postgres", postgresOpts)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}
