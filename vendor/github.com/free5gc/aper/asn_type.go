package aper

import (
	"reflect"
)

// BIT STRING

// BitString is for an ASN.1 BIT STRING type, BitLength means the effective bits.
type BitString struct {
	Bytes     []byte // bits packed into bytes.
	BitLength uint64 // length in bits.
}

// OCTET STRING

// OctetString is for an ASN.1 OCTET STRING type
type OctetString []byte

// OBJECT IDENTIFIER

// ObjectIdentifier is for an ASN.1 OBJECT IDENTIFIER type
type ObjectIdentifier []byte

// ENUMERATED

// An Enumerated is represented as a plain uint64.
type Enumerated uint64

var (
	// BitStringType is the type of BitString
	BitStringType = reflect.TypeOf(BitString{})
	// OctetStringType is the type of OctetString
	OctetStringType = reflect.TypeOf(OctetString{})
	// ObjectIdentifierType is the type of ObjectIdentify
	ObjectIdentifierType = reflect.TypeOf(ObjectIdentifier{})
	// EnumeratedType is the type of Enumerated
	EnumeratedType = reflect.TypeOf(Enumerated(0))
)
