package mw

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AnonGroup wraps an echo.Group for the anonymous (non-JWT) routes.
// All routes in the app must be registered through AddNamedRoute /
// AddNamedAnonRoute so views/handlers can resolve them via router.Reverse.
type AnonGroup struct {
	EchoGroup *echo.Group
}

func NewAnonGroup(g *echo.Group) *AnonGroup {
	return &AnonGroup{EchoGroup: g}
}

// Group creates a named sub-group (path prefix). The sub-group still
// supports AddNamedRoute / AddNamedAnonRoute.
func (ag *AnonGroup) Group(path string, m ...echo.MiddlewareFunc) *AnonGroup {
	return NewAnonGroup(ag.EchoGroup.Group(path, m...))
}

// AddNamedRoute registers a plain echo handler (echo.HandlerFunc) under a
// named route. The first registered method carries Route.Name.
func (ag *AnonGroup) AddNamedRoute(name, path string, h echo.HandlerFunc, methods ...string) {
	if len(methods) == 0 {
		methods = []string{http.MethodGet}
	}
	for i, method := range methods {
		r := ag.EchoGroup.Add(method, path, h)
		if i == 0 {
			r.Name = name
		}
	}
}

// AddNamedAnonRoute registers an anon-user handler (typed *AnonUserContext)
// under a named route. The handler is wrapped via SetAnonUserContext +
// AnonUserHandler so it sees the AnonUserContext that the parent middleware
// produced.
func (ag *AnonGroup) AddNamedAnonRoute(name, path string, h func(*AnonUserContext) error, methods ...string) {
	if len(methods) == 0 {
		methods = []string{http.MethodGet}
	}
	wrapped := AnonUserHandler(h)
	for i, method := range methods {
		r := ag.EchoGroup.Add(method, path, wrapped)
		if i == 0 {
			r.Name = name
		}
	}
}
