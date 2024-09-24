package ark

import (
	"fmt"
	"go/format"
	"net"
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kesvc "github.com/cloudwego/kitex/pkg/serviceinfo"
	kesrv "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"

	"github.com/arklib/ark/codegen"
	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/logger"
	"github.com/arklib/ark/registry"
	"github.com/arklib/ark/rpc"
)

// Logger
type Logger struct {
	*logger.Logger
}

func (l *Logger) SetLevel(level klog.Level) {
	l.Logger.SetLevel(hlog.Level(level))
}

type rpcServer struct {
	*rpcRouter

	srv      *Server
	keSrv    kesrv.Server
	keSvc    *kesvc.ServiceInfo
	allRoute RPCRoutes
}

func newRPCServer(srv *Server) (*rpcServer, error) {
	s := &rpcServer{
		rpcRouter: newRPCRouter("", nil),
		srv:       srv,
	}
	return s, s.init(srv)
}

func (s *rpcServer) init(srv *Server) error {
	srv.Logger.Info("[ark] init rpc server")

	// set logger
	klog.SetLogger(&Logger{srv.Logger})

	config := srv.config.RPCServer
	if config.Name == "" {
		return errx.New("rpc server name cannot be empty")
	}

	// addr
	addr, err := net.ResolveTCPAddr("tcp", config.Addr)
	if err != nil {
		return err
	}

	basicInfo := &rpcinfo.EndpointBasicInfo{ServiceName: config.Name}

	// base options
	options := []kesrv.Option{
		kesrv.WithServiceAddr(addr),
		kesrv.WithServerBasicInfo(basicInfo),
		kesrv.WithMuxTransport(),
		kesrv.WithCompatibleMiddlewareForUnary(),
	}

	// codec
	codec, err := rpc.NewCodec()
	if err != nil {
		return err
	}
	options = append(options, kesrv.WithPayloadCodec(codec))

	// tracing
	if config.UseTracing {
		trace := tracing.NewServerSuite()
		options = append(options, kesrv.WithSuite(trace))
	}

	// registry config
	if len(srv.config.Registry.Addrs) > 0 {
		r, err := registry.NewRegistry(srv.config.Registry)
		if err != nil {
			return err
		}
		options = append(options, kesrv.WithRegistry(r))
	}

	// service info
	s.keSvc = &kesvc.ServiceInfo{
		ServiceName:  config.Name,
		Methods:      make(map[string]kesvc.MethodInfo),
		PayloadCodec: kesvc.Thrift,
	}

	// new server
	s.keSrv = kesrv.NewServer(options...)

	// register service
	err = s.keSrv.RegisterService(s.keSvc, new(struct{}))
	if err != nil {
		return err
	}

	// recovery
	if config.UseRecovery {
		s.UseApiMiddleware(s.UseRecovery())
	}

	// validate
	if config.UseValidate {
		s.UseApiMiddleware(s.UseValidate())
	}

	return nil
}

func (s *rpcServer) UseRecovery() ApiMiddleware {
	return func(p *ApiPayload) (err error) {
		defer func() {
			if val := recover(); val != nil {
				err = errx.New(val)
				return
			}
		}()
		return p.Next()
	}
}

func (s *rpcServer) UseValidate() ApiMiddleware {
	return func(p *ApiPayload) error {
		err := s.srv.Validator.Test(p.In, "")
		if err != nil {
			return err
		}
		return p.Next()
	}
}

func (s *rpcServer) KeServer() kesrv.Server {
	return s.keSrv
}

func (s *rpcServer) ClientSource(pkgName string) ([]byte, error) {
	config := s.srv.config.RPCServer

	// pkg code
	pkg := codegen.NewPackage(pkgName)
	pkg.AddImport("github.com/arklib/ark")

	code := ""
	write := func(format string, v ...any) {
		code += fmt.Sprintf(format+"\n", v...)
	}

	write("type Service struct {")
	write("    at *ark.At")
	write("}\n")

	write("func New(at *ark.At) *Service {")
	write("    return &Service{at}")
	write("}\n")

	for _, route := range s.allRoute {
		hInfo := route.Handler
		in := pkg.AddStruct(hInfo.NewInput())
		out := pkg.AddStruct(hInfo.NewOutput())

		// comment: title
		if route.Title != "" {
			write("// %s %s", hInfo.Name, route.Title)
		}

		// comment: intro
		if route.Intro != "" {
			write("// %s", route.Intro)
		}

		write("func (s *Service) %s(in *%s) (out *%s, err error) {",
			hInfo.Name,
			in.Name,
			out.Name,
		)
		write("    out = new(%s)", out.Name)
		write(`    err = s.at.FetchSvc("%s/%s", in, out)`, config.Name, route.FullPath)
		write("    return")
		write("}\n")
	}

	header := "// Code generated by ark. DO NOT EDIT."
	source := fmt.Sprintf("%s\n\n%s\n\n%s",
		header,
		pkg.Source(),
		code,
	)
	return format.Source([]byte(source))
}

func (s *rpcServer) genCode() error {
	config := s.srv.config.RPCServer.UseCodeGen
	if !config.Enable {
		return nil
	}

	// client code
	_, pkgName := codegen.ParsePkgPath(config.Output)
	code, err := s.ClientSource(pkgName)
	if err != nil {
		return err
	}

	// output code file
	codeFile := fmt.Sprintf("%s/%s.go", config.Output, pkgName)
	return os.WriteFile(codeFile, code, 0444)
}

func (s *rpcServer) AllRoute() RPCRoutes {
	return s.allRoute
}

func (s *rpcServer) run() error {
	// setup router
	err := s.rpcRouter.setupRouter(s, &s.allRoute)
	if err != nil {
		return err
	}

	// gen code
	err = s.genCode()
	if err != nil {
		return err
	}

	return s.keSrv.Run()
}
