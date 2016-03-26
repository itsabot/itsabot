package dt

import (
	"net/http"
	"path"

	"github.com/julienschmidt/httprouter"
)

// HTTPRoute defines a route to be used within a HandlerMap.
type HTTPRoute struct {
	Method string
	Path   string
}

// HandlerMap maps HTTPRoutes (the method and URL path) to a echo router
// handler.
type HandlerMap map[HTTPRoute]http.Handler

// RouteHandler is a complete struct containing both an HTTPRoute and a handler.
type RouteHandler struct {
	Method  string
	Path    string
	Handler http.Handler
}

// AddRoutes to the router dynamically, enabling drivers to add routes to an
// application at runtime usually as part of their initialization.
func (hm HandlerMap) AddRoutes(prefix string, r *httprouter.Router) {
	for httpRoute, h := range hm {
		p := path.Join("/", prefix, httpRoute.Path)
		r.Handler(httpRoute.Method, p, h)
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
