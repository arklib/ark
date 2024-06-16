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
	*httpRouter

	srv      *Server
	hzSrv    *hzsrv.Hertz
	allRoute HttpRoutes
}

func newHttpServer(srv *Server) *httpServer {
	s := &httpServer{
		httpRouter: newHttpRouter("", nil),
		srv:        srv,
	}
	s.init(srv)
	return s
}

func (s *httpServer) init(srv *Server) {
	logger := srv.Logger
	logger.Info("[ark] init http server")
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
		logger.Info("[http.server] recovery enabled")
	}

	// cors
	if config.UseCORS {
		hzSrv.Use(cors.Default())
		logger.Info("[http.server] cors enabled")
	}

	// gzip
	if config.UseGzip != 0 {
		hzSrv.Use(gzip.Gzip(config.UseGzip))
		logger.Info("[http.server] gzip enabled")
	}

	// etag
	if config.UseETag {
		// hzSrv.Use(etag.New())
		logger.Info("[http.server] etag enabled")
	}

	// access log
	if config.UseAccessLog {
		hzSrv.Use(accesslog.New())
		logger.Info("[http.server] accessLog enabled")
	}

	// static file routes
	for _, route := range config.UseFileRoutes {
		hzSrv.Static(route.Path, route.Root)
		logger.Infof("[http.server] fileRoute '%s' -> '%s'", route.Path, route.Root)
	}

	s.hzSrv = hzSrv
}

func (s *httpServer) HzServer() *hzsrv.Hertz {
	return s.hzSrv
}

func (s *httpServer) AllRoute() HttpRoutes {
	return s.allRoute
}

func (s *httpServer) run() error {
	// setup router
	err := s.setupRouter(s, &s.allRoute)
	if err != nil {
		return err
	}

	return s.hzSrv.Run()
}
