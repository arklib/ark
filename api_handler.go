package ark

import (
	"context"

	hz "github.com/cloudwego/hertz/pkg/app"
	_ "github.com/cloudwego/kitex/pkg/remote/codec/thrift"

	"github.com/arklib/ark/auth"
	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/http/result"
	"github.com/arklib/ark/util"
)

type (
	ApiMiddlewares []ApiMiddleware
	ApiMiddleware  func(*ApiPayload) error

	ApiPayload struct {
		Ctx  *Ctx
		Path string
		In   any
		Out  any
		Next func() error
	}

	ApiProxy struct {
		Srv  *Server
		Path string

		Name        string
		NewInput    func() any
		NewOutput   func() any
		BaseHandler func(*Ctx, any) (any, error)
		setOutput   func(dst any, src any)
		middlewares ApiMiddlewares
	}
)

func ApiHandler[In, Out any](handler func(*Ctx, *In) (*Out, error)) *ApiProxy {
	return &ApiProxy{
		Name:      util.GetFnName(handler),
		NewInput:  func() any { return new(In) },
		NewOutput: func() any { return new(Out) },
		BaseHandler: func(c *Ctx, in any) (any, error) {
			return handler(c, in.(*In))
		},
		setOutput: func(val any, newVal any) {
			*val.(*Out) = *newVal.(*Out)
		},
	}
}

func (proxy *ApiProxy) Use(middlewares ...ApiMiddleware) {
	proxy.middlewares = append(proxy.middlewares, middlewares...)
}

func (proxy *ApiProxy) Handle(c *Ctx, in, out any) (p *ApiPayload, err error) {
	p = &ApiPayload{
		Ctx:  c,
		Path: proxy.Path,
		In:   in,
		Out:  out,
	}

	index := 0
	p.Next = func() error {
		if index == len(proxy.middlewares) {
			p.Out, err = proxy.BaseHandler(c, in)
			return err
		}
		fn := proxy.middlewares[index]
		index++
		return fn(p)
	}
	return p, p.Next()
}

func (proxy *ApiProxy) HttpHandler(ctx context.Context, reqCtx *hz.RequestContext) {
	srv := proxy.Srv
	in := proxy.NewInput()

	// bind input
	err := reqCtx.Bind(in)
	if err != nil {
		err = errx.New("input error", err)
		result.Error(reqCtx, err)
		return
	}

	// bind auth
	data, ok := reqCtx.Get(auth.StoreAuthKey)
	if ok {
		err = util.BindStructFromMap(in, auth.StoreAuthKey, data.(auth.Payload))
		if err != nil {
			result.Error(reqCtx, auth.ErrAuthFailed)
			return
		}
	}

	// validate input
	lang := string(reqCtx.GetHeader("Accept-Language"))
	if err = srv.Validator.Test(in, lang); err != nil {
		result.Error(reqCtx, err)
		return
	}

	// proxy handle
	c := newCtx(ctx, srv, reqCtx)
	payload, err := proxy.Handle(c, in, proxy.NewOutput())
	if err != nil {
		result.Error(reqCtx, err)
		return
	}
	result.Success(reqCtx, payload.Out)
}

func (proxy *ApiProxy) RPCHandler(ctx context.Context, _, in, out any) error {
	srv := proxy.Srv
	c := newCtx(ctx, srv, nil)

	payload, err := proxy.Handle(c, in, out)
	if err != nil {
		return err
	}

	proxy.setOutput(out, payload.Out)
	return nil
}
