package marshal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/spec"
)

const (
	SUPPORTED_MAJOR_VERSION = 4
	SUPPORTED_MINOR_VERSION = 8

	NIL_SIGN                = '0'
	TRUE_SIGN               = 'T'
	FALSE_SIGN              = 'F'
	FIXNUM_SIGN             = 'i'
	RAWSTRING_SIGN          = '"'
	SYMBOL_SIGN             = ':'
	SYMBOL_LINK_SIGN        = ';'
	OBJECT_SIGN             = 'o'
	OBJECT_LINK_SIGN        = '@'
	ARRAY_SIGN              = '['
	IVAR_SIGN               = 'I'
	HASH_SIGN               = '{'
	CLASS_SIGN              = 'c'
	USER_CLASS_SIGN         = 'C'
	USER_DEFINED_SIGN       = 'u'
	USER_MARSHAL_SIGN       = 'U'
	EXTENDED_BY_MODULE_SIGN = 'e'
	MODULE_SIGN             = 'm'
	EMPTY_STRING            = 26
)

func encInt(buff *bytes.Buffer, i int) error {
	var len int

	if i == 0 {
		return buff.WriteByte(0)
	} else if 0 < i && i < 123 {
		return buff.WriteByte(byte(i + 5))
	} else if -124 < i && i <= -1 {
		return buff.WriteByte(byte(i - 5))
	} else if 122 < i && i <= 0xff {
		len = 1
	} else if 0xff < i && i <= 0xffff {
		len = 2
	} else if 0xffff < i && i <= 0xffffff {
		len = 3
	} else if 0xffffff < i && i <= 0x3fffffff {
		//for compatibility with 32bit Ruby, Fixnum should be less than 1073741824
		len = 4
	} else if -0x100 <= i && i < -123 {
		len = -1
	} else if -0x10000 <= i && i < -0x100 {
		len = -2
	} else if -0x1000000 <= i && i < -0x100000 {
		len = -3
	} else if -0x40000000 <= i && i < -0x1000000 {
		//for compatibility with 32bit Ruby, Fixnum should be greater than -1073741825
		len = -4
	}

	if err := buff.WriteByte(byte(len)); err != nil {
		return err
	}
	if len < 0 {
		len = -len
	}

	for c := 0; c < len; c++ {
		if err := buff.WriteByte(byte(i >> uint(8*c) & 0xff)); err != nil {
			return err
		}
	}

	return nil
}

func encHash(buff *bytes.Buffer, size int, olinktbl map[string]int, linkidx *int) {
	buff.WriteByte(HASH_SIGN)
	encInt(buff, size)
	if olinktbl[string([]byte{HASH_SIGN})] == 0 {
		*linkidx += 1
		olinktbl[string([]byte{HASH_SIGN})] = *linkidx
	}
}

func encArray(buff *bytes.Buffer, size int, olinktbl map[string]int, olinkidx *int) {
	buff.WriteByte(ARRAY_SIGN)
	arrlen := size
	encInt(buff, arrlen)
	if olinktbl[string([]byte{ARRAY_SIGN})] == 0 {
		*olinkidx += 1
		olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
	}
}

func encArrayNoCache(buff *bytes.Buffer, size int) {
	buff.WriteByte(ARRAY_SIGN)
	arrlen := size
	encInt(buff, arrlen)
}

func encArrayAndIncrementIndex(buff *bytes.Buffer, size int, olinktbl map[string]int, olinkidx *int) {
	buff.WriteByte(ARRAY_SIGN)
	arrlen := size
	encInt(buff, arrlen)
	*olinkidx += 1
	olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
}

func encSymbol(buff *bytes.Buffer, symbol []byte, slinktbl map[string]int, slinkidx *int) {
	if slinktbl[string(symbol)] != 0 {
		buff.WriteByte(SYMBOL_LINK_SIGN)
		encInt(buff, slinktbl[string(symbol)]-1)
	} else {
		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, (len(symbol)))
		buff.Write(symbol)
		*slinkidx += 1
		slinktbl[string(symbol)] = *slinkidx
	}
}

func encStringNoCache(buff *bytes.Buffer, str string, olinkidx *int, slinktbl map[string]int, slinkidx *int) {
	buff.WriteByte(IVAR_SIGN)
	buff.WriteByte(RAWSTRING_SIGN)
	strlen := len(str)
	encInt(buff, strlen)
	buff.WriteString(str)
	buff.WriteByte(6)
	*olinkidx += 1
	encSymbol(buff, []byte{'E'}, slinktbl, slinkidx)
	buff.WriteByte(TRUE_SIGN)
}

