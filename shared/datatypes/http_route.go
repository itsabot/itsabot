package dt

import (
	"fmt"
	"path"

	"github.com/labstack/echo"
)

// HTTPRoute defines a route to be used within a HandlerMap.
type HTTPRoute struct {
	Method string
	Path   string
}

// HandlerMap maps HTTPRoutes (the method and URL path) to a echo router
// handler.
type HandlerMap map[HTTPRoute]echo.Handler

// RouteHandler is a complete struct containing both an HTTPRoute and a handler.
type RouteHandler struct {
	Method  string
	Path    string
	Handler echo.Handler
}

// AddRoutes to the Echo router dynamically, enabling drivers to add routes to
// an application at runtime usually as part of their initialization. AddRoutes
// panics if the HTTP method in the HandlerMap is unknown (i.e. not GET, POST,
// PUT, PATCH, or DELETE).
func (hm HandlerMap) AddRoutes(prefix string, e *echo.Echo) {
	for httpRoute, h := range hm {
		p := path.Join("/", prefix, httpRoute.Path)
		switch httpRoute.Method {
		case echo.GET:
			e.Get(p, h)
		case echo.POST:
			e.Post(p, h)
		case echo.PUT:
			e.Put(p, h)
		case echo.PATCH:
			e.Patch(p, h)
		case echo.DELETE:
			e.Delete(p, h)
		default:
			panic(fmt.Errorf("unrecognized HTTP method: %s",
				httpRoute.Method))
		}
	}
}

// NewHandlerMap builds a HandlerMap from a slice of RouteHandlers. This is a
// convenience function, since RouteHandlers directly is very verbose for
// plugins.
func NewHandlerMap(rhs []RouteHandler) HandlerMap {
	hm := HandlerMap{}
	for _, rh := range rhs {
		route := HTTPRoute{
			Path:   rh.Path,
			Method: rh.Method,
		}
		hm[route] = rh.Handler
	}
	return hm
}
