package driftflow

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type orderedMap struct {
	keys   []string
	values map[string]interface{}
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]interface{})}
}

func (om *orderedMap) set(key string, value interface{}) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

func (om orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range om.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		v, err := json.Marshal(om.values[k])
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "%q:%s", k, v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
