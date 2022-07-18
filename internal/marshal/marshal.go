package marshal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gscho/gemfast/internal/spec"

	log "github.com/sirupsen/logrus"
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

func DumpSpecs(specs []*spec.Spec) ([]byte) {
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
	log.SetOutput(os.Stdout)
	var specs []*spec.Spec
	var slinktbl [][]byte
	var olinktbl [][]byte
	reader := bufio.NewReader(src)
	_, err := reader.ReadByte() // Major version
	_, err = reader.ReadByte()  // Minor version
	if err != nil {
		panic(err)
	}
	_, err = reader.ReadByte()      // Array sign
	
	osize, err := readInt(reader) // Outer Array Len
	i := 0
	for i < int(osize) {
		b, err := reader.ReadByte() // Array sign
		if b != ARRAY_SIGN {
			log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("LoadSpecs: excepted ARRAY_SIGN")
			log.WithFields(log.Fields{"index": i}).Error("Failed index")
			panic(err)
		}
		isize, err := readInt(reader) // Inner array len (3)
		if err != nil || isize != 3 {
			log.WithFields(log.Fields{"len": isize}).Error("readInt failed to parse")
			panic(err)
		}
		name, err := readName(reader, &slinktbl, &olinktbl)
		if err != nil {
			log.WithFields(log.Fields{"index": i, "name": name}).Error("readName failed to parse a string")
			panic(err)
		}
		version, err := readVersion(reader, &slinktbl, &olinktbl)
		if err != nil {
			log.WithFields(log.Fields{"index": i, "version": version}).Error("readVersion failed to parse a string")
			panic(err)
		}
		platform, err := readPlatform(reader, &slinktbl, &olinktbl)
		if err != nil {
			log.WithFields(log.Fields{"index": i, "platform": platform}).Error("readPlatform failed to parse a string")
			panic(err)
		}				
		olinktbl = append(olinktbl, []byte{'['})

		spec := spec.Spec{
			Name:             name,
			Version:          version,
			OriginalPlatform: platform,
		}
		// log.WithFields(log.Fields{"name": spec.Name, "version": spec.Version, "platform": spec.OriginalPlatform}).Info("Loaded spec")
		specs = append(specs, &spec)
		i++
	}
	olinktbl = append(olinktbl, []byte{'['})
	return specs
}

func readName(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	b, err := r.ReadByte()       // IVAR
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		log.WithFields(log.Fields{"actual": string(b)}).Error("readName: excepted IVAR_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte()       // RAWSTRING
	if b != RAWSTRING_SIGN {
		log.WithFields(log.Fields{"actual": string(b)}).Error("readName: excepted RAWSTRING_SIGN")
		return string(b), errors.New("")
	}
	strlen, err := readInt(r) // String length
	if err != nil {
		return fmt.Sprint(strlen), err
	}
	j := 0
	var nameBytes []byte
	for j < int(strlen) {
		b, err = r.ReadByte()
		nameBytes = append(nameBytes, b)
		j++
	}
	*olinktbl = append(*olinktbl, nameBytes)
	b, err = r.ReadByte() // 6
	b, err = r.ReadByte() // Symbol sign
	if b != SYMBOL_SIGN && b != SYMBOL_LINK_SIGN {
		log.WithFields(log.Fields{"actual": string(b)}).Error("readName: excepted SYMBOL_SIGN or SYMBOL_LINK_SIGN")
		return string(b), errors.New("")
	}
	if b == SYMBOL_LINK_SIGN {
		b, err = r.ReadByte() // 0
	} else {
		len, _ := r.ReadByte() // 6
		sym, _ := r.ReadByte() // E

		*slinktbl = append(*slinktbl, []byte{len, sym})
	}
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
		log.WithFields(log.Fields{"actual": string(b)}).Error("readName: excepted 'T'")
		return string(b), errors.New("")
	}
	return string(nameBytes), nil
}

