package mw

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// UserGroup wraps an echo.Group to provide methods that use *mw.UserContext
type UserGroup struct {
	EchoGroup *echo.Group // Renamed the embedded field to avoid name collision with the Group method
}

// NewUserGroup creates a new UserGroup instance from an existing echo.Group.
// Ensure that the echo.Group passed here has the SetUserContext middleware applied
// to it or its parent Echo instance.
func NewUserGroup(g *echo.Group) *UserGroup {
	return &UserGroup{EchoGroup: g} // Initialize the named field
}

// GET registers a new GET route with a handler that accepts *mw.UserContext.
// It internally uses mw.UserHandler to convert echo.Context to *echo.HandlerFunc.
func (ug *UserGroup) GET(path string, h func(*UserContext) error, m ...echo.MiddlewareFunc) *echo.Route {
	// Call the GET method on the explicitly named embedded field
	return ug.EchoGroup.GET(path, UserHandler(h), m...)
}

// POST registers a new POST route with a handler that accepts *mw.UserContext.
// It internally uses mw.UserHandler to convert echo.Context to *echo.HandlerFunc.
func (ug *UserGroup) POST(path string, h func(*UserContext) error, m ...echo.MiddlewareFunc) *echo.Route {
	// Call the POST method on the explicitly named embedded field
	return ug.EchoGroup.POST(path, UserHandler(h), m...)
}

// DELETE registers a new DELETE route with a handler that accepts *mw.UserContext.
// It internally uses mw.UserHandler to convert echo.Context to *echo.HandlerFunc.
func (ug *UserGroup) DELETE(path string, h func(*UserContext) error, m ...echo.MiddlewareFunc) *echo.Route {
	// Call the POST method on the explicitly named embedded field
	return ug.EchoGroup.DELETE(path, UserHandler(h), m...)
}

// Group creates a new sub-group from the UserGroup.
// This method now exists without conflict with the embedded field.
func (ug *UserGroup) Group(path string, m ...echo.MiddlewareFunc) *UserGroup {
	// Call the Group method on the explicitly named embedded field, then wrap its apiResult.
	return NewUserGroup(ug.EchoGroup.Group(path, m...))
}

// AddNamedRoute registers a named route for one or more HTTP methods with the
// same handler. The Echo route name is attached to the first registered method;
// subsequent methods share the same path/handler but no extra name (sufficient
// for router.Reverse lookups).
//
// All routes registered in the application MUST go through AddNamedRoute
// (or AddNamedRouteP), never via plain GET/POST/DELETE on the embedded
// echo.Group. The name is the contract by which views resolve URLs via
// router.Reverse.
func (ug *UserGroup) AddNamedRoute(name, path string, h func(*UserContext) error, methods ...string) {
	ug.addNamedRoute(name, path, nil, h, methods...)
}

// AddNamedRouteP variant for routes that require per-method permissions.
// Use nil/empty slice for methods without extra permission requirements.
func (ug *UserGroup) AddNamedRouteP(name, path string, h func(*UserContext) error, perms MethodPermissions) {
	i := 0
	for method, p := range perms {
		handler := h
		if len(p) > 0 {
			handler = RequirePermissions(h, p...)
		}
		r := ug.EchoGroup.Add(method, path, UserHandler(handler))
		if i == 0 {
			r.Name = name
		}
		i++
	}
}

// MethodPermissions maps HTTP methods to required permission strings.
type MethodPermissions map[string][]string

func (ug *UserGroup) addNamedRoute(name, path string, perms MethodPermissions, h func(*UserContext) error, methods ...string) {
	if len(methods) == 0 {
		methods = []string{http.MethodGet}
	}
	for i, method := range methods {
		handler := h
		if p, ok := perms[method]; ok && len(p) > 0 {
			handler = RequirePermissions(h, p...)
		}
		r := ug.EchoGroup.Add(method, path, UserHandler(handler))
		if i == 0 {
			r.Name = name
		}
	}
}
