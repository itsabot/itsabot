package dt

import (
	"fmt"
	"path"

	"github.com/labstack/echo"
)

type HTTPRoute struct {
	Method string
	Path   string
}

type HandlerMap map[HTTPRoute]func(*echo.Context) error

type RouteHandler struct {
	Method  string
	Path    string
	Handler func(*echo.Context) error
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
