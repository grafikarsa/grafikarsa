package body

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"slices"

	"github.com/google/uuid"
)

// CanonicalHash returns the SHA-256 of the canonical JSON encoding of v.
// Object keys are sorted recursively and no insignificant whitespace is
// emitted, so semantically identical values with different key order hash
// equally. It is used for content_hash on version snapshots.
func CanonicalHash(v any) ([]byte, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(encoded))
	dec.UseNumber()
	var decoded any
	if err := dec.Decode(&decoded); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("canonical hash: trailing JSON data")
	}

	canonical, err := marshalCanonical(decoded)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	return sum[:], nil
}

// marshalCanonical encodes v as compact JSON without HTML escaping. Go sorts
// map keys deterministically, which makes the output canonical for objects.
func marshalCanonical(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func slicesSortUUIDs(ids []uuid.UUID) {
	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})
}
