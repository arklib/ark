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
		Router          *HttpRouter
		Title           string
		Describe        string
		Method          string
		Path            string
		Handler         *ApiProxy
		ApiMiddlewares  ApiMiddlewares
		HttpMiddlewares HttpMiddlewares
		FullPath        string
	}
	HttpRouter struct {
		parent          *HttpRouter
		nodes           []*HttpRouter
		routes          HttpRoutes
		apiMiddlewares  ApiMiddlewares
		httpMiddlewares HttpMiddlewares
		Path            string
		Title           string
		Describe        string
	}
)

func newHttpRouter(path string, r *HttpRouter) *HttpRouter {
	return &HttpRouter{Path: path, parent: r}
}

func (r *HttpRouter) WithApiMiddleware(middlewares ...ApiMiddleware) *HttpRouter {
	r.apiMiddlewares = append(r.apiMiddlewares, middlewares...)
	return r
}

func (r *HttpRouter) WithHttpMiddleware(middlewares ...HttpMiddleware) *HttpRouter {
	r.httpMiddlewares = append(r.httpMiddlewares, middlewares...)
	return r
}

func (r *HttpRouter) Group(path string, middlewares ...HttpMiddleware) *HttpRouter {
	path = strings.Trim(path, "/")
	if path == "" {
		log.Println("group Path cannot be empty")
		return nil
	}

	if r.Path != "" {
		path = fmt.Sprintf("%s/%s", r.Path, path)
	}

	router := newHttpRouter(path, r)
	router.httpMiddlewares = middlewares

	r.nodes = append(r.nodes, router)
	return router
}

func (r *HttpRouter) AddRoute(route *HttpRoute) *HttpRouter {
	route.Router = r
	r.routes = append(r.routes, route)
	return r
}

func (r *HttpRouter) AddRoutes(routes HttpRoutes, middlewares ...HttpMiddleware) *HttpRouter {
	for _, route := range routes {
		r.AddRoute(route)
	}
	r.WithHttpMiddleware(middlewares...)
	return r
}
func (r *HttpRouter) upApiMiddlewares() ApiMiddlewares {
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
	upApiMiddlewares := r.upApiMiddlewares()
	upHttpMiddlewares := r.upHttpMiddlewares()

	for _, route := range r.routes {
		if route.Handler == nil {
			return errx.New("route BaseHandler cannot be empty")
		}

		route.Path = strings.Trim(route.Path, "/")

		if route.Method == "" {
			route.Method = "POST"
		}

		// route full Path
		if r.Path == "" {
			route.FullPath = route.Path
		} else {
			route.FullPath = fmt.Sprintf("%s/%s", r.Path, route.Path)
		}

		// add route to routes
		*routes = append(*routes, route)

		// set api proxy info
		apiProxy := route.Handler
		apiProxy.Srv = httpSrv.srv
		apiProxy.Path = route.FullPath

		// up httpMiddlewares + route.httpMiddlewares
		route.HttpMiddlewares = append(upHttpMiddlewares, route.HttpMiddlewares...)

		// up apiMiddlewares + route.apiMiddlewares
		route.ApiMiddlewares = append(upApiMiddlewares, route.ApiMiddlewares...)

		// add api.proxy httpMiddlewares
		apiProxy.Use(route.ApiMiddlewares...)

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
