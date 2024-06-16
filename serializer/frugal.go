package serializer

import "github.com/cloudwego/frugal"

type Frugal struct {
	Serializer
}

func (Frugal) Encode(val any) ([]byte, error) {
	data := make([]byte, frugal.EncodedSize(val))
	_, err := frugal.EncodeObject(data, nil, val)
	return data, err
}

func (Frugal) Decode(data []byte, val any) error {
	_, err := frugal.DecodeObject(data, val)
	return err
}

func NewFrugal() *Frugal {
	return new(Frugal)
}
