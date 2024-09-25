package ark

import (
	"fmt"
	"log"
	"strings"

	kesvc "github.com/cloudwego/kitex/pkg/serviceinfo"

	"github.com/arklib/ark/errx"
)

type (
	RPCRoutes = []*RPCRoute
	RPCRoute  struct {
		Title       string
		Describe    string
		Path        string
		Handler     *ApiProxy
		Middlewares Middlewares
		FullPath    string
	}
	rpcRouter struct {
		parent      *rpcRouter
		nodes       []*rpcRouter
		path        string
		routes      RPCRoutes
		middlewares Middlewares
		Title       string
		Describe    string
	}
)

func newRPCRouter(path string, r *rpcRouter) *rpcRouter {
	return &rpcRouter{path: path, parent: r}
}

func (r *rpcRouter) AddMiddleware(middlewares ...Middleware) *rpcRouter {
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

func (r *rpcRouter) Group(path string) *rpcRouter {
	path = strings.Trim(path, "/")
	if path == "" {
		log.Println("group path cannot be empty")
		return nil
	}

	if r.path != "" {
		path = fmt.Sprintf("%s/%s", r.path, path)
	}

	router := newRPCRouter(path, r)
	r.nodes = append(r.nodes, router)
	return router
}

func (r *rpcRouter) AddRoute(route *RPCRoute) *rpcRouter {
	r.routes = append(r.routes, route)
	return r
}

func (r *rpcRouter) AddRoutes(routes RPCRoutes) *rpcRouter {
	for _, route := range routes {
		r.AddRoute(route)
	}
	return r
}

func (r *rpcRouter) upMiddlewares() Middlewares {
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

func (r *rpcRouter) setupRouter(rpcSrv *rpcServer, routes *RPCRoutes) error {
	config := rpcSrv.srv.config.RPCServer
	httpSrv := rpcSrv.srv.HttpServer
	upMiddlewares := r.upMiddlewares()

	for _, route := range r.routes {
		if route.Handler == nil {
			return errx.New("route BaseHandler cannot be empty")
		}

		route.Path = strings.Trim(route.Path, "/")

		// add rpc method
		if r.path == "" {
			route.FullPath = route.Path
		} else {
			route.FullPath = fmt.Sprintf("%s/%s", r.path, route.Path)
		}

		// add route to all routes
		*routes = append(*routes, route)

		// set api proxy info
		apiProxy := route.Handler
		apiProxy.Srv = rpcSrv.srv
		apiProxy.Path = route.FullPath

		// up api httpMiddlewares + route.middlewares
		route.Middlewares = append(upMiddlewares, route.Middlewares...)
		// add api proxy httpMiddlewares
		apiProxy.Use(route.Middlewares...)

		// add kitex service method
		rpcSrv.keSvc.Methods[route.FullPath] = kesvc.NewMethodInfo(
			apiProxy.RPCHandler,
			apiProxy.NewInput,
			apiProxy.NewOutput,
			false,
		)

		// add http route
		if config.UseHttp && httpSrv != nil {
			httpSrv.AddRoute(&HttpRoute{
				Title:       route.Title,
				Describe:    route.Describe,
				Method:      "POST",
				Path:        fmt.Sprintf("%s/%s", config.Name, route.FullPath),
				Handler:     route.Handler,
				Middlewares: route.Middlewares,
			})
		}
	}

	for _, node := range r.nodes {
		err := node.setupRouter(rpcSrv, routes)
		if err != nil {
			return err
		}
	}
	return nil
}
