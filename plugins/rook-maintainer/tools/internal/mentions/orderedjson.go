package mentions

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// decodeObject walks a JSON object in document order, handing each key to fn,
// which must consume exactly one value from dec.
//
// Key order is preserved because the sweep dir's JSON is re-read and re-written
// on every run: sorting it would rewrite the whole file the first time this
// tool touched a sweep the Python script had produced.
func decodeObject(data []byte, fn func(key string, dec *json.Decoder) error) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("expected a JSON object")
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := kt.(string)
		if !ok {
			return fmt.Errorf("expected an object key, got %v", kt)
		}
		if err := fn(key, dec); err != nil {
			return err
		}
	}
	_, err = dec.Token()
	return err
}

// indentedObject renders an ordered mapping exactly as Python's
// `json.dump(obj, f, indent=1)` does — one-space indent, no trailing newline —
// so a sweep dir stays diff-clean across the two implementations.
func indentedObject(keys []string, value func(string) any) ([]byte, error) {
	if len(keys) == 0 {
		return []byte("{}"), nil
	}
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n ")
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteString(": ")
		vb, err := json.MarshalIndent(value(k), " ", " ")
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteString("\n}")
	return b.Bytes(), nil
}
