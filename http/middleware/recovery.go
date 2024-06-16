package middleware

import (
	"context"

	hzapp "github.com/cloudwego/hertz/pkg/app"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/http/result"
)

func Recovery() hzapp.HandlerFunc {
	return func(c context.Context, ctx *hzapp.RequestContext) {
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
