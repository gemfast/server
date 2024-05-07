package spec

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
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
	GemMetadata      *GemMetadata
	Checksum         string
	Ruby             string //TODO: parse required_ruby_version from metadata
	RubyGems         string //TODO: parse required_rubygems_version from metadata
}

func untar(dir, fullName, gemfile string) ([]byte, string, error) {
	tmpdir, err := os.MkdirTemp(dir, fullName)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmpdir")
		return nil, "", err
	}
	log.Trace().Msg(fmt.Sprintf("created tmpdir %s", tmpdir))
	file, err := os.Open(gemfile)

	if err != nil {
		log.Error().Err(err).Str("detail", gemfile).Msg("failed to open gemfile")
		return nil, "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Error().Err(err).Str("detail", gemfile).Msg("failed to hash gemfile")
		return nil, "", err
	}
	file.Seek(0, 0) // reset file pointer to beginning of file

	var fileReader io.ReadCloser = file
	tarBallReader := tar.NewReader(fileReader)

	// Extracting tar.gz files
	for {
		header, err := tarBallReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Error().Err(err).Str("detail", fullName).Msg("bad header")
			return nil, "", err
		}

		// get the individual filename and extract to the current directory
		dest := fmt.Sprintf("%s/%s", tmpdir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// handle directory
			log.Trace().Str("detail", dest).Msg("extracting directory")
			err = os.MkdirAll(dest, os.FileMode(header.Mode))
			if err != nil {
				log.Error().Err(err).Str("detail", dest).Msg("failed to make directory")
				return nil, "", err
			}

		case tar.TypeReg:
			// handle normal file
			log.Trace().Msg(fmt.Sprintf("extracting file to %s", dest))
			writer, err := os.Create(dest)
			if err != nil {
				log.Error().Err(err).Str("detail", dest).Msg("failed to create files")
				return nil, "", err
			}
			defer writer.Close()

			io.Copy(writer, tarBallReader)
			err = os.Chmod(dest, os.FileMode(header.Mode))

			if err != nil {
				log.Error().Err(err).Str("detail", dest).Msg("failed to chmod file")
				return nil, "", err
			}
		default:
			log.Error().Err(err).Str("detail", dest).Bytes("type", []byte{header.Typeflag}).Msg("unrecognized file type")
			return nil, "", err
		}
	}
	return h.Sum(nil), tmpdir, nil
}

func GunzipMetadata(path string) (string, error) {
	meta := fmt.Sprintf("%s/metadata.gz", path)
	file, err := os.Open(meta)
	if err != nil {
		log.Error().Err(err).Str("detail", meta).Msg("failed to open file")
		return "", err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	gzreader, err := gzip.NewReader(fileReader)
	if err != nil {
		log.Error().Err(err).Str("detail", meta).Msg("failed to create gzip reader")
		return "", err
	}

	output, err := io.ReadAll(gzreader)
	if err != nil {
		log.Error().Err(err).Str("detail", meta).Msg("failed to read gzip content")
		return "", err
	}

	yaml := string(output)
	return yaml, nil
}

// TODO: break this method up into smaller methods
func ParseGemMetadata(yamlBytes []byte) (*GemMetadata, error) {
	var metadata GemMetadata
	err := yaml.Unmarshal(yamlBytes, &metadata)
	if err != nil {
		log.Error().Err(err).Msg("failed to unmarshal gem metadata")
		return &GemMetadata{}, err
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
			log.Trace().Str("detail", metadata.Name).Msg("nil email")
		}
	default:
		log.Error().Err(err).Str("detail", metadata.Name).Str("detail", fmt.Sprintf("%T", t)).Msg("unknown email type in gem metadata")
		return &GemMetadata{}, err
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
				log.Error().Err(err).Str("detail", metadata.Name).Str("detail", fmt.Sprintf("%T", t)).Msg("unknown requirements type in gem metadata")
				return &GemMetadata{}, err
			}
		}
		metadata.Dependencies[i] = dep
	}
	for _, req := range metadata.RequiredRubyVersion.Requirements {
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
						metadata.RequiredRubyVersion.VersionConstraints = append(metadata.RequiredRubyVersion.VersionConstraints, vc)
						c = ""
						v = ""
					}
				}
			}
		default:
			log.Error().Err(err).Str("detail", metadata.Name).Str("detail", fmt.Sprintf("%T", t)).Msg("unknown ruby requirement type in gem metadata")
			return &GemMetadata{}, err

		}
	}
	for _, req := range metadata.RequiredRubyGemsVersion.Requirements {
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
						metadata.RequiredRubyGemsVersion.VersionConstraints = append(metadata.RequiredRubyGemsVersion.VersionConstraints, vc)
						c = ""
						v = ""
					}
				}
			}
		default:
			log.Error().Err(err).Str("detail", metadata.Name).Str("detail", fmt.Sprintf("%T", t)).Msg("unknown rubygems requirement type in gem metadata")
			return &GemMetadata{}, err
		}
	}
	return &metadata, nil
}