func encString(buff *bytes.Buffer, str string, olinktbl map[string]int, olinkidx *int, slinktbl map[string]int, slinkidx *int) {
	if olinktbl[str] != 0 {
		buff.WriteByte(OBJECT_LINK_SIGN)
		encInt(buff, olinktbl[str])
	} else {
		buff.WriteByte(IVAR_SIGN)
		buff.WriteByte(RAWSTRING_SIGN)
		strlen := len(str)
		encInt(buff, strlen)
		buff.WriteString(str)
		buff.WriteByte(6)
		*olinkidx += 1
		olinktbl[str] = *olinkidx
		encSymbol(buff, []byte{'E'}, slinktbl, slinkidx)
		buff.WriteByte(TRUE_SIGN)
	}
}

func encGemVersion(buff *bytes.Buffer, version string, olinktbl map[string]int, olinkidx *int, slinktbl map[string]int, slinkidx *int) {
	class := "Gem::Version"
	key := class + version
	if olinktbl[key] != 0 {
		buff.WriteByte(OBJECT_LINK_SIGN)
		encInt(buff, olinktbl[key])
	} else {
		buff.WriteByte(USER_MARSHAL_SIGN)
		encSymbol(buff, []byte("Gem::Version"), slinktbl, slinkidx)
		encArrayNoCache(buff, 1)
		// encString(buff, version, olinktbl, olinkidx, slinktbl, slinkidx)
		encStringNoCache(buff, version, olinkidx, slinktbl, slinkidx)
		*olinkidx += 1
		olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
		*olinkidx += 1
		olinktbl[string([]byte{USER_MARSHAL_SIGN})] = *olinkidx
	}
}

func DumpBundlerDeps(gems []*db.Gem) ([]byte, error) {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	olinktbl := make(map[string]int)
	buff := bytes.NewBuffer(nil)
	buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	encArray(buff, len(gems), olinktbl, &olinkidx)
	for _, gem := range gems {
		encHash(buff, 4, olinktbl, &olinkidx)
		encSymbol(buff, []byte("name"), slinktbl, &slinkidx)
		encStringNoCache(buff, gem.Name, &olinkidx, slinktbl, &slinkidx)
		encSymbol(buff, []byte("number"), slinktbl, &slinkidx)
		encStringNoCache(buff, gem.Number, &olinkidx, slinktbl, &slinkidx)
		encSymbol(buff, []byte("platform"), slinktbl, &slinkidx)
		encStringNoCache(buff, gem.Platform, &olinkidx, slinktbl, &slinkidx)
		encSymbol(buff, []byte("dependencies"), slinktbl, &slinkidx)
		encArrayAndIncrementIndex(buff, len(gem.Dependencies), olinktbl, &olinkidx)
		for _, dep := range gem.Dependencies {
			depArr := []string{dep.Name, dep.VersionConstraints}
			encArrayAndIncrementIndex(buff, len(depArr), olinktbl, &olinkidx)
			for _, d := range depArr {
				encStringNoCache(buff, d, &olinkidx, slinktbl, &slinkidx)
			}
		}
	}
	return buff.Bytes(), nil
}

// TODO: Encode strings and cache them. This reduces the spec index sizes by roughly 1/2
func DumpSpecs(specs []*spec.Spec) []byte {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	olinktbl := make(map[string]int)
	buff := bytes.NewBuffer(nil)
	buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	encArrayNoCache(buff, len(specs))
	for _, spec := range specs {
		encArrayAndIncrementIndex(buff, 3, olinktbl, &olinkidx) // Inner Array Len (Always 3 for modern indicies)
		// encString(buff, spec.Name, olinktbl, &olinkidx, slinktbl, &slinkidx)
		encStringNoCache(buff, spec.Name, &olinkidx, slinktbl, &slinkidx)
		encGemVersion(buff, spec.Version, olinktbl, &olinkidx, slinktbl, &slinkidx)
		// encString(buff, spec.OriginalPlatform, olinktbl, &olinkidx, slinktbl, &slinkidx)
		encStringNoCache(buff, spec.OriginalPlatform, &olinkidx, slinktbl, &slinkidx)
	}

	return buff.Bytes()
}

