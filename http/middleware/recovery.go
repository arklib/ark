package middleware

import (
	"context"

	hz "github.com/cloudwego/hertz/pkg/app"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/http/result"
)

func Recovery() hz.HandlerFunc {
	return func(c context.Context, ctx *hz.RequestContext) {
		defer func() {
			if val := recover(); val != nil {
				switch {
				case errx.IsAppError(val):
					result.Error(ctx, val)
				default:
					result.Error(ctx, errx.NewX(val, errx.DefaultErrMessage))
				}
			}
		}()
		ctx.Next(c)
	}
}
