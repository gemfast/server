package marshal

import (
	"bufio"
	"bytes"
	"io"

	"github.com/gscho/gemfast/internal/spec"
)

const (
	SUPPORTED_MAJOR_VERSION = 4
	SUPPORTED_MINOR_VERSION = 8

	//   NIL_SIGN         = '0'
	TRUE_SIGN = 'T'
	//   FALSE_SIGN       = 'F'
	//   FIXNUM_SIGN      = 'i'
	RAWSTRING_SIGN   = '"'
	SYMBOL_SIGN      = ':'
	SYMBOL_LINK_SIGN = ';'
	//   OBJECT_SIGN      = 'o'
	  OBJECT_LINK_SIGN = '@'
	ARRAY_SIGN = '['
	IVAR_SIGN  = 'I'
	//   HASH_SIGN        = '{'
	//   BIGNUM_SIGN      = 'l'
	//   REGEXP_SIGN      = '/'
	CLASS_SIGN = 'c'

//   MODULE_SIGN      = 'm'
)

func DumpSpecs(specs []*spec.Spec) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	buff.WriteByte(ARRAY_SIGN)
	buff.WriteByte(byte(len(specs) + 5)) // Outer Array Len
	for idx, spec := range specs {
		buff.WriteByte(ARRAY_SIGN)
		buff.WriteByte(8) // Inner Array Len (Always 3 for modern indicies)
		s := spec.Name
		l := len(s) + 5

		// String "chef-ruby-lvm-attrib"
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(l))
		buff.WriteString(s)
		buff.WriteByte(6)
		if idx == 0 {
			buff.WriteByte(SYMBOL_SIGN)
			buff.WriteByte(6)
			buff.WriteString("E")
		} else {
			buff.WriteByte(SYMBOL_LINK_SIGN)
			buff.WriteByte(0)
		}
		buff.WriteByte(TRUE_SIGN)

		// Gem::Version.new("0.3.10")
		cname := "Gem::Version"
		v := spec.Version
		l3 := len(cname) + 5
		buff.Write([]byte{'U'})
		if idx == 0 {
			buff.WriteByte(SYMBOL_SIGN)
			buff.WriteByte(byte(l3))
			buff.WriteString(cname)
		} else {
			buff.WriteByte(SYMBOL_LINK_SIGN)
			buff.WriteByte(6)
		}
		buff.WriteByte(ARRAY_SIGN)
		buff.WriteByte(6) // Array Len
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(len(v) + 5))
		buff.WriteString(v)
		buff.WriteByte(6)
		buff.WriteByte(SYMBOL_LINK_SIGN)
		buff.WriteByte(0)
		buff.WriteByte(TRUE_SIGN)

		// String "ruby"
		s2 := spec.OriginalPlatform
		l2 := len(s2) + 5
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(l2))
		buff.WriteString(s2)
		buff.WriteByte(6)
		buff.WriteByte(SYMBOL_LINK_SIGN)
		buff.WriteByte(0)
		buff.WriteByte(TRUE_SIGN)
	}

	return buff.Bytes()
}

func LoadSpecs(src io.Reader) []*spec.Spec {
	var specs []*spec.Spec
	reader := bufio.NewReader(src)
	b, err := reader.ReadByte() // Major version
	b, err = reader.ReadByte()  // Minor version
	if err != nil {
		panic(err)
	}
	b, err = reader.ReadByte()      // Array sign
	osize, err := reader.ReadByte() // Outer Array Len
	osize = osize - 5
	i := 0
	j := 0
	for i < int(osize) {
		b, err = reader.ReadByte() // Array sign
		b, err = reader.ReadByte() // Inner array len (3)

		b, err = reader.ReadByte()       // IVAR
		b, err = reader.ReadByte()       // RAWSTRING
		strlen, err := reader.ReadByte() // String length

		strlen = strlen - 5
		if err != nil {
			panic(err)
		}
		j = 0
		var nameBytes []byte
		for j < int(strlen) {
			b, err = reader.ReadByte()
			nameBytes = append(nameBytes, b)
			j++
		}
		b, err = reader.ReadByte() // 6
		b, err = reader.ReadByte() // Symbol sign
		if b == SYMBOL_LINK_SIGN {
			b, err = reader.ReadByte() // 0
		} else {
			b, err = reader.ReadByte() // 6
			b, err = reader.ReadByte() // E
		}
		b, err = reader.ReadByte() // TRUE sign

		// Version string seciton //
		b, err = reader.ReadByte() // U
		b, err = reader.ReadByte() // Symbol sign
		if b == SYMBOL_LINK_SIGN {
			b, err = reader.ReadByte() // 0
		} else {
			strlen, err = reader.ReadByte() // Length of string
			strlen = strlen - 5
			j = 0
			for j < int(strlen) {
				b, err = reader.ReadByte()
				j++
			}
		}

		b, err = reader.ReadByte()      // Array sign
		b, err = reader.ReadByte()      // Array len (6 aka 1)
		b, err = reader.ReadByte()      // IVAR
		b, err = reader.ReadByte()      // RAWSTRING
		strlen, err = reader.ReadByte() // Length of version string
		strlen = strlen - 5
		var versionBytes []byte
		j = 0
		for j < int(strlen) {
			b, err = reader.ReadByte()
			versionBytes = append(versionBytes, b)
			j++
		}
		b, err = reader.ReadByte() // 1
		b, err = reader.ReadByte() // Symbol Link sign
		b, err = reader.ReadByte() // 0
		// b, err = reader.ReadByte()      // E
		b, err = reader.ReadByte() // TRUE sign

		//Platform//
		b, err = reader.ReadByte()      // IVAR
		b, err = reader.ReadByte()      // RAWSTR
		strlen, err = reader.ReadByte() // length of platform string
		strlen = strlen - 5
		var platformBytes []byte
		j = 0
		for j < int(strlen) {
			b, err = reader.ReadByte()
			platformBytes = append(platformBytes, b)
			j++
		}
		b, err = reader.ReadByte() // 6
		b, err = reader.ReadByte() // Symbol link sign
		b, err = reader.ReadByte() // 0
		// b, err = reader.ReadByte() // E
		b, err = reader.ReadByte() // TRUE sign

		spec := spec.Spec{
			Name:             string(nameBytes),
			Version:          string(versionBytes),
			OriginalPlatform: string(platformBytes),
		}
		specs = append(specs, &spec)
		i++
	}
	return specs
}
