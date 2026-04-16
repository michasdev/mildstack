package http

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/michasdev/mildstack/core/internal/application/orchestrator"
)

var (
	ErrInvalidRoute   = errors.New("http registrar: invalid route")
	ErrDuplicateRoute = errors.New("http registrar: duplicate route")
)

type Registrar struct {
	mu       sync.RWMutex
	services map[string]*registeredService
}

type registeredService struct {
	routes    []RegisteredRoute
	routeKeys map[string]struct{}
	nameKeys  map[string]struct{}
}

type ServiceCatalogEntry struct {
	Name   string
	Routes []RegisteredRoute
}

type RegisteredRoute struct {
	Method string
	Path   string
	Name   string
}

func NewRegistrar() *Registrar {
	return &Registrar{
		services: make(map[string]*registeredService),
	}
}

func (r *Registrar) Register(route orchestrator.Route) error {
	serviceName, normalized, err := normalizeRoute(route)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	service := r.services[serviceName]
	if service == nil {
		service = &registeredService{
			routeKeys: make(map[string]struct{}),
			nameKeys:  make(map[string]struct{}),
		}
		r.services[serviceName] = service
	}

	methodPathKey := routeKey(normalized.Method, normalized.Path)
	if _, exists := service.routeKeys[methodPathKey]; exists {
		return fmt.Errorf("%w: %s %s", ErrDuplicateRoute, normalized.Method, normalized.Path)
	}
	if _, exists := service.nameKeys[normalized.Name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRoute, normalized.Name)
	}

	service.routeKeys[methodPathKey] = struct{}{}
	service.nameKeys[normalized.Name] = struct{}{}
	service.routes = append(service.routes, normalized)
	return nil
}

func (r *Registrar) Services() []ServiceCatalogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]ServiceCatalogEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, ServiceCatalogEntry{
			Name:   name,
			Routes: cloneRoutes(r.services[name].routes),
		})
	}

	return entries
}

func (r *Registrar) Service(name string) (ServiceCatalogEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, ok := r.services[name]
	if !ok {
		return ServiceCatalogEntry{}, false
	}

	return ServiceCatalogEntry{
		Name:   name,
		Routes: cloneRoutes(service.routes),
	}, true
}

func normalizeRoute(route orchestrator.Route) (string, RegisteredRoute, error) {
	method := strings.ToUpper(strings.TrimSpace(route.Method))
	if method == "" {
		return "", RegisteredRoute{}, fmt.Errorf("%w: method is empty", ErrInvalidRoute)
	}

	name := strings.TrimSpace(route.Name)
	if name == "" {
		return "", RegisteredRoute{}, fmt.Errorf("%w: name is empty", ErrInvalidRoute)
	}

	serviceName, normalizedPath, err := normalizeRoutePath(route.Path)
	if err != nil {
		return "", RegisteredRoute{}, err
	}

	return serviceName, RegisteredRoute{
		Method: method,
		Path:   normalizedPath,
		Name:   name,
	}, nil
}

func normalizeRoutePath(rawPath string) (string, string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", "", fmt.Errorf("%w: path is empty", ErrInvalidRoute)
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "", "", fmt.Errorf("%w: path must start with /", ErrInvalidRoute)
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return "", "", fmt.Errorf("%w: path contains whitespace", ErrInvalidRoute)
	}
	if strings.Contains(trimmed, "//") {
		return "", "", fmt.Errorf("%w: path contains empty segments", ErrInvalidRoute)
	}

	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return "", "", fmt.Errorf("%w: service segment is missing", ErrInvalidRoute)
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", "", fmt.Errorf("%w: path contains invalid segment %q", ErrInvalidRoute, segment)
		}
	}

	normalized := path.Join(append([]string{"/api/v1/runtime/services"}, segments...)...)
	return segments[0], normalized, nil
}

func routeKey(method, path string) string {
	return method + " " + path
}

func cloneRoutes(routes []RegisteredRoute) []RegisteredRoute {
	copied := append([]RegisteredRoute(nil), routes...)
	sort.SliceStable(copied, func(i, j int) bool {
		if copied[i].Method != copied[j].Method {
			return copied[i].Method < copied[j].Method
		}
		if copied[i].Path != copied[j].Path {
			return copied[i].Path < copied[j].Path
		}
		return copied[i].Name < copied[j].Name
	})
	return copied
}
