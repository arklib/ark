package ark

import (
	"context"

	hzapp "github.com/cloudwego/hertz/pkg/app"

	"github.com/arklib/ark/auth"
)

type At struct {
	context.Context

	Srv     *Server
	User    *auth.User
	Console any
}

func newAt(ctx context.Context, srv *Server, user *auth.User, console any) *At {
	return &At{
		Context: ctx,
		Srv:     srv,
		User:    user,
		Console: console,
	}
}

func (at *At) Http() *hzapp.RequestContext {
	return at.Console.(*hzapp.RequestContext)
}

// FetchApi http service
func (at *At) FetchApi(path string, in, out any) error {
	// return at.Srv.ApiClient.Call(at.Context, path, in, out)
	return nil
}

// FetchSvc rpc service
func (at *At) FetchSvc(path string, in, out any) error {
	return at.Srv.RPCClient.Call(at.Context, path, in, out)
}
