package spec

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

// spec.original_name
// spec.version.prerelease?
// spec.original_platform
// spec.full_name # used for logging
// spec.name
// spec.version (This is an object of type Gem::Version)

type Spec struct {
	OriginalName     string
	OriginalPlatform string
	FullName         string
	Name             string
	Version          string
	PreRelease       bool
	LoadedFrom       string
}

type gemMetadata struct {
	Name     string `yaml:"name"`
	Platform string `yaml:"platform"`
	Version  struct {
		Version string `yaml:"version"`
	}
}

func untar(full_name string, gemfile string) string {
	tmpdir, err := os.MkdirTemp("/tmp", full_name)
	if err != nil {
		panic(err)
	}
	fmt.Println("Temp dir name:", tmpdir)
	file, err := os.Open(gemfile)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
			fmt.Println("Untarring :", filename)
			writer, err := os.Create(filename)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			io.Copy(writer, tarBallReader)

			err = os.Chmod(filename, os.FileMode(header.Mode))

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			writer.Close()
		default:
			fmt.Printf("Unable to untar type : %c in file %s", header.Typeflag, filename)
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

func New(gemfile string) *Spec {
	path_chunks := strings.Split(gemfile, "/")
	full := path_chunks[len(path_chunks)-1]
	ogName := strings.TrimSuffix(full, ".gem")
	tmpdir := untar(full, gemfile)
	defer os.RemoveAll(tmpdir)
	res := GunzipMetadata(tmpdir)
	var metadata gemMetadata
	err := yaml.Unmarshal([]byte(res), &metadata)
	if err != nil {
		panic(err)
	}
	s := Spec{OriginalName: ogName, OriginalPlatform: metadata.Platform, FullName: full, Name: metadata.Name, Version: metadata.Version.Version, PreRelease: false, LoadedFrom: full}
	fmt.Println(s)
	return &s
}

func PartitionSpecs(specs []*Spec, inc_latest bool) ([]*Spec, []*Spec, []*Spec) {
	var prerelease []*Spec
	var released []*Spec
	var latest []*Spec
	hash := make(map[string][]*Spec)
	for _, s := range specs {
		match, _ := regexp.MatchString("[a-zA-Z]", s.Version)
		if match {
			prerelease = append(prerelease, s)
		} else {
			released = append(released, s)
			if inc_latest {
				hash[s.Name] = append(hash[s.Name], s)
			}
		}
	}

	if inc_latest {
		for _, v := range hash {
			latest = append(latest, v[len(v)-1])
		}
	}
	return prerelease, released, latest
}
