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

func encHash(buff *bytes.Buffer, size int, olinktbl map[string]int, linkidx *int) error {
	err := buff.WriteByte(HASH_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write hash sign: %w", err)
	}
	err = encInt(buff, size)
	if err != nil {
		return fmt.Errorf("failed to encode hash size: %w", err)
	}
	if olinktbl[string([]byte{HASH_SIGN})] == 0 {
		*linkidx += 1
		olinktbl[string([]byte{HASH_SIGN})] = *linkidx
	}
	return nil
}

func encArray(buff *bytes.Buffer, size int, olinktbl map[string]int, olinkidx *int) error {
	err := buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write array sign: %w", err)
	}
	arrlen := size
	err = encInt(buff, arrlen)
	if err != nil {
		return fmt.Errorf("failed to encode array size: %w", err)
	}
	if olinktbl[string([]byte{ARRAY_SIGN})] == 0 {
		*olinkidx += 1
		olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
	}
	return nil
}

func encArrayNoCache(buff *bytes.Buffer, size int) error {
	err := buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write array sign: %w", err)
	}
	arrlen := size
	err = encInt(buff, arrlen)
	if err != nil {
		return fmt.Errorf("failed to encode array size: %w", err)
	}
	return nil
}

func encArrayAndIncrementIndex(buff *bytes.Buffer, size int, olinktbl map[string]int, olinkidx *int) error {
	err := buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write array sign: %w", err)
	}
	arrlen := size
	err = encInt(buff, arrlen)
	if err != nil {
		return fmt.Errorf("failed to encode array size: %w", err)
	}
	*olinkidx += 1
	olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
	return nil
}

func encSymbol(buff *bytes.Buffer, symbol []byte, slinktbl map[string]int, slinkidx *int) error {
	if slinktbl[string(symbol)] != 0 {
		err := buff.WriteByte(SYMBOL_LINK_SIGN)
		if err != nil {
			return fmt.Errorf("failed to write symbol link sign: %w", err)
		}
		err = encInt(buff, slinktbl[string(symbol)]-1)
		if err != nil {
			return fmt.Errorf("failed to encode symbol link index: %w", err)
		}
	} else {
		err := buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return fmt.Errorf("failed to write symbol sign: %w", err)
		}
		err = encInt(buff, (len(symbol)))
		if err != nil {
			return fmt.Errorf("failed to encode symbol length: %w", err)
		}
		_, err = buff.Write(symbol)
		if err != nil {
			return fmt.Errorf("failed to write symbol: %w", err)
		}
		*slinkidx += 1
		slinktbl[string(symbol)] = *slinkidx
	}
	return nil
}

func encStringNoCache(buff *bytes.Buffer, str string, olinkidx *int, slinktbl map[string]int, slinkidx *int) error {
	err := buff.WriteByte(IVAR_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write IVAR sign: %w", err)
	}
	err = buff.WriteByte(RAWSTRING_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write raw string sign: %w", err)
	}
	strlen := len(str)
	err = encInt(buff, strlen)
	if err != nil {
		return fmt.Errorf("failed to encode string length: %w", err)
	}
	_, err = buff.WriteString(str)
	if err != nil {
		return fmt.Errorf("failed to write string: %w", err)
	}
	err = buff.WriteByte(6)
	if err != nil {
		return fmt.Errorf("failed to write end of string sign: %w", err)
	}
	*olinkidx += 1
	err = encSymbol(buff, []byte{'E'}, slinktbl, slinkidx)
	if err != nil {
		return fmt.Errorf("failed to encode symbol 'E': %w", err)
	}
	err = buff.WriteByte(TRUE_SIGN)
	if err != nil {
		return fmt.Errorf("failed to write true sign: %w", err)
	}
	return nil
}

// TODO: implement caching
// func encString(buff *bytes.Buffer, str string, olinktbl map[string]int, olinkidx *int, slinktbl map[string]int, slinkidx *int) {
// 	if olinktbl[str] != 0 {
// 		buff.WriteByte(OBJECT_LINK_SIGN)
// 		encInt(buff, olinktbl[str])
// 	} else {
// 		buff.WriteByte(IVAR_SIGN)
// 		buff.WriteByte(RAWSTRING_SIGN)
// 		strlen := len(str)
// 		encInt(buff, strlen)
// 		buff.WriteString(str)
// 		buff.WriteByte(6)
// 		*olinkidx += 1
// 		olinktbl[str] = *olinkidx
// 		encSymbol(buff, []byte{'E'}, slinktbl, slinkidx)
// 		buff.WriteByte(TRUE_SIGN)
// 	}
// }

