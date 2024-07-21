package pshttp

import (
	"encoding/json"
	"io"
)

// EncodeFunc encodes a value of type T to the writer. The encoded bytes must be
// valid UTF-8. Every EncodeFunc should have a corresponding DecodeFunc.
type EncodeFunc[T any] func(T, io.Writer) error

// DecodeFunc decodes a value of type T from the reader. The decoded bytes are
// expected to be valid UTF-8. Every DecodeFunc should have a corresponding
// EncodeFunc.
type DecodeFunc[T any] func(io.Reader, *T) error

// Encode is a default EncodeFunc that encodes the value as JSON.
func Encode[T any](v T, w io.Writer) error {
	return json.NewEncoder(w).Encode(v)
}

// Decode is a default DecodeFunc that decodes the value as JSON.
func Decode[T any](r io.Reader, v *T) error {
	return json.NewDecoder(r).Decode(v)
}
