package ark

import (
	"context"

	hz "github.com/cloudwego/hertz/pkg/app"
)

type Ctx struct {
	context.Context
	req any
	srv *Server
}

func newCtx(ctx context.Context, req any, srv *Server) *Ctx {
	return &Ctx{ctx, req, srv}
}

func (c *Ctx) HttpReq() *hz.RequestContext {
	return c.req.(*hz.RequestContext)
}