func encGemVersion(buff *bytes.Buffer, version string, olinktbl map[string]int, olinkidx *int, slinktbl map[string]int, slinkidx *int) error {
	class := "Gem::Version"
	key := class + version
	if olinktbl[key] != 0 {
		err := buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return fmt.Errorf("failed to write object link sign: %w", err)
		}
		err = encInt(buff, olinktbl[key])
		if err != nil {
			return fmt.Errorf("failed to encode object link index: %w", err)
		}
	} else {
		err := buff.WriteByte(USER_MARSHAL_SIGN)
		if err != nil {
			return fmt.Errorf("failed to write user marshal sign: %w", err)
		}
		err = encSymbol(buff, []byte("Gem::Version"), slinktbl, slinkidx)
		if err != nil {
			return fmt.Errorf("failed to encode symbol 'Gem::Version': %w", err)
		}
		err = encArrayNoCache(buff, 1)
		if err != nil {
			return fmt.Errorf("failed to encode array for Gem::Version: %w", err)
		}
		// encString(buff, version, olinktbl, olinkidx, slinktbl, slinkidx)
		err = encStringNoCache(buff, version, olinkidx, slinktbl, slinkidx)
		if err != nil {
			return fmt.Errorf("failed to encode string for Gem::Version: %w", err)
		}
		*olinkidx += 1
		olinktbl[string([]byte{ARRAY_SIGN})] = *olinkidx
		*olinkidx += 1
		olinktbl[string([]byte{USER_MARSHAL_SIGN})] = *olinkidx
	}
	return nil
}

func DumpBundlerDeps(gems []*db.Gem) ([]byte, error) {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	olinktbl := make(map[string]int)
	buff := bytes.NewBuffer(nil)
	_, err := buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	if err != nil {
		return nil, fmt.Errorf("failed to write version bytes: %w", err)
	}
	err = encArray(buff, len(gems), olinktbl, &olinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array of gems: %w", err)
	}
	for _, gem := range gems {
		err = encHash(buff, 4, olinktbl, &olinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode hash for gem: %w", err)
		}
		err = encSymbol(buff, []byte("name"), slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol 'name': %w", err)
		}
		err = encStringNoCache(buff, gem.Name, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem name: %w", err)
		}
		err = encSymbol(buff, []byte("number"), slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol 'number': %w", err)
		}
		err = encStringNoCache(buff, gem.Number, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem number: %w", err)
		}
		err = encSymbol(buff, []byte("platform"), slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol 'platform': %w", err)
		}
		err = encStringNoCache(buff, gem.Platform, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem platform: %w", err)
		}
		err = encSymbol(buff, []byte("dependencies"), slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol 'dependencies': %w", err)
		}
		err = encArrayAndIncrementIndex(buff, len(gem.Dependencies), olinktbl, &olinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode dependencies array: %w", err)
		}
		for _, dep := range gem.Dependencies {
			depArr := []string{dep.Name, dep.VersionConstraints}
			err = encArrayAndIncrementIndex(buff, len(depArr), olinktbl, &olinkidx)
			if err != nil {
				return nil, fmt.Errorf("failed to encode dependency array: %w", err)
			}
			for _, d := range depArr {
				err = encStringNoCache(buff, d, &olinkidx, slinktbl, &slinkidx)
				if err != nil {
					return nil, fmt.Errorf("failed to encode dependency string: %w", err)
				}
			}
		}
	}
	return buff.Bytes(), nil
}

// TODO: Encode strings and cache them. This reduces the spec index sizes by roughly 1/2
func DumpSpecs(specs []*spec.Spec) ([]byte, error) {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	olinktbl := make(map[string]int)
	buff := bytes.NewBuffer(nil)
	_, err := buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	if err != nil {
		return nil, fmt.Errorf("failed to write version bytes: %w", err)
	}
	err = encArrayNoCache(buff, len(specs))
	if err != nil {
		return nil, fmt.Errorf("failed to encode array of specs: %w", err)
	}
	for _, spec := range specs {
		err = encArrayAndIncrementIndex(buff, 3, olinktbl, &olinkidx) // Inner Array Len (Always 3 for modern indicies)
		if err != nil {
			return nil, fmt.Errorf("failed to encode inner array for spec: %w", err)
		}
		// encString(buff, spec.Name, olinktbl, &olinkidx, slinktbl, &slinkidx)
		err = encStringNoCache(buff, spec.Name, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode spec name: %w", err)
		}
		err = encGemVersion(buff, spec.Version, olinktbl, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode spec version: %w", err)
		}
		// encString(buff, spec.OriginalPlatform, olinktbl, &olinkidx, slinktbl, &slinkidx)
		err = encStringNoCache(buff, spec.OriginalPlatform, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode spec original platform: %w", err)
		}
	}

	return buff.Bytes(), nil
}

