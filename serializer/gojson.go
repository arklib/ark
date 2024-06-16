package serializer

import "encoding/json"

type GoJson struct {
	Serializer
}

func (GoJson) Encode(val any) ([]byte, error) {
	return json.Marshal(val)
}

func (GoJson) Decode(data []byte, val any) error {
	return json.Unmarshal(data, val)
}

func NewGoJson() *GoJson {
	return new(GoJson)
}
