package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Route struct {
	Path       string
	TargetURL  string
	Methods    []string
	Middleware []Middleware
}

func (s *Server) InitializeRoutes() {
	logConfig := LoggerMiddleware{logger: s.logger}
	// TODO: Make routes configurable via Redis for live reloading
	routes := []Route{
		{
			Path:      "/api/example",
			TargetURL: "http://localhost:8081/hello",
			Middleware: []Middleware{
				logConfig.LogHandler,
				CORS,
				MethodMiddleware([]string{"GET", "POST"}),
			},
		},
	}

	for _, route := range routes {
		targetURL, err := url.Parse(route.TargetURL)
		if err != nil {
			s.logger.Fatal("invalid target URL: ", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			proxy.ServeHTTP(writer, request)
		})

		// Add middleware Tower
		towerHandler := Tower(handler, route.Middleware...)
		s.router.Handle(route.Path, towerHandler)
	}
}