func DumpGemspecGemfast(meta *spec.GemMetadata) ([]byte, error) {
	slinkidx := 0
	slinktbl := make(map[string]int)
	olinkidx := 0
	buff := bytes.NewBuffer(nil)
	_, err := buff.Write([]byte{SUPPORTED_MAJOR_VERSION, SUPPORTED_MINOR_VERSION})
	if err != nil {
		return nil, fmt.Errorf("failed to write version bytes: %w", err)
	}
	err = buff.WriteByte(OBJECT_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object sign: %w", err)
	}
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign: %w", err)
	}
	err = encInt(buff, 18)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length: %w", err)
	}
	_, err = buff.WriteString("Gem::Specification")
	if err != nil {
		return nil, fmt.Errorf("failed to write Gem::Specification: %w", err)
	}
	num, err := meta.NumInstanceVars()
	if err != nil {
		return nil, fmt.Errorf("failed to get number of instance variables: %w", err)
	}
	err = encInt(buff, num) // Number of instance variables
	if err != nil {
		return nil, fmt.Errorf("failed to encode number of instance variables: %w", err)
	}

	// Name
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for name: %w", err)
	}
	err = buff.WriteByte(10)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol for name: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for name: %w", err)
	}
	_, err = buff.WriteString("name")
	if err != nil {
		return nil, fmt.Errorf("failed to write name: %w", err)
	}
	err = encStringNoCache(buff, meta.Name, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem name: %w", err)
	}

	// Version
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for version: %w", err)
	}
	err = buff.WriteByte(13)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol for version: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for version: %w", err)
	}
	_, err = buff.WriteString("version")
	if err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	err = buff.WriteByte(USER_MARSHAL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write user marshal sign for version: %w", err)
	}
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for version class: %w", err)
	}
	err = encInt(buff, 12)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for version class: %w", err)
	}
	_, err = buff.WriteString("Gem::Version")
	if err != nil {
		return nil, fmt.Errorf("failed to write Gem::Version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for version: %w", err)
	}
	err = encStringNoCache(buff, meta.Version.Version, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem version: %w", err)
	}

	// Summary
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for summary: %w", err)
	}
	err = buff.WriteByte(13)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol for summary: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for summary: %w", err)
	}
	_, err = buff.WriteString("summary")
	if err != nil {
		return nil, fmt.Errorf("failed to write summary: %w", err)
	}
	err = encStringNoCache(buff, meta.Summary, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem summary: %w", err)
	}

	// Required Ruby Version
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for required ruby version: %w", err)
	}
	err = encInt(buff, 22) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for required ruby version: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for required ruby version: %w", err)
	}
	_, err = buff.WriteString("required_ruby_version")
	if err != nil {
		return nil, fmt.Errorf("failed to write required ruby version: %w", err)
	}
	err = buff.WriteByte(USER_MARSHAL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write user marshal sign for required ruby version: %w", err)
	}
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for Gem::Requirement: %w", err)
	}
	err = encInt(buff, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for Gem::Requirement: %w", err)
	}
	_, err = buff.WriteString("Gem::Requirement")
	if err != nil {
		return nil, fmt.Errorf("failed to write Gem::Requirement: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for required ruby version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for required ruby version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write inner array sign for required ruby version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner array length for required ruby version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write inner inner array sign for required ruby version: %w", err)
	}
	err = encInt(buff, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner inner array length for required ruby version: %w", err)
	}
	err = encStringNoCache(buff, ">=", &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode required ruby version operator: %w", err)
	}

	err = buff.WriteByte(USER_MARSHAL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write user marshal sign for required ruby version: %w", err)
	}
	err = buff.WriteByte(SYMBOL_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol link sign for required ruby version: %w", err)
	}
	err = buff.WriteByte(9)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol link for required ruby version: %w", err)
	}
	err = encArrayNoCache(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array for required ruby version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write inner array sign for required ruby version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner array length for required ruby version: %w", err)
	}
	err = encStringNoCache(buff, "2.6.0", &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode required ruby version string: %w", err)
	}

	// Required Rubygems Version
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for required rubygems version: %w", err)
	}
	err = encInt(buff, 26) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for required rubygems version: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for required rubygems version: %w", err)
	}
	_, err = buff.WriteString("required_rubygems_version")
	if err != nil {
		return nil, fmt.Errorf("failed to write required rubygems version: %w", err)
	}
	err = buff.WriteByte(USER_MARSHAL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write user marshal sign for required rubygems version: %w", err)
	}
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for Gem::Requirement: %w", err)
	}
	err = encInt(buff, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for Gem::Requirement: %w", err)
	}
	_, err = buff.WriteString("Gem::Requirement")
	if err != nil {
		return nil, fmt.Errorf("failed to write Gem::Requirement: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for required rubygems version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for required rubygems version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write inner array sign for required rubygems version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner array length for required rubygems version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write inner inner array sign for required rubygems version: %w", err)
	}
	err = encInt(buff, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inner inner array length for required rubygems version: %w", err)
	}
	err = encStringNoCache(buff, ">=", &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode required rubygems version operator: %w", err)
	}
	err = buff.WriteByte(USER_MARSHAL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write user marshal sign for required rubygems version: %w", err)
	}
	err = buff.WriteByte(SYMBOL_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol link sign for required rubygems version: %w", err)
	}
	err = buff.WriteByte(9)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol link for required rubygems version: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for required rubygems version: %w", err)
	}
	err = encInt(buff, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for required rubygems version: %w", err)
	}
	err = encStringNoCache(buff, "3.3.3", &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode required rubygems version string: %w", err)
	}

	// Original Platform
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for original platform: %w", err)
	}
	err = encInt(buff, 18) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for original platform: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for original platform: %w", err)
	}
	_, err = buff.WriteString("original_platform")
	if err != nil {
		return nil, fmt.Errorf("failed to write original platform: %w", err)
	}
	err = encStringNoCache(buff, meta.Platform, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem original platform: %w", err)
	}

	// Email
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for email: %w", err)
	}
	err = encInt(buff, 6)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for email: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for email: %w", err)
	}
	_, err = buff.WriteString("email")
	if err != nil {
		return nil, fmt.Errorf("failed to write email: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for email: %w", err)
	}
	arrlen := len(meta.Emails)
	err = encInt(buff, arrlen) // Length of array
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for email: %w", err)
	}
	for _, email := range meta.Emails {
		err = encStringNoCache(buff, email, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem email: %w", err)
		}
	}

	// Authors
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for authors: %w", err)
	}
	err = buff.WriteByte(13)
	if err != nil {
		return nil, fmt.Errorf("failed to write length of symbol for authors: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for authors: %w", err)
	}
	_, err = buff.WriteString("authors")
	if err != nil {
		return nil, fmt.Errorf("failed to write authors: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for authors: %w", err)
	}
	arrlen = len(meta.Authors)
	err = encInt(buff, arrlen)
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for authors: %w", err)
	}
	for _, author := range meta.Authors {
		err = encStringNoCache(buff, author, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem author: %w", err)
		}
	}

	// Description
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for description: %w", err)
	}
	err = encInt(buff, 12) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for description: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for description: %w", err)
	}
	_, err = buff.WriteString("description")
	if err != nil {
		return nil, fmt.Errorf("failed to write description: %w", err)
	}
	err = encStringNoCache(buff, meta.Description, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem description: %w", err)
	}

	// Homepage
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for homepage: %w", err)
	}
	err = encInt(buff, 9) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for homepage: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for homepage: %w", err)
	}
	_, err = buff.WriteString("homepage")
	if err != nil {
		return nil, fmt.Errorf("failed to write homepage: %w", err)
	}
	err = encStringNoCache(buff, meta.Homepage, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gem homepage: %w", err)
	}

	// Licenses
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for licenses: %w", err)
	}
	err = encInt(buff, 9)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for licenses: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for licenses: %w", err)
	}
	_, err = buff.WriteString("licenses")
	if err != nil {
		return nil, fmt.Errorf("failed to write licenses: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for licenses: %w", err)
	}
	arrlen = len(meta.Licenses)
	err = encInt(buff, arrlen) // Length of array
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for licenses: %w", err)
	}
	for _, lic := range meta.Licenses {
		err = encStringNoCache(buff, lic, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem license: %w", err)
		}
	}

	// Require Paths
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for require_paths: %w", err)
	}
	err = encInt(buff, 14)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for require_paths: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for require_paths: %w", err)
	}
	_, err = buff.WriteString("require_paths")
	if err != nil {
		return nil, fmt.Errorf("failed to write require_paths: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for require_paths: %w", err)
	}
	arrlen = len(meta.RequirePaths)
	err = encInt(buff, arrlen) // Length of array
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for require_paths: %w", err)
	}
	for _, rp := range meta.RequirePaths {
		err = encStringNoCache(buff, rp, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode gem require_path: %w", err)
		}
	}

	// Specification Version
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for specification_version: %w", err)
	}
	err = encInt(buff, 22) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for specification_version: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for specification_version: %w", err)
	}
	_, err = buff.WriteString("specification_version")
	if err != nil {
		return nil, fmt.Errorf("failed to write specification_version: %w", err)
	}
	err = buff.WriteByte(FIXNUM_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write fixnum sign for specification_version: %w", err)
	}
	err = encInt(buff, meta.SpecVersion) //specification version value
	if err != nil {
		return nil, fmt.Errorf("failed to encode specification_version value: %w", err)
	}

	// Dependencies
	err = buff.WriteByte(SYMBOL_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write symbol sign for dependencies: %w", err)
	}
	err = encInt(buff, 13) //Length of symbol + 1 for the '@' character
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol length for dependencies: %w", err)
	}
	err = buff.WriteByte(OBJECT_LINK_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write object link sign for dependencies: %w", err)
	}

	_, err = buff.WriteString("dependencies")
	if err != nil {
		return nil, fmt.Errorf("failed to write dependencies: %w", err)
	}
	err = buff.WriteByte(ARRAY_SIGN)
	if err != nil {
		return nil, fmt.Errorf("failed to write array sign for dependencies: %w", err)
	}
	arrlen = len(meta.Dependencies)
	err = encInt(buff, arrlen) // Length of arr
	if err != nil {
		return nil, fmt.Errorf("failed to encode array length for dependencies: %w", err)
	}
	for _, dep := range meta.Dependencies {
		err = buff.WriteByte(OBJECT_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object sign for dependency: %w", err)
		}
		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for dependency: %w", err)
		}
		err = encInt(buff, 15)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for dependency: %w", err)
		}
		_, err = buff.WriteString("Gem::Dependency")
		if err != nil {
			return nil, fmt.Errorf("failed to write Gem::Dependency: %w", err)
		}
		err = buff.WriteByte(10)
		if err != nil {
			return nil, fmt.Errorf("failed to write number of instance variables for dependency: %w", err)
		}
		err = buff.WriteByte(SYMBOL_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol link sign for dependency: %w", err)
		}
		err = buff.WriteByte(6)
		if err != nil {
			return nil, fmt.Errorf("failed to write length of symbol link for dependency: %w", err)
		}
		err = encStringNoCache(buff, dep.Name, &olinkidx, slinktbl, &slinkidx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode dependency name: %w", err)
		}

		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for version: %w", err)
		}
		err = encInt(buff, 12) //Length of symbol + 1 for the '@' character
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for version: %w", err)
		}
		err = buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object link sign for version: %w", err)
		}
		_, err = buff.WriteString("requirement")
		if err != nil {
			return nil, fmt.Errorf("failed to write requirement: %w", err)
		}
		err = buff.WriteByte(USER_MARSHAL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write user marshal sign for requirement: %w", err)
		}
		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for Gem::Requirement: %w", err)
		}
		err = encInt(buff, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for Gem::Requirement: %w", err)
		}
		_, err = buff.WriteString("Gem::Requirement")
		if err != nil {
			return nil, fmt.Errorf("failed to write Gem::Requirement: %w", err)
		}
		err = buff.WriteByte(ARRAY_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write array sign for requirement: %w", err)
		}
		err = encInt(buff, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to encode array length for requirement: %w", err)
		}
		err = buff.WriteByte(ARRAY_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write inner array sign for requirement: %w", err)
		}
		err = encInt(buff, len(dep.Requirement.VersionConstraints))
		if err != nil {
			return nil, fmt.Errorf("failed to encode inner array length for requirement: %w", err)
		}
		for _, vc := range dep.Requirement.VersionConstraints {
			err = buff.WriteByte(ARRAY_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write array sign for version constraint: %w", err)
			}
			err = encInt(buff, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to encode inner inner array length for version constraint: %w", err)
			}
			err = encStringNoCache(buff, vc.Constraint, &olinkidx, slinktbl, &slinkidx)
			if err != nil {
				return nil, fmt.Errorf("failed to encode version constraint: %w", err)
			}

			err = buff.WriteByte(USER_MARSHAL_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write user marshal sign for version constraint: %w", err)
			}
			err = buff.WriteByte(SYMBOL_LINK_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write symbol link sign for version constraint: %w", err)
			}
			err = encInt(buff, 4)
			if err != nil {
				return nil, fmt.Errorf("failed to encode symbol length for version constraint: %w", err)
			}

			err = buff.WriteByte(ARRAY_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write array sign for version constraint: %w", err)
			}
			err = encInt(buff, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to encode inner array length for version constraint: %w", err)
			}
			err = encStringNoCache(buff, vc.Version, &olinkidx, slinktbl, &slinkidx)
			if err != nil {
				return nil, fmt.Errorf("failed to encode version constraint version: %w", err)
			}

		}
		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for type: %w", err)
		}
		err = encInt(buff, 5) //Length of symbol + 1 for the '@' character
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for type: %w", err)
		}
		err = buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object link sign for type: %w", err)
		}
		_, err = buff.WriteString("type")
		if err != nil {
			return nil, fmt.Errorf("failed to write type: %w", err)
		}
		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for type value: %w", err)
		}
		strlen := len(dep.Type) - 1
		err = encInt(buff, strlen)
		if err != nil {
			return nil, fmt.Errorf("failed to encode type length: %w", err)
		}
		_, err = buff.WriteString(dep.Type[1:])
		if err != nil {
			return nil, fmt.Errorf("failed to write type value: %w", err)
		}
		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for prerelease: %w", err)
		}
		err = encInt(buff, 11) //Length of symbol + 1 for the '@' character
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for prerelease: %w", err)
		}
		err = buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object link sign for prerelease: %w", err)
		}
		_, err = buff.WriteString("prerelease")
		if err != nil {
			return nil, fmt.Errorf("failed to write prerelease: %w", err)
		}
		if dep.Prerelease {
			err = buff.WriteByte(TRUE_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write true sign for prerelease: %w", err)
			}
		} else {
			err = buff.WriteByte(FALSE_SIGN)
			if err != nil {
				return nil, fmt.Errorf("failed to write false sign for prerelease: %w", err)
			}

		}

		err = buff.WriteByte(SYMBOL_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write symbol sign for version_requirements: %w", err)
		}
		err = encInt(buff, 21) //Length of symbol + 1 for the '@' character
		if err != nil {
			return nil, fmt.Errorf("failed to encode symbol length for version_requirements: %w", err)
		}
		err = buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object link sign for version_requirements: %w", err)
		}
		_, err = buff.WriteString("version_requirements")
		if err != nil {
			return nil, fmt.Errorf("failed to write version_requirements: %w", err)
		}
		err = buff.WriteByte(OBJECT_LINK_SIGN)
		if err != nil {
			return nil, fmt.Errorf("failed to write object link sign for version_requirements: %w", err)
		}
		err = buff.WriteByte(EMPTY_STRING)
		if err != nil {
			return nil, fmt.Errorf("failed to write empty string for version_requirements: %w", err)
		}
	}

	// Rubygems version
	err = encSymbol(buff, []byte("rubygems_version"), slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode symbol 'rubygems_version': %w", err)
	}
	err = encStringNoCache(buff, meta.RubygemsVersion, &olinkidx, slinktbl, &slinkidx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode rubygems version: %w", err)
	}
	return buff.Bytes(), nil
}

