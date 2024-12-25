package main

type Encoder struct {
}

func (e *Encoder) Encode(data string) []byte {
	return []byte{}
}

func EncodeQr(data string) ([]byte, error) {
	enc := &Encoder{}
	return enc.Encode(data), nil
}