func DumpGemspecGemfast(meta *spec.GemMetadata) []byte {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	buff := bytes.NewBuffer(nil)
	buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	buff.WriteByte(OBJECT_SIGN)
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 18)
	buff.WriteString("Gem::Specification")
	num, err := meta.NumInstanceVars()
	if err != nil {
		panic(err)
	}
	encInt(buff, num) // Number of instance variables

	// Name
	buff.WriteByte(SYMBOL_SIGN)
	buff.WriteByte(10)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("name")
	encStringNoCache(buff, meta.Name, &olinkidx, slinktbl, &slinkidx)

	// Version
	buff.WriteByte(SYMBOL_SIGN)
	buff.WriteByte(13)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("version")
	buff.WriteByte(USER_MARSHAL_SIGN)
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 12)
	buff.WriteString("Gem::Version")
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	encStringNoCache(buff, meta.Version.Version, &olinkidx, slinktbl, &slinkidx)

	// Summary
	buff.WriteByte(SYMBOL_SIGN)
	buff.WriteByte(13)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("summary")
	encStringNoCache(buff, meta.Summary, &olinkidx, slinktbl, &slinkidx)

	// Required Ruby Version
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 22) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("required_ruby_version")
	buff.WriteByte(USER_MARSHAL_SIGN)
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 16)
	buff.WriteString("Gem::Requirement")
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 2)
	encStringNoCache(buff, ">=", &olinkidx, slinktbl, &slinkidx)

	buff.WriteByte(USER_MARSHAL_SIGN)
	buff.WriteByte(SYMBOL_LINK_SIGN)
	buff.WriteByte(9)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	encStringNoCache(buff, "2.6.0", &olinkidx, slinktbl, &slinkidx)

	// Required Rubygems Version
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 26) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("required_rubygems_version")
	buff.WriteByte(USER_MARSHAL_SIGN)
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 16)
	buff.WriteString("Gem::Requirement")
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 2)
	encStringNoCache(buff, ">=", &olinkidx, slinktbl, &slinkidx)
	buff.WriteByte(USER_MARSHAL_SIGN)
	buff.WriteByte(SYMBOL_LINK_SIGN)
	buff.WriteByte(9)
	buff.WriteByte(ARRAY_SIGN)
	encInt(buff, 1)
	encStringNoCache(buff, "3.3.3", &olinkidx, slinktbl, &slinkidx)

	// Original Platform
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 18) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("original_platform")
	encStringNoCache(buff, meta.Platform, &olinkidx, slinktbl, &slinkidx)

	// Email
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 6)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("email")
	buff.WriteByte(ARRAY_SIGN)
	arrlen := len(meta.Emails)
	encInt(buff, arrlen) // Length of array
	for _, email := range meta.Emails {
		encStringNoCache(buff, email, &olinkidx, slinktbl, &slinkidx)
	}

	// Authors
	buff.WriteByte(SYMBOL_SIGN)
	buff.WriteByte(13)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("authors")
	buff.WriteByte(ARRAY_SIGN)
	arrlen = len(meta.Authors)
	encInt(buff, arrlen)
	for _, author := range meta.Authors {
		encStringNoCache(buff, author, &olinkidx, slinktbl, &slinkidx)
	}

	// Description
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 12) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("description")
	encStringNoCache(buff, meta.Description, &olinkidx, slinktbl, &slinkidx)

	// Homepage
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 9) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("homepage")
	encStringNoCache(buff, meta.Homepage, &olinkidx, slinktbl, &slinkidx)

	// Licenses
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 9)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("licenses")
	buff.WriteByte(ARRAY_SIGN)
	arrlen = len(meta.Licenses)
	encInt(buff, arrlen) // Length of array
	for _, lic := range meta.Licenses {
		encStringNoCache(buff, lic, &olinkidx, slinktbl, &slinkidx)
	}

	// Require Paths
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 14)
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("require_paths")
	buff.WriteByte(ARRAY_SIGN)
	arrlen = len(meta.RequirePaths)
	encInt(buff, arrlen) // Length of array
	for _, rp := range meta.RequirePaths {
		encStringNoCache(buff, rp, &olinkidx, slinktbl, &slinkidx)
	}

	// Specification Version
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 22) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("specification_version")
	buff.WriteByte(FIXNUM_SIGN)
	encInt(buff, meta.SpecVersion) //specification version value

	// Dependencies
	buff.WriteByte(SYMBOL_SIGN)
	encInt(buff, 13) //Length of symbol + 1 for the '@' character
	buff.WriteByte(OBJECT_LINK_SIGN)
	buff.WriteString("dependencies")
	buff.WriteByte(ARRAY_SIGN)
	arrlen = len(meta.Dependencies)
	encInt(buff, arrlen) // Length of arr
	for _, dep := range meta.Dependencies {
		buff.WriteByte(OBJECT_SIGN)
		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 15)
		buff.WriteString("Gem::Dependency")
		buff.WriteByte(10)
		buff.WriteByte(SYMBOL_LINK_SIGN)
		buff.WriteByte(6)
		encStringNoCache(buff, dep.Name, &olinkidx, slinktbl, &slinkidx)

		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 12) //Length of symbol + 1 for the '@' character
		buff.WriteByte(OBJECT_LINK_SIGN)
		buff.WriteString("requirement")
		buff.WriteByte(USER_MARSHAL_SIGN)
		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 16)
		buff.WriteString("Gem::Requirement")
		buff.WriteByte(ARRAY_SIGN)
		encInt(buff, 1)
		buff.WriteByte(ARRAY_SIGN)
		encInt(buff, len(dep.Requirement.VersionConstraints))
		for _, vc := range dep.Requirement.VersionConstraints {
			buff.WriteByte(ARRAY_SIGN)
			encInt(buff, 2)
			encStringNoCache(buff, vc.Constraint, &olinkidx, slinktbl, &slinkidx)

			buff.WriteByte(USER_MARSHAL_SIGN)
			buff.WriteByte(SYMBOL_LINK_SIGN)
			encInt(buff, 4)
			buff.WriteByte(ARRAY_SIGN)
			encInt(buff, 1)
			encStringNoCache(buff, vc.Version, &olinkidx, slinktbl, &slinkidx)

		}
		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 5) //Length of symbol + 1 for the '@' character
		buff.WriteByte(OBJECT_LINK_SIGN)
		buff.WriteString("type")
		buff.WriteByte(SYMBOL_SIGN)
		strlen := len(dep.Type) - 1
		encInt(buff, strlen)
		buff.WriteString(dep.Type[1:])
		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 11) //Length of symbol + 1 for the '@' character
		buff.WriteByte(OBJECT_LINK_SIGN)
		buff.WriteString("prerelease")
		if dep.Prerelease {
			buff.WriteByte(TRUE_SIGN)
		} else {
			buff.WriteByte(FALSE_SIGN)
		}

		buff.WriteByte(SYMBOL_SIGN)
		encInt(buff, 21) //Length of symbol + 1 for the '@' character
		buff.WriteByte(OBJECT_LINK_SIGN)
		buff.WriteString("version_requirements")
		buff.WriteByte(OBJECT_LINK_SIGN)
		buff.WriteByte(EMPTY_STRING)
	}

	// Rubygems version
	encSymbol(buff, []byte("rubygems_version"), slinktbl, &slinkidx)
	encStringNoCache(buff, meta.RubygemsVersion, &olinkidx, slinktbl, &slinkidx)

	return buff.Bytes()
}

