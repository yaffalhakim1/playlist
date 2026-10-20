package innertube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// omap is a JSON object that remembers the order of its keys.
//
// InnerTube answers with deeply nested objects whose order carries meaning:
// the tracks of a playlist sit in an array, but the array itself may sit in
// any of several keys of one object. A plain map loses that order, so the
// parser walks an ordered tree instead.
type omap struct {
	keys []string
	vals map[string]any
}

func newOmap() *omap {
	return &omap{vals: map[string]any{}}
}

func (m *omap) set(key string, val any) {
	if _, seen := m.vals[key]; !seen {
		m.keys = append(m.keys, key)
	}
	m.vals[key] = val
}

// get returns the value of a key, or nil when the object has no such key.
func (m *omap) get(key string) any {
	return m.vals[key]
}

// object returns the value of a key when it is an object, and nil otherwise.
func (m *omap) object(key string) *omap {
	o, _ := m.vals[key].(*omap)
	return o
}

// array returns the value of a key when it is an array, and nil otherwise.
func (m *omap) array(key string) []any {
	a, _ := m.vals[key].([]any)
	return a
}

// text returns the value of a key when it is a string, and "" otherwise.
func (m *omap) text(key string) string {
	switch v := m.vals[key].(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

// number returns the value of a key when it is a number, and 0 otherwise.
func (m *omap) number(key string) int {
	switch v := m.vals[key].(type) {
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	default:
		return 0
	}
}

// decodeOrdered parses JSON into a tree of *omap, []any and scalars, keeping
// the order of every object's keys.
func decodeOrdered(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		obj := newOmap()
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, fmt.Errorf("innertube: object key is not a string")
			}
			val, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			obj.set(key, val)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return obj, nil
	case '[':
		items := []any{}
		for dec.More() {
			val, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			items = append(items, val)
		}
		if _, err := dec.Token(); err != nil {
			return nil, err
		}
		return items, nil
	default:
		return nil, fmt.Errorf("innertube: unexpected %v", delim)
	}
}

// walk calls fn for every object in the tree, parents before children, in
// document order. It stops descending into an object fn reports as handled.
func walk(node any, fn func(*omap) bool) {
	switch v := node.(type) {
	case *omap:
		if !fn(v) {
			return
		}
		for _, key := range v.keys {
			walk(v.vals[key], fn)
		}
	case []any:
		for _, item := range v {
			walk(item, fn)
		}
	}
}
