package codec

import (
	"sync"

	"github.com/klauspost/compress/zstd"
)

var (
	encoderOnce sync.Once
	decoderOnce sync.Once
	encoder     *zstd.Encoder
	decoder     *zstd.Decoder
	encoderErr  error
	decoderErr  error
)

func getEncoder() (*zstd.Encoder, error) {
	encoderOnce.Do(func() {
		encoder, encoderErr = zstd.NewWriter(nil)
	})
	return encoder, encoderErr
}

func getDecoder() (*zstd.Decoder, error) {
	decoderOnce.Do(func() {
		decoder, decoderErr = zstd.NewReader(nil)
	})
	return decoder, decoderErr
}

func Compress(data []byte) ([]byte, error) {
	enc, err := getEncoder()
	if err != nil {
		return nil, err
	}
	return enc.EncodeAll(data, make([]byte, 0, len(data))), nil
}

func Decompress(data []byte) ([]byte, error) {
	dec, err := getDecoder()
	if err != nil {
		return nil, err
	}
	return dec.DecodeAll(data, nil)
}
