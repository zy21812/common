package resp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

var (
	runOnce sync.Once
	_echo   *Echo
)

type HandlerFunc func(c *Context) error
type Echo struct {
	echo *echo.Echo
}

func init() {
	jsoniter.RegisterTypeEncoderFunc("time.Time", func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
		t := *((*time.Time)(ptr))
		stream.WriteString(t.Format("2006-01-02 15:04:05"))
	}, func(ptr unsafe.Pointer) bool {
		return false
	})
}
func GetEcho() *Echo {
	runOnce.Do(func() {
		_echo = &Echo{echo: echo.New()}
		_echo.echo.HideBanner = true
		// e.Use(middleware.Recover())
		_echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			Skipper:      middleware.DefaultSkipper,
			AllowOrigins: []string{"*"},
			AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		}))
		_echo.echo.JSONSerializer = &JSONSerializer{}
		_echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				if err := next(c); err != nil {
					logrus.Errorf("error=>%s=>%s=>%s", err, c.Request().Method, c.Path())
					if he, ok := err.(*echo.HTTPError); ok {
						message := fmt.Sprintf("%v", he.Message)
						return c.JSON(200, map[string]interface{}{"code": he.Code, "msg": message})
					}
					code := 500
					if err.Error() == "record not found" {
						log.Println("c.Response().Status", c.Response().Status)
						code = 404
					}
					return c.JSON(200, map[string]interface{}{"code": code, "msg": err.Error()})
				}
				if c.Response().Status == 404 {
					logrus.Errorf("succ1=>%d=>%s=>%s", c.Response().Status, c.Request().Method, c.Path())
					c.Response().Status = 200
					return c.HTMLBlob(200, []byte{})
				}
				return nil
			}
		})
		_echo.Use(Auth)

	})
	return _echo
}

func (e *Echo) Static(www http.FileSystem, indexFile []byte) {
	_echo.routeNotFound("/*", func(c echo.Context) error {
		if strings.HasPrefix(c.Request().URL.Path, "/api") {
			return c.JSON(200, map[string]interface{}{"code": 404, "msg": "not found"})
		} else {
			return c.HTMLBlob(200, indexFile)
		}
	})
	assetHandler := http.FileServer(www)
	_echo.GET("/", wrapHandler(assetHandler))
	_echo.GET("/assets/*", wrapHandler(http.StripPrefix("/", assetHandler)))
}

func (e *Echo) Group(path string) *Group {
	return &Group{e.echo.Group(path)}
}
func (e *Echo) Use(middleware ...echo.MiddlewareFunc) {
	e.echo.Use(middleware...)
}
func (e *Echo) routeNotFound(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.echo.RouteNotFound(path, h, m...)
}
func (e *Echo) GET(path string, h HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.echo.GET(path, func(c echo.Context) error { return h(c.(*Context)) }, m...)
}
func (e *Echo) POST(path string, h HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.echo.POST(path, func(c echo.Context) error { return h(c.(*Context)) }, m...)
}
func wrapHandler(h http.Handler) HandlerFunc {
	return func(c *Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
func (e *Echo) Start(addr string) error {
	return e.echo.Start(addr)
}
func (e *Echo) Shutdown(ctx context.Context) error {
	return e.echo.Shutdown(ctx)
}

type Group struct {
	group *echo.Group
}

func (e *Group) GET(path string, h HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.group.GET(path, func(c echo.Context) error { return h(c.(*Context)) }, m...)
}
func (e *Group) POST(path string, h HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return e.group.POST(path, func(c echo.Context) error { return h(c.(*Context)) }, m...)
}
