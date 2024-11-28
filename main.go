package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	ListenAddr     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int
}

type Server struct {
	Config
	router *http.ServeMux
	logger *zap.SugaredLogger
}

func initLogger() (*zap.SugaredLogger, error) {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel) // Set debug level

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger.Sugar(), nil
}

func NewServer(cfg Config, logger zap.SugaredLogger) *Server {
	return &Server{
		Config: cfg,
		router: http.NewServeMux(),
		logger: &logger,
	}
}

func (s *Server) Start() error {
	server := &http.Server{
		Addr:           s.ListenAddr,
		Handler:        s.router,
		ReadTimeout:    s.ReadTimeout,
		WriteTimeout:   s.WriteTimeout,
		IdleTimeout:    s.IdleTimeout,
		MaxHeaderBytes: s.MaxHeaderBytes,
	}

	go func() {
		s.logger.Info("Starting server on port ", s.ListenAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatal("Server failed: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(ctx)
}

func main() {
	cfg := Config{
		ListenAddr:     ":8080",
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1mb
	}

	logger, err := initLogger()
	if err != nil {
		panic("initializing logger")
	}

	server := NewServer(cfg, *logger)
	server.InitializeRoutes()

	if err := server.Start(); err != nil {
		logger.Fatal("starting server")
	}
}