// TODO: Fix reads so "_" gem doesnt end up as "\fGem::Version"
func LoadSpecs(src io.Reader) ([]*spec.Spec, error) {
	var specs []*spec.Spec
	var slinktbl [][]byte
	var olinktbl [][]byte
	reader := bufio.NewReader(src)
	_, err := reader.ReadByte() // Major version
	if err != nil {
		return nil, fmt.Errorf("failed to read major version: %w", err)
	}
	_, err = reader.ReadByte() // Minor version
	if err != nil {
		return nil, fmt.Errorf("failed to read major and minor version: %w", err)
	}
	_, err = reader.ReadByte() // Array sign
	if err != nil {
		return nil, fmt.Errorf("failed to read array sign: %w", err)
	}

	osize, err := readInt(reader) // Outer Array Len
	if err != nil {
		return nil, fmt.Errorf("failed to read outer array size: %w", err)
	}
	i := 0
	for i < int(osize) {
		b, err := reader.ReadByte() // Array sign
		if err != nil {
			return nil, fmt.Errorf("failed to read array sign: %w", err)
		}
		if b != ARRAY_SIGN {
			return nil, fmt.Errorf("expected ARRAY_SIGN but got %c", b)
		}
		olinktbl = append(olinktbl, []byte{'['})
		isize, err := readInt(reader) // Inner array len (3)
		if err != nil || isize != 3 {
			return nil, fmt.Errorf("expected inner array size of 3 but got %d: %w", isize, err)
		}
		name, err := readName(reader, &slinktbl, &olinktbl)
		if err != nil {
			return nil, fmt.Errorf("failed to read name: %w", err)
		}
		version, err := readVersion(reader, &slinktbl, &olinktbl)
		if err != nil {
			return nil, fmt.Errorf("failed to read version: %w", err)
		}
		platform, err := readPlatform(reader, &slinktbl, &olinktbl)
		if err != nil {
			return nil, fmt.Errorf("failed to read platform: %w", err)
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
	return specs, nil
}

func readName(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	b, err := r.ReadByte() // IVAR
	if err != nil {
		return "", fmt.Errorf("failed to read IVAR: %w", err)
	}
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // RAWSTRING
	if err != nil {
		return "", fmt.Errorf("failed to read RAWSTRING: %w", err)
	}
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
		if err != nil {
			return "", fmt.Errorf("failed to read name byte: %w", err)
		}
		nameBytes = append(nameBytes, b)
		j++
	}
	*olinktbl = append(*olinktbl, nameBytes)
	_, err = r.ReadByte() // 6
	if err != nil {
		return "", fmt.Errorf("failed to read 6 after name: %w", err)
	}
	b, err = r.ReadByte() // Symbol sign
	if err != nil {
		return "", fmt.Errorf("failed to read symbol sign: %w", err)
	}
	if b != SYMBOL_SIGN && b != SYMBOL_LINK_SIGN {
		return string(b), errors.New("")
	}
	if b == SYMBOL_LINK_SIGN {
		_, err = r.ReadByte() // 0
		if err != nil {
			return "", fmt.Errorf("failed to read symbol link sign: %w", err)
		}
	} else {
		len, _ := r.ReadByte() // 6
		sym, _ := r.ReadByte() // E

		*slinktbl = append(*slinktbl, []byte{len, sym})
	}
	b, err = r.ReadByte() // TRUE sign
	if err != nil {
		return "", fmt.Errorf("failed to read TRUE sign: %w", err)
	}
	if b != TRUE_SIGN {
		return string(b), errors.New("")
	}
	return string(nameBytes), nil
}

func readVersion(r *bufio.Reader, slinktbl *[][]byte, olinktbl *[][]byte) (string, error) {
	var versionBytes []byte
	var i int
	b, err := r.ReadByte() // U
	if err != nil {
		return "", fmt.Errorf("failed to read U: %w", err)
	}
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != 'U' {
		b, err = r.ReadByte()
		if err != nil {
			return "", fmt.Errorf("failed to read next byte after U: %w", err)
		}
		return string(b), errors.New("not u")
	}
	b, err = r.ReadByte() // Symbol sign
	if err != nil {
		return "", fmt.Errorf("failed to read symbol sign: %w", err)
	}
	if b != SYMBOL_SIGN && b != SYMBOL_LINK_SIGN {
		return string(b), errors.New("not symbol or link")
	}
	if b == SYMBOL_LINK_SIGN {
		_, err = r.ReadByte() // 0
		if err != nil {
			return "", fmt.Errorf("failed to read symbol link sign: %w", err)
		}
	} else {
		strlen, _ := readInt(r) // Length of string
		tmp := []byte{byte(strlen)}
		for i < int(strlen) {
			b, err = r.ReadByte()
			if err != nil {
				return "", fmt.Errorf("failed to read version byte: %w", err)
			}
			tmp = append(tmp, b)
			i++
		}
		*slinktbl = append(*slinktbl, tmp)
	}

	b, err = r.ReadByte() // Array sign
	if err != nil {
		return "", fmt.Errorf("failed to read array sign: %w", err)
	}
	if b != ARRAY_SIGN {
		return string(b), errors.New("not array")
	}
	_, err = r.ReadByte() // Array len (6 aka 1)
	if err != nil {
		return "", fmt.Errorf("failed to read array length: %w", err)
	}
	b, err = r.ReadByte() // IVAR
	if err != nil {
		return "", fmt.Errorf("failed to read IVAR: %w", err)
	}
	if b != IVAR_SIGN {
		return string(b), errors.New("not ivar")
	}
	b, err = r.ReadByte() // RAWSTRING
	if err != nil {
		return "", fmt.Errorf("failed to read RAWSTRING: %w", err)
	}
	if b != RAWSTRING_SIGN {
		return string(b), errors.New("not string")
	}
	strlen, _ := readInt(r) // Length of version string
	i = 0
	for i < int(strlen) {
		b, err = r.ReadByte()
		if err != nil {
			return "", fmt.Errorf("failed to read version byte: %w", err)
		}
		versionBytes = append(versionBytes, b)
		i++
	}
	*olinktbl = append(*olinktbl, versionBytes)
	_, err = r.ReadByte() // 1
	if err != nil {
		return "", fmt.Errorf("failed to read 1 after version: %w", err)
	}
	b, err = r.ReadByte() // Symbol Link sign
	if err != nil {
		return "", fmt.Errorf("failed to read symbol link sign: %w", err)
	}
	if b != SYMBOL_LINK_SIGN {
		return string(b), errors.New("")
	}
	_, err = r.ReadByte() // 0
	if err != nil {
		return "", fmt.Errorf("failed to read 0 after symbol link sign: %w", err)
	}
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
	if err != nil {
		return "", fmt.Errorf("failed to read IVAR: %w", err)
	}
	if b == OBJECT_LINK_SIGN {
		return readObjectLink(r, olinktbl)
	}
	if b != IVAR_SIGN {
		return string(b), errors.New("")
	}
	b, err = r.ReadByte() // RAWSTR
	if err != nil {
		return "", fmt.Errorf("failed to read RAWSTR: %w", err)
	}
	if b != RAWSTRING_SIGN {
		return string(b), errors.New("")
	}
	strlen, _ := readInt(r) // length of platform string
	var platformBytes []byte
	j := 0
	for j < int(strlen) {
		b, err = r.ReadByte()
		if err != nil {
			return "", fmt.Errorf("failed to read platform byte: %w", err)
		}
		platformBytes = append(platformBytes, b)
		j++
	}
	*olinktbl = append(*olinktbl, platformBytes)
	_, err = r.ReadByte() // 6
	if err != nil {
		return "", fmt.Errorf("failed to read 6 after platform: %w", err)
	}
	b, err = r.ReadByte() // Symbol link sign
	if err != nil {
		return "", fmt.Errorf("failed to read symbol link sign: %w", err)
	}
	if b != SYMBOL_LINK_SIGN {
		return string(b), errors.New("")
	}
	_, err = r.ReadByte() // 0
	if err != nil {
		return "", fmt.Errorf("failed to read 0 after symbol link sign: %w", err)
	}
	// b, err = r.ReadByte() // E
	b, err = r.ReadByte() // TRUE sign
	if b != TRUE_SIGN {
		return string(b), errors.New("")
	}
	return string(platformBytes), err
}

func readObjectLink(r *bufio.Reader, olinktbl *[][]byte) (string, error) {
	idx, err := readInt(r)
	if err != nil {
		return "", fmt.Errorf("failed to read object link index: %w", err)
	}
	idx = idx - 1 // First index is 1
	tmp := (*olinktbl)[idx]
	return string(tmp), nil
}

func readInt(r *bufio.Reader) (int, error) {
	var result int
	b, err := r.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("failed to read byte: %w", err)
	}
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
			n, err := r.ReadByte()
			if err != nil {
				return 0, fmt.Errorf("failed to read byte for int: %w", err)
			}
			result |= int(uint(n) << (8 * uint(i)))
		}
	} else {
		result = -1
		c = -c
		for i := 0; i < c; i++ {
			n, err := r.ReadByte()
			if err != nil {
				return 0, fmt.Errorf("failed to read byte for int: %w", err)
			}
			result &= ^(0xff << uint(8*i))
			result |= int(n) << uint(8*i)
		}
	}
	return result, nil
}
