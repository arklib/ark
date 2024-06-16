package serializer

type Serializer interface {
	Encode(val any) ([]byte, error)
	Decode(data []byte, val any) error
}
