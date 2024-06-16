package result

import (
	hz "github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/arklib/ark/errx"
)

type Result struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func Success(ctx *hz.RequestContext, data any) {
	ctx.AbortWithStatusJSON(consts.StatusOK, &Result{
		Code: consts.StatusOK,
		Data: data,
	})
}

func Error(ctx *hz.RequestContext, err any) {
	e := errx.Wrap(err)

	if e.IsBasic() || e.IsFatal() {
		hlog.DefaultLogger().Error(e.FullError())
	}

	ctx.AbortWithStatusJSON(e.Code(), &Result{
		Code:    e.Code(),
		Message: e.Error(),
	})
}
