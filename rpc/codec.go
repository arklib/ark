package rpc

import (
	"github.com/cloudwego/kitex/pkg/remote"
	"github.com/cloudwego/kitex/pkg/remote/codec/thrift"
)

func NewCodec() (remote.PayloadCodec, error) {
	codec := thrift.NewThriftCodecWithConfig(thrift.FrugalRead | thrift.FrugalWrite)
	return codec, nil
}
