package sweepprefetch

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Object is a JSON object that marshals its keys in insertion order.
//
// snapshot.json and refs-types.json are keyed by item number as a string, so
// Go's sorted-map marshalling would emit "10" before "2" and rewrite every
// file relative to what the reference implementation produced. Consumers look
// keys up rather than iterate, but keeping fetch order is what keeps the two
// implementations' output diffable.
type Object struct {
	keys []string
	vals map[string]any
}

func NewObject() *Object {
	return &Object{vals: map[string]any{}}
}

// Set appends a new key or updates an existing one in place.
func (o *Object) Set(key string, val any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = val
}

func (o *Object) Has(key string) bool {
	_, ok := o.vals[key]
	return ok
}

func (o *Object) Get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

func (o *Object) Keys() []string { return o.keys }

func (o *Object) Len() int { return len(o.keys) }

func (o *Object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := marshalPlain(k)
		if err != nil {
			return nil, err
		}
		val, err := marshalPlain(o.vals[k])
		if err != nil {
			return nil, fmt.Errorf("marshalling %q: %w", k, err)
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// DecodeObject reads a JSON object keeping the key order of the input, so a
// refs-types.json that grows over several passes is only ever appended to.
func DecodeObject(data []byte) (*Object, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("expected a JSON object, got %v", tok)
	}
	o := NewObject()
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected an object key, got %v", tok)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		o.Set(key, raw)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return o, nil
}

// Encode renders v the way these files have always been written: a one-space
// indent, no trailing newline, and HTML metacharacters left alone rather than
// escaped to their numeric form. PR titles are full of them and a reviewer
// reads these files by eye.
func Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func marshalPlain(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
