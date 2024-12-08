package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Middleware func(http.Handler) http.Handler

// Stack middlewares. Order of middleware matters, where
// first middleware passed executes first, second executes
// second, etc. Named after similar Rust crate
func Tower(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}

type LoggerMiddleware struct {
	logger *zap.SugaredLogger
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func NewLoggerMiddleware() (*LoggerMiddleware, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel) // Set debug level

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &LoggerMiddleware{
		logger: logger.Sugar(),
	}, nil
}

func (l *LoggerMiddleware) LogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(wrw, r)

		l.logger.Info("http request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Int("status", wrw.status),
			zap.Duration("latency", time.Since(start)),
		)
	})
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if request.Method == "OPTIONS" {
			writer.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func MethodMiddleware(allowedMethods []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			// Always allow OPTIONS for CORS preflight
			if request.Method == "OPTIONS" {
				next.ServeHTTP(writer, request)
				return
			}

			// Verify method is allowed
			allowed := false
			for _, method := range allowedMethods {
				if method == request.Method {
					allowed = true
					break
				}
			}

			if !allowed {
				writer.Header().Set("Allow", strings.Join(allowedMethods, ", "))
				http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

type User struct {
	Username string
	Password string
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var u User
	json.NewDecoder(r.Body).Decode(&u)

	// TODO: adapt to reading auth from a configurable database
	// use repository pattern for DB access, with ENV variables for table to query
	if u.Username == "admin" && u.Password == "123456" {
		tokenString, err := createToken(u.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(w, "no user found")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Println(w, tokenString)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Println(w, "invalid credentials")
	}
}

func ProtectedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "missing authorization header")
		return
	}
	tokenString = tokenString[len("Bearer "):]

	err := verifyToken(tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "invalid token")
		return
	}

	fmt.Fprint(w, "access granted")
}
