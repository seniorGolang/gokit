package types

import (
	"reflect"
)

type KeyValue struct {
	key   string
	value interface{}
}

func KV(key string, value interface{}) KeyValue {
	return KeyValue{key: key, value: value}
}

func (kv KeyValue) Key() string {
	return kv.key
}

func (kv KeyValue) Value() interface{} {
	return kv.value
}

func (kv KeyValue) IsZero() bool {
	return reflect.ValueOf(kv.value).IsZero()
}
