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
		Title          string
		Intro          string
		Path           string
		Handler        *ApiProxy
		ApiMiddlewares ApiMiddlewares
		FullPath       string
	}
)

type rpcRouter struct {
	parent         *rpcRouter
	nodes          []*rpcRouter
	path           string
	routes         RPCRoutes
	apiMiddlewares ApiMiddlewares
	Title          string
	Intro          string
}

func newRPCRouter(path string, r *rpcRouter) *rpcRouter {
	return &rpcRouter{path: path, parent: r}
}

func (r *rpcRouter) UseApiMiddleware(middlewares ...ApiMiddleware) *rpcRouter {
	r.apiMiddlewares = append(r.apiMiddlewares, middlewares...)
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

func (r *rpcRouter) upApiMiddlewares() ApiMiddlewares {
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

func (r *rpcRouter) setupRouter(rpcSrv *rpcServer, routes *RPCRoutes) error {
	config := rpcSrv.srv.config.RPCServer
	httpSrv := rpcSrv.srv.HttpServer
	upApiMiddlewares := r.upApiMiddlewares()

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

		// up api middlewares + route.apiMiddlewares
		route.ApiMiddlewares = append(upApiMiddlewares, route.ApiMiddlewares...)
		// add api proxy middlewares
		apiProxy.Use(route.ApiMiddlewares...)

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
				Title:   route.Title,
				Intro:   route.Intro,
				Method:  "POST",
				Path:    fmt.Sprintf("%s/%s", config.Name, route.FullPath),
				Handler: route.Handler,
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