func FromFile(dir, gemfile string) (*Spec, error) {
	chunks := strings.Split(gemfile, "/")
	full := chunks[len(chunks)-1]
	ogName := strings.TrimSuffix(full, ".gem")
	log.Trace().Str("detail", gemfile).Msg("untarring gemfile")
	sum, tmpdir, err := untar(dir, full, gemfile)
	log.Trace().Str("detail", fmt.Sprintf("%x", sum)).Msg("checksum of gem")
	defer os.RemoveAll(tmpdir)
	if err != nil {
		log.Error().Err(err).Str("detail", full).Msg("failed to untar gem")
		return &Spec{}, err
	}
	log.Trace().Str("detail", tmpdir).Msg("gunzip tmpdir")
	res, err := GunzipMetadata(tmpdir)
	if err != nil {
		log.Error().Err(err).Str("detail", full).Msg("failed to gunzip gem metadata")
		return &Spec{}, err
	}
	metadata, err := ParseGemMetadata([]byte(res))
	if err != nil {
		log.Error().Err(err).Str("detail", full).Msg("failed to parse gem metadata")
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
		Checksum:         fmt.Sprintf("%x", sum),
	}
	setRequiredRubyVersion(&s)
	setRequiredRubyGemsVersion(&s)
	return &s, nil
}

func setRequiredRubyVersion(s *Spec) {
	if len(s.GemMetadata.RequiredRubyVersion.VersionConstraints) > 0 {
		var toAdd []string
		for _, vc := range s.GemMetadata.RequiredRubyVersion.VersionConstraints {
			toAdd = append(toAdd, vc.Constraint+" "+vc.Version)
		}
		sort.Strings(toAdd)
		s.Ruby = strings.Join(toAdd, "&")
	}
}

func setRequiredRubyGemsVersion(s *Spec) {
	if len(s.GemMetadata.RequiredRubyGemsVersion.VersionConstraints) > 0 {
		var toAdd []string
		for _, vc := range s.GemMetadata.RequiredRubyGemsVersion.VersionConstraints {
			toAdd = append(toAdd, vc.Constraint+" "+vc.Version)
		}
		sort.Strings(toAdd)
		s.RubyGems = strings.Join(toAdd, "&")
	}
}

func PartitionSpecs(specs []*Spec) ([]*Spec, []*Spec, []*Spec) {
	var prerelease []*Spec
	var released []*Spec
	var latest []*Spec
	hash := make(map[string]*Spec)
	r := regexp.MustCompile("[a-zA-Z]")
	for _, s := range specs {
		match := r.MatchString(s.Version)
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
		log.Warn().Str("detail", fmt.Sprintf("%s %s %s", spec.Name, spec.Version, spec.OriginalPlatform)).Msg("spec")
		if spec.Name != s.Name {
			break
		} else if spec.Version == s.Version && spec.OriginalPlatform == s.OriginalPlatform {
			return i
		}
	}
	return -1
}
