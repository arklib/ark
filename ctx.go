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

func newCtx(ctx context.Context, srv *Server, req any) *Ctx {
	return &Ctx{ctx, req, srv}
}

func (c *Ctx) Http() *hz.RequestContext {
	return c.req.(*hz.RequestContext)
}
