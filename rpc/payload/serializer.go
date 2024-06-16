package payload

import (
	"strings"

	"github.com/arklib/ark/errx"
	"github.com/arklib/ark/serializer"
)

func NewSerializer(name string) (codec Codec, err error) {
	switch strings.ToLower(name) {
	case "frugal":
		serialize := serializer.NewFrugal()
		codec = NewCodec("Frugal", serialize)
	case "gojson":
		serialize := serializer.NewGoJson()
		codec = NewCodec("GoJson", serialize)
	case "sonic":
		serialize := serializer.NewSonic()
		codec = NewCodec("Sonic", serialize)
	default:
		err = errx.Sprintf("unknown payload codec: %s", name)
	}
	return
}
