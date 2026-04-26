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
	engine       *gin.Engine
	config       Config
	registrar    *Registrar
	snapshotter  Snapshotter
	basePath     string
	runtimePath  string
	servicesPath string
}

func NewRouter(config Config, snapshotter Snapshotter) *Router {
	config = normalizeConfig(config)

	engine := gin.New()
	engine.Use(gin.Recovery())

	registrar := NewRegistrar()

	base := engine.Group(config.BasePath)
	runtimeGroup := base.Group("runtime")

	health := newHealthHandler(snapshotter)
	info := newRuntimeHandler(snapshotter)
	services := newServicesHandler(snapshotter, registrar)
	runtimeGroup.GET("/health", health.handleHealth)
	runtimeGroup.GET("/ready", health.handleReady)
	runtimeGroup.GET("/info", info.handleInfo)
	runtimeGroup.GET("/services", services.handleIndex)
	runtimeGroup.GET("/services/:service", services.handleService)

	return &Router{
		engine:       engine,
		config:       config,
		registrar:    registrar,
		snapshotter:  snapshotter,
		basePath:     config.BasePath,
		runtimePath:  path.Join(config.BasePath, "runtime"),
		servicesPath: path.Join(config.BasePath, "runtime", "services"),
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

func (r *Router) ServicesPath() string {
	return r.servicesPath
}

func (r *Router) Registrar() *Registrar {
	return r.registrar
}

// RegisterSNSNative wires the SNS AWS-compatible native adapter middleware.
func (r *Router) RegisterSNSNative(service SNSNativeService) {
	if r == nil {
		return
	}
	RegisterSNSNativeRoutes(r.engine, service)
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
