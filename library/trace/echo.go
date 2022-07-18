package trace

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func (t *Tracer) Echo(e *echo.Echo) *EchoWrapper {
	return &EchoWrapper{t, e}
}

type EchoWrapper struct {
	t *Tracer
	*echo.Echo
}

func (ew *EchoWrapper) trace(method string, path string) echo.MiddlewareFunc {
	path = fmt.Sprintf("%s %s", method, path)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			c.SetRequest(
				c.Request().WithContext(
					context.WithValue(c.Request().Context(), pathKey, path),
				),
			)

			start := time.Now()
			if err = next(c); err != nil {
				c.Error(err)
			}
			duration := time.Since(start)

			rm := &requestMetric{
				path:     path,
				duration: float64(duration.Microseconds()) / 1000,
			}
			go func() {
				ew.t.rmsMutex.Lock()
				ew.t.rms = append(ew.t.rms, rm)
				ew.t.rmsMutex.Unlock()
			}()
			return
		}
	}
}

func (ew *EchoWrapper) CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.CONNECT(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodConnect, path)}, m...)...)
}

func (ew *EchoWrapper) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.DELETE(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodDelete, path)}, m...)...)
}

func (ew *EchoWrapper) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.GET(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodGet, path)}, m...)...)
}

func (ew *EchoWrapper) HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.HEAD(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodHead, path)}, m...)...)
}

func (ew *EchoWrapper) OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.OPTIONS(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodOptions, path)}, m...)...)
}

func (ew *EchoWrapper) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.PATCH(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodPatch, path)}, m...)...)
}

func (ew *EchoWrapper) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.POST(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodPost, path)}, m...)...)
}

func (ew *EchoWrapper) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.PUT(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodPut, path)}, m...)...)
}

func (ew *EchoWrapper) TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return ew.Echo.TRACE(path, h, append([]echo.MiddlewareFunc{ew.trace(http.MethodTrace, path)}, m...)...)
}

func (ew *EchoWrapper) Static(pathPrefix string, fsRoot string) *echo.Route {
	subFs := echo.MustSubFS(ew.Echo.Filesystem, fsRoot)
	return ew.Echo.Add(
		http.MethodGet,
		pathPrefix+"*",
		echo.StaticDirectoryHandler(subFs, false),
		ew.trace(http.MethodGet, pathPrefix),
	)
}

// type EchoInterface interface {
// 	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
// 	Static(pathPrefix string, fsRoot string) *echo.Route
// }