// TODO: Fix reads so "_" gem doesnt end up as "\fGem::Version"
func LoadSpecs(src io.Reader) []*spec.Spec {
	var specs []*spec.Spec
	var slinktbl [][]byte
	var olinktbl [][]byte
	reader := bufio.NewReader(src)
	_, err := reader.ReadByte() // Major version
	_, err = reader.ReadByte()  // Minor version
	if err != nil {
		panic(err)
	}
	_, err = reader.ReadByte() // Array sign

	osize, err := readInt(reader) // Outer Array Len
	i := 0
	for i < int(osize) {
		b, err := reader.ReadByte() // Array sign
		if b != ARRAY_SIGN {
			panic(err)
		}
		olinktbl = append(olinktbl, []byte{'['})
		isize, err := readInt(reader) // Inner array len (3)
		if err != nil || isize != 3 {
			panic(err)
		}
		name, err := readName(reader, &slinktbl, &olinktbl)
		if err != nil {
			panic(err)
		}
		version, err := readVersion(reader, &slinktbl, &olinktbl)
		if err != nil {
			panic(err)
		}
		platform, err := readPlatform(reader, &slinktbl, &olinktbl)
		if err != nil {
			panic(err)
		}

		spec := spec.Spec{
			Name:             name,
			Version:          version,
			OriginalPlatform: platform,
		}
		specs = append(specs, &spec)
		i++
	}
	olinktbl = append(olinktbl, []byte{'['})
	return specs
}

func readName(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	b, err := r.ReadByte() // IVAR
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // RAWSTRING
	if b != RAWSTRING_SIGN {
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
		return string(b), errors.New("not u")
	}
	b, err = r.ReadByte() // Symbol sign
	if b != SYMBOL_SIGN && b != SYMBOL_LINK_SIGN {
		return string(b), errors.New("not symbol or link")
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
		*slinktbl = append(*slinktbl, tmp)
	}

	b, err = r.ReadByte() // Array sign
	if b != ARRAY_SIGN {
		return string(b), errors.New("not array")
	}
	b, err = r.ReadByte() // Array len (6 aka 1)
	b, err = r.ReadByte() // IVAR
	if b != IVAR_SIGN {
		return string(b), errors.New("not ivar")
	}
	b, err = r.ReadByte() // RAWSTRING
	if b != RAWSTRING_SIGN {
		return string(b), errors.New("not string")
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
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // 0
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
		return string(b), errors.New("")
	}
	*olinktbl = append(*olinktbl, []byte{'['})
	*olinktbl = append(*olinktbl, []byte{'U'})
	return string(versionBytes), err
}

func readPlatform(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	b, err := r.ReadByte() // IVAR
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // RAWSTR
	if b != RAWSTRING_SIGN {
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
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // 0
	// b, err = r.ReadByte() // E
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
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
