package ark

import (
	"fmt"
	"log"
	"strings"

	hz "github.com/cloudwego/hertz/pkg/app"

	"github.com/arklib/ark/errx"
)

type (
	HttpBaseHandler = hz.HandlerFunc
	HttpMiddlewares = []HttpMiddleware
	HttpMiddleware  = hz.HandlerFunc

	HttpRoutes = []*HttpRoute
	HttpRoute  struct {
		Title           string
		Describe        string
		Method          string
		Path            string
		Handler         *ApiProxy
		Middlewares     Middlewares
		HttpMiddlewares HttpMiddlewares
		FullPath        string
	}
)

type HttpRouter struct {
	parent          *HttpRouter
	nodes           []*HttpRouter
	path            string
	routes          HttpRoutes
	middlewares     Middlewares
	httpMiddlewares HttpMiddlewares
	Title           string
	Describe        string
}

func newHttpRouter(path string, r *HttpRouter) *HttpRouter {
	return &HttpRouter{path: path, parent: r}
}

func (r *HttpRouter) AddMiddleware(middlewares ...Middleware) *HttpRouter {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *HttpRouter) AddHttpMiddleware(middlewares ...HttpMiddleware) *HttpRouter {
	r.httpMiddlewares = append(r.httpMiddlewares, middlewares...)
	return r
}

func (r *HttpRouter) Group(path string, middlewares ...HttpMiddleware) *HttpRouter {
	path = strings.Trim(path, "/")
	if path == "" {
		log.Println("group path cannot be empty")
		return nil
	}

	if r.path != "" {
		path = fmt.Sprintf("%s/%s", r.path, path)
	}

	router := newHttpRouter(path, r)
	router.httpMiddlewares = middlewares

	r.nodes = append(r.nodes, router)
	return router
}

func (r *HttpRouter) AddRoute(route *HttpRoute, middlewares ...HttpMiddleware) *HttpRouter {
	route.HttpMiddlewares = append(route.HttpMiddlewares, middlewares...)
	r.routes = append(r.routes, route)
	return r
}

func (r *HttpRouter) AddRoutes(routes HttpRoutes, middlewares ...HttpMiddleware) *HttpRouter {
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.AddHttpMiddleware(middlewares...)
	return r
}
func (r *HttpRouter) upMiddlewares() Middlewares {
	router := r
	var middlewares Middlewares
	for {
		if router == nil {
			break
		}
		if len(router.middlewares) > 0 {
			middlewares = append(router.middlewares, middlewares...)
		}
		router = router.parent
	}
	return middlewares
}

func (r *HttpRouter) upHttpMiddlewares() HttpMiddlewares {
	router := r
	var middlewares HttpMiddlewares
	for {
		if router == nil {
			break
		}
		if len(router.httpMiddlewares) > 0 {
			middlewares = append(router.httpMiddlewares, middlewares...)
		}
		router = router.parent
	}
	return middlewares
}

func (r *HttpRouter) setupRouter(httpSrv *httpServer, routes *HttpRoutes) (err error) {
	upMiddlewares := r.upMiddlewares()
	upHttpMiddlewares := r.upHttpMiddlewares()

	for _, route := range r.routes {
		if route.Handler == nil {
			return errx.New("route BaseHandler cannot be empty")
		}

		route.Path = strings.Trim(route.Path, "/")

		if route.Method == "" {
			route.Method = "POST"
		}

		// route full path
		if r.path == "" {
			route.FullPath = route.Path
		} else {
			route.FullPath = fmt.Sprintf("%s/%s", r.path, route.Path)
		}

		// add route to routes
		*routes = append(*routes, route)

		// set api proxy info
		apiProxy := route.Handler
		apiProxy.Srv = httpSrv.srv
		apiProxy.Path = route.FullPath

		// up httpMiddlewares + route.httpMiddlewares
		route.HttpMiddlewares = append(upHttpMiddlewares, route.HttpMiddlewares...)

		// up api httpMiddlewares + route.middlewares
		route.Middlewares = append(upMiddlewares, route.Middlewares...)

		// add api.proxy httpMiddlewares
		apiProxy.Use(route.Middlewares...)

		// httpMiddlewares + apiProxy.HttpHandler
		handlers := append(route.HttpMiddlewares, apiProxy.HttpHandler)

		// register route
		httpSrv.hzSrv.Handle(route.Method, route.FullPath, handlers...)
	}

	for _, node := range r.nodes {
		err = node.setupRouter(httpSrv, routes)
		if err != nil {
			return err
		}
	}
	return nil
}
