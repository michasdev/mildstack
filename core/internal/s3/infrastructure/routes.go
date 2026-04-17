package infrastructure

import "github.com/michasdev/mildstack/core/internal/application/orchestrator"

func Routes() []orchestrator.Route {
	routes := make([]orchestrator.Route, 0, len(bucketRoutes())+len(bucketSubresourceRoutes())+len(bucketEventRoutes())+len(versioningRoutes())+len(objectRoutes())+len(multipartRoutes()))
	routes = append(routes, bucketRoutes()...)
	routes = append(routes, bucketSubresourceRoutes()...)
	routes = append(routes, bucketEventRoutes()...)
	routes = append(routes, versioningRoutes()...)
	routes = append(routes, objectRoutes()...)
	routes = append(routes, multipartRoutes()...)
	return routes
}
