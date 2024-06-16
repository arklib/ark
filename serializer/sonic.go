package serializer

import "github.com/bytedance/sonic"

type Sonic struct {
	Serializer
}

func (Sonic) Encode(val any) ([]byte, error) {
	return sonic.Marshal(val)
}

func (Sonic) Decode(data []byte, val any) error {
	return sonic.Unmarshal(data, val)
}

func NewSonic() *Sonic {
	return new(Sonic)
}
