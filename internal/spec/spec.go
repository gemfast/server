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

func untar(full_name string, gemfile string) string {
	tmpdir, err := os.MkdirTemp("/tmp", full_name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmpdir")
		panic(err)
	}
	log.Trace().Msg(fmt.Sprintf("created tmpdir %s", tmpdir))
	file, err := os.Open(gemfile)

	if err != nil {
		log.Error().Err(err).Str("file", gemfile).Msg("failed to open gemfile")
		panic(err)
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	tarBallReader := tar.NewReader(fileReader)

	// Extracting tarred files

	for {
		header, err := tarBallReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			os.Exit(1)
		}

		// get the individual filename and extract to the current directory
		filename := fmt.Sprintf("%s/%s", tmpdir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// handle directory
			fmt.Println("Creating directory :", filename)
			err = os.MkdirAll(filename, os.FileMode(header.Mode)) // or use 0755 if you prefer

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

		case tar.TypeReg:
			// handle normal file
			log.Trace().Msg(fmt.Sprintf("untarring %s", filename))
			writer, err := os.Create(filename)

			if err != nil {
				log.Error().Err(err).Str("file", filename).Msg("failed to untar file")
				panic(err)
			}

			io.Copy(writer, tarBallReader)

			err = os.Chmod(filename, os.FileMode(header.Mode))

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			writer.Close()
		default:
			log.Error().Err(err).Str("file", filename).Bytes("type", []byte{header.Typeflag}).Msg("unrecognized file type")
		}
	}
	return tmpdir
}

func GunzipMetadata(path string) string {
	file, err := os.Open(fmt.Sprintf("%s/metadata.gz", path))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	gzreader, e1 := gzip.NewReader(fileReader)
	if e1 != nil {
		fmt.Println(e1) // Maybe panic here, depends on your error handling.
	}

	output, e2 := ioutil.ReadAll(gzreader)
	if e2 != nil {
		fmt.Println(e2)
	}

	yaml := string(output)
	return yaml
}

func ParseGemMetadata(yamlBytes []byte) GemMetadata {
	var metadata GemMetadata
	err := yaml.Unmarshal(yamlBytes, &metadata)
	if err != nil {
		panic(err)
	}
	// var email string
	switch t := metadata.Email.(type) {
	case []interface{}:
		{
			for _, entry := range t {
				if t != nil {
					metadata.Emails = append(metadata.Emails, entry.(string))
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
		}
	default:
		panic(fmt.Sprintf("Unknown type: %T for email", t))
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
				panic(fmt.Sprintf("Unknown type: %T for gem requirement requirements", t))
			}
		}
		metadata.Dependencies[i] = dep
	}
	return metadata
}

func FromFile(gemfile string) *Spec {
	path_chunks := strings.Split(gemfile, "/")
	full := path_chunks[len(path_chunks)-1]
	ogName := strings.TrimSuffix(full, ".gem")
	tmpdir := untar(full, gemfile)
	defer os.RemoveAll(tmpdir)
	res := GunzipMetadata(tmpdir)
	metadata := ParseGemMetadata([]byte(res))
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
	return &s
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
