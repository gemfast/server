package marshal

import (
	"bytes"
	// "log"
	// "os"
	"github.com/gscho/gemfast/internal/spec"
)

const (
	SUPPORTED_MAJOR_VERSION = 4
	SUPPORTED_MINOR_VERSION = 8

	//   NIL_SIGN         = '0'
	TRUE_SIGN = 'T'
	//   FALSE_SIGN       = 'F'
	//   FIXNUM_SIGN      = 'i'
	RAWSTRING_SIGN = '"'
	SYMBOL_SIGN    = ':'
	// SYMBOL_LINK_SIGN = ';'
	//   OBJECT_SIGN      = 'o'
	//   OBJECT_LINK_SIGN = '@'
	ARRAY_SIGN = '['
	IVAR_SIGN  = 'I'
	//   HASH_SIGN        = '{'
	//   BIGNUM_SIGN      = 'l'
	//   REGEXP_SIGN      = '/'
	CLASS_SIGN = 'c'

//   MODULE_SIGN      = 'm'
)

func DumpSpecs(specs []*spec.Spec) ([]byte) {
	buff := bytes.NewBuffer(nil)
	buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	buff.WriteByte(ARRAY_SIGN)
	buff.WriteByte(byte(len(specs) + 5)) // Outer Array Len
	for _, spec := range(specs) {
		buff.WriteByte(ARRAY_SIGN)
		buff.WriteByte(8) // Inner Array Len (Always 3 for modern indicies)
		s := spec.Name
		l := len(s) + 5

		// String "chef-ruby-lvm-attrib"
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(l))
		buff.WriteString(s)
		buff.WriteByte(6)
		buff.WriteByte(SYMBOL_SIGN)
		buff.WriteByte(6)
		buff.WriteString("E")
		buff.WriteByte(TRUE_SIGN)

		// Gem::Version.new("0.3.10")
		cname := "Gem::Version"
		v := spec.Version
		l3 := len(cname) + 5
		buff.Write([]byte{'U'})
		buff.WriteByte(SYMBOL_SIGN)
		buff.WriteByte(byte(l3))
		buff.WriteString(cname)
		buff.WriteByte(ARRAY_SIGN)
		buff.WriteByte(6) // Array Len
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(len(v) + 5))
		buff.WriteString(v)
		buff.WriteByte(6)
		buff.WriteByte(SYMBOL_SIGN)
		buff.WriteByte(6)
		buff.WriteString("E")
		buff.WriteByte(TRUE_SIGN)

		// String "ruby"
		s2 := spec.OriginalPlatform
		l2 := len(s2) + 5
		buff.Write([]byte{IVAR_SIGN, RAWSTRING_SIGN})
		buff.WriteByte(byte(l2))
		buff.WriteString(s2)
		buff.WriteByte(6)
		buff.WriteByte(SYMBOL_SIGN)
		buff.WriteByte(6)
		buff.WriteString("E")
		buff.WriteByte(TRUE_SIGN)
	}

	return buff.Bytes()
	// // Open a new file for writing only
	// file, err := os.OpenFile(
	// 	"test.txt",
	// 	os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
	// 	0666,
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer file.Close()

	// // Write bytes to file
	// bytesWritten, err := file.Write(buff.Bytes())
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("Wrote %d bytes.\n", bytesWritten)
}