func readVersion(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	var versionBytes []byte
	var i int
	b, err := r.ReadByte() // U
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != 'U' {
		b, err = r.ReadByte()
		log.WithFields(log.Fields{"actual": string(b)}).Error("readVersion: expected U")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // Symbol sign
	if b != SYMBOL_SIGN && b != SYMBOL_LINK_SIGN {
		log.WithFields(log.Fields{"actual": string(b)}).Error("readVersion: excepted SYMBOL_SIGN or SYMBOL_LINK_SIGN")
		return string(b), errors.New("")
	}
	if b == SYMBOL_LINK_SIGN {
		b, err = r.ReadByte() // 0
	} else {
		strlen, _ := readInt(r) // Length of string
		tmp := []byte{byte(strlen)}
		for i < int(strlen) {
			b, err = r.ReadByte()
			tmp = append(tmp, b)
			i++
		}
		*olinktbl = append(*olinktbl, tmp)
	}

	b, err = r.ReadByte()      // Array sign
	if b != ARRAY_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readVersion: excepted ARRAY_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte()      // Array len (6 aka 1)
	b, err = r.ReadByte()      // IVAR
	if b != IVAR_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readVersion: excepted IVAR_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte()      // RAWSTRING
	if b != RAWSTRING_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readVersion: excepted RAWSTRING_SIGN")
		return string(b), errors.New("")
	}
	strlen, _ := readInt(r) // Length of version string
	i = 0
	for i < int(strlen) {
		b, err = r.ReadByte()
		versionBytes = append(versionBytes, b)
		i++
	}
	*olinktbl = append(*olinktbl, versionBytes)
	b, err = r.ReadByte() // 1
	b, err = r.ReadByte() // Symbol Link sign
	if b != SYMBOL_LINK_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readVersion: excepted SYMBOL_LINK_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // 0
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readVersion: excepted TRUE_SIGN")
		return string(b), errors.New("")
	}
	*olinktbl = append(*olinktbl, []byte{'['})
	*olinktbl = append(*olinktbl, []byte{'U'})
	return string(versionBytes), err
}

func readPlatform(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	b, err := r.ReadByte()      // IVAR
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readPlatform: excepted IVAR_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte()      // RAWSTR
	if b != RAWSTRING_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readPlatform: excepted RAWSTRING_SIGN")
		return string(b), errors.New("")
	}
	strlen, _ := readInt(r) // length of platform string
	var platformBytes []byte
	j := 0
	for j < int(strlen) {
		b, err = r.ReadByte()
		platformBytes = append(platformBytes, b)
		j++
	}
	*olinktbl = append(*olinktbl, platformBytes)
	b, err = r.ReadByte() // 6
	b, err = r.ReadByte() // Symbol link sign
	if b != SYMBOL_LINK_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readPlatform: excepted SYMBOL_LINK_SIGN")
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // 0
	// b, err = r.ReadByte() // E
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
		log.WithFields(log.Fields{"actual": b, "ASCII": string(b)}).Error("readPlatform: excepted TRUE_SIGN")
		return string(b), errors.New("")
	}
	return string(platformBytes), err
}

func readObjectLink(r *bufio.Reader, olinktbl *[][]byte) (string, error) {
	idx, err := readInt(r)
	idx = idx - 1 // First index is 1
	tmp := (*olinktbl)[idx]
	return string(tmp), err 
}

func readInt(r *bufio.Reader) (int, error) {
	var result int
	b, _ := r.ReadByte()
	c := int(int8(b))
	if c == 0 {
		return 0, nil
	} else if 5 < c && c < 128 {
		return c - 5, nil
	} else if -129 < c && c < -5 {
		return c + 5, nil
	}
	cInt8 := int8(b)
	if cInt8 > 0 {
		result = 0
		for i := int8(0); i < cInt8; i++ {
			n, _ := r.ReadByte()
			result |= int(uint(n) << (8 * uint(i)))
		}
	} else {
		result = -1
		c = -c
		for i := 0; i < c; i++ {
			n, _ := r.ReadByte()
			result &= ^(0xff << uint(8*i))
			result |= int(n) << uint(8*i)
		}
	}
	return result, nil
}
