package http

import (
	"context"
	"fmt"
	"keysight/laas/controller/config"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var log = config.GetLogger("http")

// Route defines the parameters for an api endpoint.
type Route struct {
	Name    string
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Controller creates a set of HTTP routes.
type HttpController interface {
	Routes() []Route
}

// AppendRoutes appends the routes of one or more Controllers to a mux.Router.
// If a nil router is passed, a new router will be created here.
func AppendRoutes(router *mux.Router, controllers ...HttpController) *mux.Router {
	if router == nil {
		router = mux.NewRouter()
	}

	for _, controller := range controllers {
		for _, route := range controller.Routes() {
			router.
				Methods(route.Method).
				Path(route.Path).
				Name(route.Name).
				Handler(route.Handler)
		}
	}
	return router
}

// ServeHTTP spawns HTTP server in background
func ServeHTTP(stopChan chan bool) chan error {
	errChan := make(chan error)

	configHandler := NewConfigurationHandler()

	controllers := []HttpController{
		configHandler.GetController(),
	}
	apiRouter := AppendRoutes(nil, controllers...)

	cfg := config.Config

	// serve docs through a static entry
	apiRouter.
		PathPrefix("/docs").
		Handler(http.StripPrefix("/docs", http.FileServer(http.Dir(*cfg.WebDir+"/docs"))))

	addr := fmt.Sprintf(":%d", *cfg.HTTPPort)
	server := &http.Server{
		Addr: addr, Handler: apiRouter,
		// TLSConfig: &tls.Config{
		// 	MinVersion: tls.VersionTLS12,
		// },
	}

	go stopHandler(server, errChan, stopChan)

	go serveHTTPS(server, errChan)

	return errChan
}

func serveHTTPS(srv *http.Server, errChan chan error) {
	log.Info().
		Str("addr", srv.Addr).
		Msg("HTTP Server started")
	err := srv.ListenAndServe()

	errChan <- fmt.Errorf("could not start HTTP server: %v", err)
}

func stopHandler(srv *http.Server, errChan chan error, stopChan chan bool) {
	// wait for any stop signal
	<-stopChan

	log.Warn().Msg("Shutting down HTTP server")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(*config.Config.TerminationTimeoutSeconds)*time.Second,
	)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		errChan <- fmt.Errorf("could not shut down HTTP server: %v", err)
	}

	log.Info().Msg("Successfully shut down HTTP server")

	// signal that this routine is done
	errChan <- nil
}
