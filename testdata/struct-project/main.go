package main // import "github.com/foxbroadcasting/cpe-commerce-paypal"

import (
	"fmt"
	"net/http"
	r "structproj/routes"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	router := newRouter()

	startServer(router)
}

func startServer(router http.Handler) {
	s := getServerObj(router)
	// Let's shutdown gracefully

	s.ListenAndServe()
}

func getServerObj(router http.Handler) *http.Server {
	listenAddr := fmt.Sprintf(":%d", 8000)
	s := &http.Server{
		Addr:         listenAddr,
		Handler:      router,
		ReadTimeout:  time.Duration(5) * time.Second,
		WriteTimeout: time.Duration(5) * time.Second,
	}
	return s
}

func newRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	// Sets up all the routes the application is requesting.
	setupRoutes(router, routes)
	return router
}

func setupRoutes(router *mux.Router, routes r.Routes) {

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

}
