package ark

import (
	"context"

	hz "github.com/cloudwego/hertz/pkg/app"
	_ "github.com/cloudwego/kitex/pkg/remote/codec/thrift"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/http/result"
	"github.com/arklib/ark/util"
)

type (
	Middlewares []Middleware
	Middleware  func(*ApiPayload) error

	ApiPayload struct {
		At   *At
		Path string
		In   any
		Out  any
		Next func() error
	}

	ApiProxy struct {
		Srv         *Server
		Path        string
		Name        string
		NewInput    func() any
		NewOutput   func() any
		BaseHandler func(*At, any) (any, error)
		setOutput   func(dst any, src any)
		middlewares Middlewares
	}
)

func ApiHandler[In, Out any](handler func(*At, *In) (*Out, error)) *ApiProxy {
	return &ApiProxy{
		Name:      util.GetFnName(handler),
		NewInput:  func() any { return new(In) },
		NewOutput: func() any { return new(Out) },
		BaseHandler: func(at *At, in any) (any, error) {
			return handler(at, in.(*In))
		},
		setOutput: func(val any, newVal any) {
			*val.(*Out) = *newVal.(*Out)
		},
	}
}

func (proxy *ApiProxy) Use(middlewares ...Middleware) {
	proxy.middlewares = append(proxy.middlewares, middlewares...)
}

func (proxy *ApiProxy) Handle(at *At, in, out any) (p *ApiPayload, err error) {
	p = &ApiPayload{
		At:   at,
		Path: proxy.Path,
		In:   in,
		Out:  out,
	}

	index := 0
	p.Next = func() error {
		if index == len(proxy.middlewares) {
			p.Out, err = proxy.BaseHandler(at, in)
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

	// validate input
	lang := string(reqCtx.GetHeader("Accept-Language"))
	err = srv.Validator.Test(in, lang)
	if err != nil {
		result.Error(reqCtx, err)
	}

	// get user
	// var user *auth.User
	// authUser, ok := req.Get(auth.StoreUserKey)
	// if ok {
	// 	user = authUser.(*auth.User)
	// }

	// proxy handle
	at := newAt(ctx, reqCtx, srv)
	payload, err := proxy.Handle(at, in, proxy.NewOutput())
	if err != nil {
		result.Error(reqCtx, err)
		return
	}
	result.Success(reqCtx, payload.Out)
}

func (proxy *ApiProxy) RPCHandler(ctx context.Context, _, in, out any) error {
	srv := proxy.Srv
	at := newAt(ctx, srv, nil)

	payload, err := proxy.Handle(at, in, out)
	if err != nil {
		return err
	}

	proxy.setOutput(out, payload.Out)
	return nil
}
