package ark

import (
	"context"

	hz "github.com/cloudwego/hertz/pkg/app"
)

type At struct {
	context.Context
	req any
	srv *Server
}

func newAt(ctx context.Context, req any, srv *Server) *At {
	return &At{ctx, req, srv}
}

func (at *At) HttpCtx() *hz.RequestContext {
	return at.req.(*hz.RequestContext)
}
