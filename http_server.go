package ark

import (
	hzsrv "github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/cors"
	"github.com/hertz-contrib/gzip"
	"github.com/hertz-contrib/logger/accesslog"
	"github.com/hertz-contrib/pprof"

	"github.com/arklib/ark/http/middleware"
)

type httpServer struct {
	*HttpRouter

	srv    *Server
	hzSrv  *hzsrv.Hertz
	Routes HttpRoutes
}

func newHttpServer(srv *Server) *httpServer {
	s := &httpServer{
		HttpRouter: newHttpRouter("", nil),
		srv:        srv,
	}
	return s
}

func (s *httpServer) run() error {
	if s.hzSrv == nil {
		s.init()
	}

	// setup router
	err := s.setupRouter(s, &s.Routes)
	if err != nil {
		return err
	}

	return s.hzSrv.Run()
}

func (s *httpServer) init() {
	srv := s.srv
	srv.Logger.Debug("[ark] init http server")

	// set logger
	hlog.SetLogger(srv.Logger)

	config := srv.config.HttpServer
	// new hertz server
	hzSrv := hzsrv.New(hzsrv.WithHostPorts(config.Addr))

	// pprof
	if config.UsePprof {
		pprof.Register(hzSrv)
	}

	// recovery
	if config.UseRecovery {
		hzSrv.Use(middleware.Recovery())
		srv.Logger.Debug("[http.server] recovery enabled")
	}

	// cors
	if config.UseCORS {
		hzSrv.Use(cors.Default())
		srv.Logger.Debug("[http.server] cors enabled")
	}

	// gzip
	if config.UseGzip != 0 {
		hzSrv.Use(gzip.Gzip(config.UseGzip))
		srv.Logger.Debug("[http.server] gzip enabled")
	}

	// etag
	if config.UseETag {
		// hzSrv.Use(etag.New())
		srv.Logger.Debug("[http.server] etag enabled")
	}

	// access log
	if config.UseAccessLog {
		hzSrv.Use(accesslog.New())
		srv.Logger.Debug("[http.server] accessLog enabled")
	}

	// static file routes
	for _, route := range config.UseFileRoutes {
		hzSrv.Static(route.Path, route.Root)
		srv.Logger.Debugf("[http.server] fileRoute '%s' -> '%s'", route.Path, route.Root)
	}

	s.hzSrv = hzSrv
}

func (s *httpServer) HzServer() *hzsrv.Hertz {
	return s.hzSrv
}

func (s *httpServer) GetRoutes() HttpRoutes {
	return s.Routes
}
