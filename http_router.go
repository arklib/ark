package ark

import (
	"fmt"
	"log"
	"strings"

	hzapp "github.com/cloudwego/hertz/pkg/app"

	"github.com/arklib/ark/errx"
)

type (
	HttpBaseHandler = hzapp.HandlerFunc
	HttpMiddlewares = []HttpMiddleware
	HttpMiddleware  = hzapp.HandlerFunc

	HttpRoutes = []*HttpRoute
	HttpRoute  struct {
		Title          string
		Intro          string
		Method         string
		Path           string
		Handler        *ApiProxy
		Middlewares    HttpMiddlewares
		ApiMiddlewares ApiMiddlewares
		FullPath       string
	}
)

type httpRouter struct {
	parent         *httpRouter
	nodes          []*httpRouter
	path           string
	routes         HttpRoutes
	middlewares    HttpMiddlewares
	apiMiddlewares ApiMiddlewares
	Title          string
	Intro          string
}

func newHttpRouter(path string, r *httpRouter) *httpRouter {
	return &httpRouter{path: path, parent: r}
}

func (r *httpRouter) Use(middlewares ...HttpMiddleware) *httpRouter {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *httpRouter) UseApiMiddleware(middlewares ...ApiMiddleware) *httpRouter {
	r.apiMiddlewares = append(r.apiMiddlewares, middlewares...)
	return r
}

func (r *httpRouter) Group(path string, middlewares ...HttpMiddleware) *httpRouter {
	path = strings.Trim(path, "/")
	if path == "" {
		log.Println("group path cannot be empty")
		return nil
	}

	if r.path != "" {
		path = fmt.Sprintf("%s/%s", r.path, path)
	}

	router := newHttpRouter(path, r)
	router.middlewares = middlewares

	r.nodes = append(r.nodes, router)
	return router
}

func (r *httpRouter) AddRoute(route *HttpRoute, middlewares ...HttpMiddleware) *httpRouter {
	route.Middlewares = append(route.Middlewares, middlewares...)
	r.routes = append(r.routes, route)
	return r
}

func (r *httpRouter) AddRoutes(routes HttpRoutes, middlewares ...HttpMiddleware) *httpRouter {
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.Use(middlewares...)
	return r
}

func (r *httpRouter) upMiddlewares() HttpMiddlewares {
	router := r
	var middlewares HttpMiddlewares
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

func (r *httpRouter) upApiMiddlewares() ApiMiddlewares {
	router := r
	var middlewares ApiMiddlewares
	for {
		if router == nil {
			break
		}
		if len(router.apiMiddlewares) > 0 {
			middlewares = append(router.apiMiddlewares, middlewares...)
		}
		router = router.parent
	}
	return middlewares
}

func (r *httpRouter) setupRouter(httpSrv *httpServer, routes *HttpRoutes) (err error) {
	upMiddlewares := r.upMiddlewares()
	upApiMiddlewares := r.upApiMiddlewares()

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

		// up middlewares + route.middlewares
		route.Middlewares = append(upMiddlewares, route.Middlewares...)

		// up api middlewares + route.apiMiddlewares
		route.ApiMiddlewares = append(upApiMiddlewares, route.ApiMiddlewares...)

		// add api.proxy middlewares
		apiProxy.Use(route.ApiMiddlewares...)

		// middlewares + apiProxy.HttpHandler
		handlers := append(route.Middlewares, apiProxy.HttpHandler)

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
