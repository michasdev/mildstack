package orchestrator

import "context"

type Service interface {
	Start(context.Context) error
	Stop(context.Context) error
	Metadata() Metadata
	Policy() EmulationPolicy
	RegisterRoutes(RouteRegistrar) error
	AttachState(StateHook) error
}

type Metadata struct {
	Name        string
	Description string
	Version     string
	Tags        []string
}

type Route struct {
	Method string
	Path   string
	Name   string
}

type RouteRegistrar interface {
	Register(Route) error
}

type StateHook interface {
	Set(string, any)
	Get(string) (any, bool)
}
