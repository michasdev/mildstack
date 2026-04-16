package http

import (
	"context"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/application/runtime"
)

type Snapshotter interface {
	Snapshot(context.Context) runtime.Snapshot
}

type Config struct {
	BasePath string
}

func DefaultConfig() Config {
	return Config{BasePath: "/api/v1"}
}

type Router struct {
	engine      *gin.Engine
	config      Config
	snapshotter Snapshotter
	basePath    string
	runtimePath string
}

func NewRouter(config Config, snapshotter Snapshotter) *Router {
	config = normalizeConfig(config)

	engine := gin.New()
	engine.Use(gin.Recovery())

	base := engine.Group(config.BasePath)
	runtimeGroup := base.Group("runtime")

	health := newHealthHandler(snapshotter)
	info := newRuntimeHandler(snapshotter)
	runtimeGroup.GET("/health", health.handleHealth)
	runtimeGroup.GET("/ready", health.handleReady)
	runtimeGroup.GET("/info", info.handleInfo)

	return &Router{
		engine:      engine,
		config:      config,
		snapshotter: snapshotter,
		basePath:    config.BasePath,
		runtimePath: path.Join(config.BasePath, "runtime"),
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) BasePath() string {
	return r.basePath
}

func (r *Router) RuntimePath() string {
	return r.runtimePath
}

func normalizeConfig(config Config) Config {
	config.BasePath = normalizeBasePath(config.BasePath)
	return config
}

func normalizeBasePath(basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return DefaultConfig().BasePath
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" {
		return DefaultConfig().BasePath
	}
	return basePath
}
