package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Redis struct {
	cacheDb  *redis.Client // DB 0: Request caching
	configDb *redis.Client // DB 1: Route configs
	ctx      context.Context
	logger   *zap.SugaredLogger
}

func NewRedis(logger *zap.SugaredLogger) (*Redis, error) {
	url := os.Getenv("REDIS_URL")
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	cacheOpts := *opts
	cacheOpts.DB = 0

	configOpts := *opts
	configOpts.DB = 1

	return &Redis{
		cacheDb:  redis.NewClient(&cacheOpts),
		configDb: redis.NewClient(&configOpts),
		ctx:      context.Background(),
		logger:   logger,
	}, nil
}

// db 0: caching of upstream requests
// {
//     "id": "...",
//     "name": "test",
//     "region": "US",
//     "age": 100
// }

// Cache DB
func (r *Redis) Set(key string, value interface{}, expiration time.Duration) error {
	r.logger.Debugw("setting redis key", "key", key, "expiration", expiration)
	return r.cacheDb.Set(r.ctx, key, value, expiration).Err()
}

// Cache DB
func (r *Redis) Get(key string) (string, error) {
	val, err := r.cacheDb.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key doesn't exist
	}
	return val, err
}

// Cache DB
func (r *Redis) Delete(key string) error {
	return r.cacheDb.Del(r.ctx, key).Err()
}

// db 1: configuration for routes/upstreams and auth methods
// {
//	    Path:      "/api/example",
//		TargetURL: "http://localhost:8081/hello",
//	    Middleware: []Middleware{ // Middleware gets injected in, we just need methods
//	        logConfig.LogHandler,
//			CORS,
//			MethodMiddleware([]string{"GET", "POST"}),
//		},
// },

// Header key and value used for auth. e.g: "authorization": "Bearer eyJ0...",
// "authorization": "Basic 290j...", "X-API-KEY": "1029ja...", etc.
type Auth struct {
	HeaderKey   string
	HeaderValue string
}

// If Cache.Enabled, cache upstream GET response for Cache.ExpiresIn seconds
type Cache struct {
	Enabled   bool
	ExpiresIn float32 // Time until cached item expires, in seconds
}

type Target struct {
	Url   string
	Cache Cache
}

type RouteConfig struct {
	Path    string   `json:"path"`
	Targets []string `json:"targets"`
	Methods []string `json:"methods"`
	Auth    Auth     `json:"auth"`
}

// Config DB.
// Key should be the RouteConfig.path
func (r *Redis) SetConf(key string, config RouteConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return r.configDb.Set(r.ctx, key, data, 0).Err()
}

// Config DB.
// Key should be the RouteConfig.path. Onus is on calling function to deserialize
// (unmarshal) into the correct struct type
func (r *Redis) GetConf(key string) (string, error) {
	val, err := r.configDb.Get(r.ctx, key).Result()
	if err != nil {
		return "", nil
	}
	return val, err
}
