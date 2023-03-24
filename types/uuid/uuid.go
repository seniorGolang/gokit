package uuid

import (
	"encoding/base64"
	"errors"

	"github.com/globalsign/mgo/bson"
	stduuid "github.com/satori/go.uuid"
)

// Predefined namespace UUIDs.
var (
	NamespaceDNS  = UUID{stduuid.NamespaceDNS}
	NamespaceURL  = UUID{stduuid.NamespaceURL}
	NamespaceOID  = UUID{stduuid.NamespaceOID}
	NamespaceX500 = UUID{stduuid.NamespaceX500}
)

var (
	// By default, new (recommended) UUID subtype/kind (0x04) is used.
	bsonKind = bson.BinaryUUID
)

// Nil The nil UUID is special form of UUID that is specified to have all
// 128 bits set to zero.
var Nil = UUID{}

// UUID representation compliant with specification
// described in RFC 4122.
type UUID struct {
	stduuid.UUID `fake:"uuid"`
}

// NewV4 returns random generated UUID v4.
func NewV4() UUID {
	return UUID{stduuid.NewV4()}
}

// NewV5 returns generated UUID v5.
func NewV5(ns UUID, name string) UUID {
	return UUID{stduuid.NewV5(ns.UUID, name)}
}

// FromBase64 returns UUID from base64 string.
func FromBase64(input string) (UUID, error) {
	data, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return Nil, err
	}
	return FromBytes(data)
}

// FromUUID returns UUID from satori UUID.
func FromUUID(id stduuid.UUID) UUID {
	return UUID{id}
}

// FromString returns UUID parsed from string input.
// Input is expected in a form accepted by UnmarshalText.
func FromString(input string) (UUID, error) {
	u, err := stduuid.FromString(input)
	if err != nil {
		return Nil, err
	}
	return UUID{u}, nil
}

func FromStringOrNil(input string) UUID {

	u := stduuid.Nil
	if id, err := stduuid.FromString(input); err == nil {
		u = id
	}
	return UUID{u}
}

func FromBytesOrNil(input []byte) UUID {

	u := stduuid.Nil
	if id, err := stduuid.FromBytes(input); err == nil {
		u = id
	}
	return UUID{u}
}

// FromBytes returns UUID converted from raw byte slice input.
// It will return error if the slice isn't 16 bytes long.
func FromBytes(input []byte) (UUID, error) {
	u, err := stduuid.FromBytes(input)
	if err != nil {
		return Nil, err
	}
	return UUID{u}, nil
}

// FromStdUUID returns UUID converted from stduuid.UUUID.
func FromStdUUID(id stduuid.UUID) UUID {
	return UUID{id}
}

// Equal returns true if u1 and u2 equals, otherwise returns false.
func Equal(u1, u2 UUID) bool {
	return stduuid.Equal(u1.UUID, u2.UUID)
}

// Must is a helper that wraps a call to a function returning (UUID, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//	var packageUUID = uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000"));
func Must(u UUID, err error) UUID {
	return UUID{stduuid.Must(u.UUID, err)}
}

// IsNil get true is uuid nil.
func (u UUID) IsNil() bool {
	return stduuid.Equal(u.UUID, stduuid.Nil)
}

// GetBSON implement interface.
func (u UUID) GetBSON() (interface{}, error) {
	return bson.Binary{Kind: bsonKind, Data: u.Bytes()}, nil
}

// SetBSON implement interface.
func (u *UUID) SetBSON(raw bson.Raw) (err error) {
	var b bson.Binary
	if err = raw.Unmarshal(&b); err != nil {
		return err
	}
	if u.UUID, err = stduuid.FromBytes(b.Data); err != nil {
		return err
	}
	return nil
}

// Base64 return base64 encoded bytes
func (u UUID) Base64() string {
	return base64.RawURLEncoding.EncodeToString(u.Bytes())
}

// SetBSONKind changes BSON UUID Kind. Only values of 0x03 (Legacy UUID) or 0x04 (new UUID) can be used.
func SetBSONKind(kind byte) error {
	if kind < 0x03 || kind > 0x04 {
		return errors.New("requested BSON UUID kind is not allowed")
	}
	bsonKind = kind
	return nil
}
