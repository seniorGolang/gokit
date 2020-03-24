package jsonrpc

import (
	"github.com/seniorGolang/gokit/types/uuid"
)

type autoUUIDGeneratorID struct{}

func NewUUIDGenerator() RequestIDGenerator {
	return &autoUUIDGeneratorID{}
}

func (i *autoUUIDGeneratorID) Generate() *RequestID {
	return &RequestID{
		stringValue: uuid.NewV4().String(),
	}
}
