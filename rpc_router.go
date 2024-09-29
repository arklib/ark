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
		Describe       string
		Path           string
		Handler        *ApiProxy
		ApiMiddlewares ApiMiddlewares
		Router         *RPCRouter
		FullPath       string
	}
	RPCRouter struct {
		parent         *RPCRouter
		nodes          []*RPCRouter
		routes         RPCRoutes
		apiMiddlewares ApiMiddlewares
		Path           string
		Title          string
		Describe       string
	}
)

func newRPCRouter(path string, r *RPCRouter) *RPCRouter {
	return &RPCRouter{Path: path, parent: r}
}

func (r *RPCRouter) WithApiMiddleware(middlewares ...ApiMiddleware) *RPCRouter {
	r.apiMiddlewares = append(r.apiMiddlewares, middlewares...)
	return r
}

func (r *RPCRouter) Group(path string) *RPCRouter {
	path = strings.Trim(path, "/")
	if path == "" {
		log.Println("group Path cannot be empty")
		return nil
	}

	if r.Path != "" {
		path = fmt.Sprintf("%s/%s", r.Path, path)
	}

	router := newRPCRouter(path, r)
	r.nodes = append(r.nodes, router)
	return router
}

func (r *RPCRouter) AddRoute(route *RPCRoute) *RPCRouter {
	route.Router = r
	r.routes = append(r.routes, route)
	return r
}

func (r *RPCRouter) AddRoutes(routes RPCRoutes) *RPCRouter {
	for _, route := range routes {
		r.AddRoute(route)
	}
	return r
}

func (r *RPCRouter) upApiMiddlewares() ApiMiddlewares {
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

func (r *RPCRouter) setupRouter(rpcSrv *rpcServer, routes *RPCRoutes) error {
	config := rpcSrv.srv.config.RPCServer
	httpSrv := rpcSrv.srv.HttpServer
	upApiMiddlewares := r.upApiMiddlewares()

	for _, route := range r.routes {
		if route.Handler == nil {
			return errx.New("route handler cannot be empty")
		}

		route.Path = strings.Trim(route.Path, "/")

		// add rpc method
		if r.Path == "" {
			route.FullPath = route.Path
		} else {
			route.FullPath = fmt.Sprintf("%s/%s", r.Path, route.Path)
		}

		// add route to all routes
		*routes = append(*routes, route)

		// set api proxy info
		apiProxy := route.Handler
		apiProxy.Srv = rpcSrv.srv
		apiProxy.Path = route.FullPath

		// up api httpMiddlewares + route.apiMiddlewares
		route.ApiMiddlewares = append(upApiMiddlewares, route.ApiMiddlewares...)
		// add api proxy httpMiddlewares
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
				Title:          route.Title,
				Describe:       route.Describe,
				Method:         "POST",
				Path:           fmt.Sprintf("%s/%s", config.Name, route.FullPath),
				Handler:        route.Handler,
				ApiMiddlewares: route.ApiMiddlewares,
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
