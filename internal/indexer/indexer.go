package indexer

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/gscho/gemfast/internal/marshal"
	"github.com/gscho/gemfast/internal/spec"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type Indexer struct {
	destDir                string
	dir                    string
	marshalIdx             string
	quickDir               string
	quickMarshalDir        string
	quickMarshalDirBase    string
	quickIdx               string
	latestIdx              string
	specsIdx               string
	latestSpecsIdx         string
	prereleaseSpecsIdx     string
	destSpecsIdx           string
	destLatestSpecsIdx     string
	destPrereleaseSpecsIdx string
	files                  []string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func New(destDir string) *Indexer {
	marshalName := "Marshal.4.8"
	indexer := Indexer{destDir: destDir}
	indexer.dir = mkTempDir("gem_generate_index")
	indexer.marshalIdx = fmt.Sprintf("%s/%s", indexer.dir, marshalName)
	indexer.quickDir = fmt.Sprintf("%s/quick", indexer.dir)
	indexer.quickMarshalDir = fmt.Sprintf("%s/%s", indexer.quickDir, marshalName)
	indexer.quickMarshalDirBase = fmt.Sprintf("quick/%s", marshalName)
	indexer.quickIdx = fmt.Sprintf("%s/index", indexer.quickDir)
	indexer.latestIdx = fmt.Sprintf("%s/latest_index", indexer.quickDir)
	indexer.specsIdx = fmt.Sprintf("%s/specs.4.8", indexer.dir)
	indexer.latestSpecsIdx = fmt.Sprintf("%s/latest_specs.4.8", indexer.dir)
	indexer.prereleaseSpecsIdx = fmt.Sprintf("%s/prerelease_specs.4.8", indexer.dir)
	indexer.destSpecsIdx = fmt.Sprintf("%s/specs.4.8", indexer.destDir)
	indexer.destLatestSpecsIdx = fmt.Sprintf("%s/latest_specs.4.8", indexer.destDir)
	indexer.destPrereleaseSpecsIdx = fmt.Sprintf("%s/prerelease_specs.4.8", indexer.destDir)
	indexer.files = []string{indexer.quickMarshalDir}
	indexer.files = append(indexer.files, indexer.specsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.specsIdx))
	indexer.files = append(indexer.files, indexer.latestSpecsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.latestSpecsIdx))
	indexer.files = append(indexer.files, indexer.prereleaseSpecsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.prereleaseSpecsIdx))
	return &indexer
}

func (indexer Indexer) GenerateIndex() {
	indexer.mkTempDirs()
	indexer.buildIndicies()
	indexer.installIndicies()
	defer os.RemoveAll(indexer.dir)
}

func mkTempDir(name string) (tmpdir string) {
	dir, err := os.MkdirTemp("/tmp", name)
	check(err)
	fmt.Println("Temp dir name:", dir)
	// Makes the /tmp/gem_generate_index/quick/Marshal.4.8 directory
	// defer os.RemoveAll(dir)
	err = os.Chmod(dir, 0700)
	check(err)
	return dir
}

func (indexer Indexer) mkTempDirs() {
	err := os.MkdirAll(indexer.quickMarshalDir, os.ModePerm)
	check(err)
}

func (indexer Indexer) gemList() []string {
	var gems []string
	filepath.WalkDir(indexer.destDir, func(s string, d fs.DirEntry, e error) error {
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

func mapGemsToSpecs(gems []string) []*spec.Spec {
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

func (indexer Indexer) buildModernIndices(specs []*spec.Spec) {
	pre, rel, latest := spec.PartitionSpecs(specs)
	buildModernIndex(rel, indexer.specsIdx, "specs")
	buildModernIndex(latest, indexer.latestSpecsIdx, "latest specs")
	buildModernIndex(pre, indexer.prereleaseSpecsIdx, "prerelease specs")
}

func buildModernIndex(specs []*spec.Spec, idxFile string, name string) {
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

func (indexer Indexer) compressIndicies() {
	gzipFile(indexer.prereleaseSpecsIdx)
	gzipFile(indexer.specsIdx)
	gzipFile(indexer.latestSpecsIdx)
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

func (indexer Indexer) buildIndicies() {
	specs := mapGemsToSpecs(indexer.gemList())
	// indexer.buildMarshalGemspecs(specs)
	indexer.buildModernIndices(specs)
	indexer.compressIndicies()
}

func (indexer Indexer) installIndicies() {
	destName := fmt.Sprintf("%s/%s", indexer.destDir, indexer.quickMarshalDirBase)
	// FileUtils.mkdir_p File.dirname(dst_name), :verbose => verbose
	fmt.Println(destName)
	fmt.Println(filepath.Dir(destName))
	err := os.MkdirAll(filepath.Dir(destName), os.ModePerm)
	check(err)
	// FileUtils.rm_rf dst_name, :verbose => verbose
	err = os.RemoveAll(destName)
	check(err)
	// FileUtils.mv(@quick_marshal_dir, dst_name,
	//              :verbose => verbose, :force => true)
	err = os.Rename(indexer.quickMarshalDir, destName)
	check(err)
	indexer.files = indexer.files[1:]
	reg := regexp.MustCompile(fmt.Sprintf("^%s/?", indexer.dir))
	for _, file := range indexer.files {
		file = reg.ReplaceAllString(file, "${1}")
		srcName := fmt.Sprintf("%s/%s", indexer.dir, file)
		destName = fmt.Sprintf("%s/%s", indexer.destDir, file)
		err = os.RemoveAll(destName)
		check(err)
		err = os.Rename(srcName, destName)
		check(err)
	}
}
