package indexer

import (
	"fmt"
	"github.com/gscho/gemfast/internal/marshal"
	"github.com/gscho/gemfast/internal/spec"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"compress/gzip"
	"io/ioutil"
	"bytes"
)

type Indexer struct {
	destDir            string
	tmpDir             string
	marshalIdx         string
	quickDir           string
	quickMarshalDir    string
	quickIdx           string
	latestIdx          string
	specsIdx           string
	latestSpecsIdx     string
	prereleaseSpecsIdx string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func New(destDir string) *Indexer {
	marshalName := "Marshal.4.8"
	i := Indexer{destDir: destDir}
	i.tmpDir = i.mkTempDirs("gem_generate_index")
	i.marshalIdx = marshalName
	i.quickDir = fmt.Sprintf("%s/quick", i.tmpDir)
	i.quickMarshalDir = fmt.Sprintf("%s/%s", i.quickDir, marshalName)
	err := os.MkdirAll(i.quickMarshalDir, os.ModePerm)
	check(err)
	i.quickIdx = fmt.Sprintf("%s/index", i.quickDir)
	i.latestIdx = fmt.Sprintf("%s/latest_index", i.quickDir)
	i.specsIdx = "specs.4.8"
	i.latestSpecsIdx = "latest_specs.4.8"
	i.prereleaseSpecsIdx = "prerelease_specs.4.8"
	return &i
}

func (i Indexer) GenerateIndex() {
	i.buildIndicies()
}

func (i Indexer) mkTempDirs(name string) (tmpdir string) {
	dir, err := os.MkdirTemp("/tmp", name)
	check(err)
	fmt.Println("Temp dir name:", dir)
	// Makes the /tmp/gem_generate_index/quick/Marshal.4.8 directory
	// defer os.RemoveAll(dir)
	err = os.Chmod(dir, 0700)
	check(err)
	return dir
}

func (i Indexer) gemList() []string {
	var gems []string
	filepath.WalkDir(i.destDir, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ".gem" {
			gems = append(gems, s)
		}
		return nil
	})
	return gems
}

func (i Indexer) mapGemsToSpecs(gems []string) []*spec.Spec {
	fmt.Println(gems)
	var specs []*spec.Spec
	var s *spec.Spec
	for _, g := range gems {
		fi, err := os.Stat(g)
		check(err)
		if fi.Size() == 0 {
			fmt.Println("Skipping zero-length gem")
			continue
		} else {
			s = spec.New(g)
			specs = append(specs, s)
		}
	}
	return specs
}

// func (i Indexer) buildMarshalGemspecs(specs []*spec.Spec) {
// 	// var files []string
// 	for _, s := range specs {
// 		specFName := fmt.Sprintf("%s.gemspec.rz", s.OriginalName)
// 		marshalName := fmt.Sprintf("%s/%s", i.quickMarshalDir, specFName)
// 		fmt.Println(marshalName)
// 		dump := marshal.Dump(s)
// 		var in bytes.Buffer
// 		w := zlib.NewWriter(&in)
// 		w.Write(dump)
// 		w.Close()

// 		var out bytes.Buffer
// 		r, _ := zlib.NewReader(&in)
// 		io.Copy(&out, r)
// 		// os.Stdout.Write(out.Bytes())

// 		// Open a new file for writing only
// 		file, err := os.OpenFile(
// 			marshalName,
// 			os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
// 			0666,
// 		)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		defer file.Close()

// 		// Write bytes to file
// 		bytesWritten, err := file.Write(out.Bytes())
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		log.Printf("Wrote %d bytes.\n", bytesWritten)
// 	}
// }

func (i Indexer) buildModernIndices(specs []*spec.Spec) {
	pre, rel, latest := spec.PartitionSpecs(specs)
	i.buildModernIndex(rel, i.specsIdx, "specs")
	i.buildModernIndex(latest, i.latestSpecsIdx, "latest specs")
	i.buildModernIndex(pre, i.prereleaseSpecsIdx, "prerelease specs")
}

func (i Indexer) buildModernIndex(specs []*spec.Spec, idxFile string, name string) {
	file, err := os.OpenFile(
		idxFile,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	dump := marshal.DumpSpecs(specs)
	bytesWritten, err := file.Write(dump)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote %d bytes.\n", bytesWritten)
}

func (i Indexer) compressIndicies() {
	gzipFile(i.prereleaseSpecsIdx)
	gzipFile(i.specsIdx)
	gzipFile(i.latestSpecsIdx)
}

func gzipFile(src string) {
	content, err := ioutil.ReadFile(src) // just pass the file name
	if err != nil {
	  panic(err)
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	defer gz.Close() //NOT SUFFICIENT, DON'T DEFER WRITER OBJECTS
	if _, err := gz.Write(content); err != nil {
	  panic(err)
	}
	// NEED TO CLOSE EXPLICITLY
	if err := gz.Close(); err != nil {
	  panic(err)
	}
	ioutil.WriteFile(fmt.Sprintf("%s.gz", src), b.Bytes(), 0666)
}

func (i Indexer) buildIndicies() {
	specs := i.mapGemsToSpecs(i.gemList())
	// i.buildMarshalGemspecs(specs)
	i.buildModernIndices(specs)
	i.compressIndicies()
}
