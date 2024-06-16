package ark

import (
	"log"
	"os"
	"time"

	"github.com/gookit/goutil/dump"

	"github.com/arklib/ark/config"
	"github.com/arklib/ark/logger"
	"github.com/arklib/ark/registry"
	"github.com/arklib/ark/validator"
)

type ServerConfig struct {
	Mode       string
	Lang       string `default:"en"`
	Logger     *logger.Config
	Registry   *registry.Config
	HttpServer struct {
		Enable        bool
		Name          string `default:"api"`
		Addr          string `default:":8888"`
		UsePprof      bool
		UseGzip       int
		UseCORS       bool
		UseRecovery   bool
		UseAccessLog  bool
		UseETag       bool
		UseFileRoutes []struct {
			Path string
			Root string
		}
	}
	RPCServer struct {
		Enable bool
		// registry service name
		Name        string `default:"svc"`
		Addr        string `default:":8889"`
		UseHttp     bool
		UseTracing  bool
		UseRecovery bool
		UseValidate bool
		UseCodec    string `default:"frugal"`
		// generate code
		UseCodeGen struct {
			Enable bool
			Output string
		}
	}
	RPCClient struct {
		Enable     bool
		Timeout    time.Duration `default:"30s"`
		Services   []string
		UseTracing bool
		UseCodec   string `default:"frugal"`
	}
}

type Server struct {
	isRun      bool
	dumper     *dump.Dumper
	config     *ServerConfig
	Mode       string
	Config     *config.Config
	Logger     *logger.Logger
	Validator  *validator.Validator
	HttpServer *httpServer
	RPCClient  *rpcClient
	RPCServer  *rpcServer
}

func MustNewServer(c *config.Config) *Server {
	server, err := NewServer(c)
	if err != nil {
		log.Fatal(err)
	}
	return server
}

func NewServer(c *config.Config) (*Server, error) {
	srv := &Server{Config: c}
	return srv, srv.init()
}

func (srv *Server) init() (err error) {
	// bind server config
	sc := new(ServerConfig)
	err = srv.Config.BindStruct("", sc)
	if err != nil {
		return
	}

	srv.Mode = sc.Mode
	srv.config = sc

	// dumper
	srv.dumper = dump.NewDumper(os.Stdout, 3)

	// logger
	if srv.IsDev() {
		srv.Dump(srv.Config.Data())
		srv.Logger = logger.NewConsole(sc.Logger)
	} else {
		srv.Logger = logger.New(sc.Logger)
	}

	// validator
	srv.Validator = validator.New(sc.Lang)

	// http server
	if sc.HttpServer.Enable {
		srv.HttpServer = newHttpServer(srv)
	}

	// rpc server
	if sc.RPCServer.Enable {
		srv.RPCServer, err = newRPCServer(srv)
		if err != nil {
			return
		}
	}

	// rpc client
	if sc.RPCClient.Enable {
		srv.RPCClient, err = newRPCClient(srv)
		if err != nil {
			return
		}
	}
	return
}

// IsDev server mode == "dev"
func (srv *Server) IsDev() bool {
	return srv.Mode == "dev"
}

// Dump dump any value
func (srv *Server) Dump(v ...any) {
	srv.dumper.Print(v...)
}

// BindConfig bind config
func (srv *Server) BindConfig(key string, value any) error {
	return srv.Config.BindStruct(key, value)
}

// Run http server & rpc server
func (srv *Server) Run() {
	if srv.isRun {
		return
	}
	srv.isRun = true

	errCh := make(chan error)
	// start http server
	if srv.HttpServer != nil {
		go func() {
			errCh <- srv.HttpServer.run()
		}()
	}

	// start rpc server
	if srv.RPCServer != nil {
		go func() {
			errCh <- srv.RPCServer.run()
		}()
	}

	err := <-errCh
	if err != nil {
		srv.Logger.Error(err)
	}
}
