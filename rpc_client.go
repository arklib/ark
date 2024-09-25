package ark

import (
	"context"
	"strings"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	keService "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/transport"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/registry"
	"github.com/arklib/ark/rpc"
)

var EmptyRPCMethodInfo = keService.NewMethodInfo(
	func(ctx context.Context, handler, in, out any) error { return nil },
	func() any { return nil },
	func() any { return nil },
	false,
)

type rpcService struct {
	keSvc  *keService.ServiceInfo
	client client.Client
}

type rpcClient struct {
	srv      *Server
	services map[string]*rpcService
}

func newRPCClient(srv *Server) *rpcClient {
	cli := &rpcClient{
		srv:      srv,
		services: make(map[string]*rpcService),
	}
	return cli
}

func (c *rpcClient) init() error {
	srv := c.srv

	srv.Logger.Debug("[ark] init rpc client")
	config := srv.config.RPCClient

	// base options
	options := []client.Option{
		client.WithMuxConnection(1),
		client.WithRPCTimeout(config.Timeout),
		client.WithTransportProtocol(transport.Framed),
	}

	// codec
	codec, err := rpc.NewCodec()
	if err != nil {
		return err
	}
	options = append(options, client.WithPayloadCodec(codec))

	// tracing
	if config.UseTracing {
		trace := tracing.NewClientSuite()
		options = append(options, client.WithSuite(trace))
	}

	// registry
	if len(srv.config.Registry.Addrs) > 0 {
		r, err := registry.NewResolver(srv.config.Registry)
		if err != nil {
			return err
		}
		options = append(options, client.WithResolver(r))
	}

	// discover services name
	for _, name := range config.Discover {
		basicInfo := &rpcinfo.EndpointBasicInfo{ServiceName: name}
		cliOptions := append(
			options,
			client.WithDestService(name),
			client.WithClientBasicInfo(basicInfo),
		)

		keSvc := &keService.ServiceInfo{
			ServiceName:  name,
			Methods:      make(map[string]keService.MethodInfo),
			PayloadCodec: keService.Thrift,
		}

		cli, err := client.NewClient(keSvc, cliOptions...)
		if err != nil {
			return err
		}

		c.services[name] = &rpcService{
			keSvc:  keSvc,
			client: cli,
		}
	}
	return nil
}

// Call rpc service
func (c *rpcClient) Call(ctx context.Context, path string, in, out any) error {
	paths := strings.SplitN(path, "/", 2)
	if len(paths) != 2 {
		return errx.New("service path error")
	}
	name, method := paths[0], paths[1]

	svc, ok := c.services[name]
	if !ok {
		return errx.Sprintf("service not found: %s", name)
	}

	_, ok = svc.keSvc.Methods[method]
	if !ok {
		svc.keSvc.Methods[method] = EmptyRPCMethodInfo
	}
	return svc.client.Call(ctx, method, in, out)
}
