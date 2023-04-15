package spec

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/rs/zerolog/log"
)

type Spec struct {
	OriginalName     string
	OriginalPlatform string
	FullName         string
	Name             string
	Version          string
	PreRelease       bool
	LoadedFrom       string
	GemMetadata      GemMetadata
}

func untar(full_name string, gemfile string) (string, error) {
	tmpdir, err := os.MkdirTemp("/tmp", full_name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmpdir")
		return "", err
	}
	log.Trace().Msg(fmt.Sprintf("created tmpdir %s", tmpdir))
	file, err := os.Open(gemfile)

	if err != nil {
		log.Error().Err(err).Str("file", gemfile).Msg("failed to open gemfile")
		return "", err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	tarBallReader := tar.NewReader(fileReader)

	// Extracting tar.gz files
	for {
		header, err := tarBallReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Error().Err(err).Str("gem", full_name).Msg("bad header")
			return "", err
		}

		// get the individual filename and extract to the current directory
		filename := fmt.Sprintf("%s/%s", tmpdir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// handle directory
			log.Trace().Str("dir", filename).Msg("extracting directory")
			err = os.MkdirAll(filename, os.FileMode(header.Mode))
			if err != nil {
				log.Error().Err(err).Str("dir", filename).Msg("failed to make directory")
				return "", err
			}

		case tar.TypeReg:
			// handle normal file
			log.Trace().Msg(fmt.Sprintf("extracting file %s", filename))
			writer, err := os.Create(filename)
			if err != nil {
				log.Error().Err(err).Str("file", filename).Msg("failed to create file")
				return "", err
			}
			defer writer.Close()

			io.Copy(writer, tarBallReader)
			err = os.Chmod(filename, os.FileMode(header.Mode))

			if err != nil {
				log.Error().Err(err).Str("file", filename).Msg("failed to chmod file")
				return "", err
			}
		default:
			log.Error().Err(err).Str("file", filename).Bytes("type", []byte{header.Typeflag}).Msg("unrecognized file type")
			return "", err
		}
	}
	return tmpdir, nil
}

func GunzipMetadata(path string) (string, error) {
	fname := fmt.Sprintf("%s/metadata.gz", path)
	file, err := os.Open(fname)
	if err != nil {
		log.Error().Err(err).Str("file", fname).Msg("failed to open file")
		return "", err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	gzreader, err := gzip.NewReader(fileReader)
	if err != nil {
		log.Error().Err(err).Str("file", fname).Msg("failed to create gzip reader")
		return "", err
	}

	output, err := ioutil.ReadAll(gzreader)
	if err != nil {
		log.Error().Err(err).Str("file", fname).Msg("failed to read gzip content")
		return "", err
	}

	yaml := string(output)
	return yaml, nil
}

func ParseGemMetadata(yamlBytes []byte) (GemMetadata, error) {
	var metadata GemMetadata
	err := yaml.Unmarshal(yamlBytes, &metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal gem metadata")
		return GemMetadata{}, err
	}
	// var email string
	switch t := metadata.Email.(type) {
	case []interface{}:
		{
			for _, entry := range t {
				if t != nil {
					email, ok := entry.(string)
					if ok {
						metadata.Emails = append(metadata.Emails, email)
					}
				}
			}
		}
	case interface{}:
		{
			metadata.Emails = append(metadata.Emails, t.(string))
		}
	case nil:
		{
			// nothing
			log.Trace().Str("gem", metadata.Name).Msg("nil email")
		}
	default:
		log.Error().Err(err).Str("gem", metadata.Name).Str("type", fmt.Sprintf("%T", t)).Msg("unknown email type in gem metadata")
		return GemMetadata{}, err
	}

	var c string
	var v string
	for i, dep := range metadata.Dependencies {
		for _, req := range dep.Requirement.Requirements {
			switch t := req.(type) {
			case []interface{}:
				{
					for _, entry := range t {
						if fmt.Sprintf("%T", entry) == "string" {
							c = fmt.Sprintf("%s", entry)
						} else {
							vmap := entry.(map[string]interface{})
							v = fmt.Sprintf("%s", vmap["version"])

						}
						if c != "" && v != "" {
							vc := VersionContraint{
								Constraint: c,
								Version:    v,
							}
							dep.Requirement.VersionConstraints = append(dep.Requirement.VersionConstraints, vc)
							c = ""
							v = ""
						}
					}
				}
			default:
				log.Error().Err(err).Str("gem", metadata.Name).Str("type", fmt.Sprintf("%T", t)).Msg("unknown requirements type in gem metadata")
				return GemMetadata{}, err
			}
		}
		metadata.Dependencies[i] = dep
	}
	return metadata, nil
}

func FromFile(gemfile string) (*Spec, error) {
	path_chunks := strings.Split(gemfile, "/")
	full := path_chunks[len(path_chunks)-1]
	ogName := strings.TrimSuffix(full, ".gem")
	log.Trace().Str("gemfile", gemfile).Msg("untarring gemfile")
	tmpdir, err := untar(full, gemfile)
	defer os.RemoveAll(tmpdir)
	if err != nil {
		log.Error().Err(err).Str("gem", full).Msg("failed to untar gem")
		return &Spec{}, err
	}
	log.Trace().Str("tmpdir", tmpdir).Msg("gunzip tmpdir")
	res, err := GunzipMetadata(tmpdir)
	if err != nil {
		log.Error().Err(err).Str("gem", full).Msg("failed to gunzip gem metadata")
		return &Spec{}, err
	}
	metadata, err := ParseGemMetadata([]byte(res))
	if err != nil {
		log.Error().Err(err).Str("gem", full).Msg("failed to parse gem metadata")
		return &Spec{}, err
	}
	s := Spec{
		OriginalName:     ogName,
		OriginalPlatform: metadata.Platform,
		FullName:         full,
		Name:             metadata.Name,
		Version:          metadata.Version.Version,
		PreRelease:       false,
		LoadedFrom:       full,
		GemMetadata:      metadata,
	}
	return &s, nil
}

func PartitionSpecs(specs []*Spec) ([]*Spec, []*Spec, []*Spec) {
	var prerelease []*Spec
	var released []*Spec
	var latest []*Spec
	hash := make(map[string]*Spec)
	for _, s := range specs {
		match, _ := regexp.MatchString("[a-zA-Z]", s.Version)
		if match {
			prerelease = append(prerelease, s)
		} else {
			released = append(released, s)
			if hash[s.Name] != nil {
				if hash[s.Name].Version < s.Version {
					hash[s.Name] = s
				}
			} else {
				hash[s.Name] = s
			}
		}
	}

	for _, v := range hash {
		latest = append(latest, v)
	}
	return prerelease, released, latest
}

func FindIndexOf(specs []*Spec, s *Spec) int {
	idx := func() int {
		k := s.Name
		l := 0
		h := len(specs) - 1
		for l <= h {
			mid := (l + h) / 2

			if specs[mid].Name == k {
				return mid
			} else if specs[mid].Name < k {
				l = mid + 1
			} else {
				h = mid - 1
			}
		}
		return -1
	}()
	if idx == -1 {
		return idx
	}
	for i := idx; i < len(specs); i++ {
		spec := specs[i]
		if spec.Name != s.Name {
			break
		} else if spec.Version == s.Version && spec.OriginalPlatform == s.OriginalPlatform {
			return i
		}
	}
	return -1
}
