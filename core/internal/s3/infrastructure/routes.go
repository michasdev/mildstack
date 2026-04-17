package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func Routes() []orchestrator.Route {
	routes := make([]orchestrator.Route, 0, len(bucketRoutes())+len(objectRoutes()))
	routes = append(routes, bucketRoutes()...)
	routes = append(routes, objectRoutes()...)
	return routes
}
