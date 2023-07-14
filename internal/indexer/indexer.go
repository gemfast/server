package indexer

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/marshal"
	"github.com/gemfast/server/internal/spec"
	"golang.org/x/exp/slices"

	"github.com/rs/zerolog/log"
)

var lock sync.Mutex

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
	cfg                    *config.Config
	db                     *db.DB
}

const (
	EPOCH         = "1969-12-31T19:00:00-05:00"
	RUBY_PLATFORM = "ruby"
)

func NewIndexer(cfg *config.Config, db *db.DB) (*Indexer, error) {
	gemfastDir := cfg.Dir
	marshalName := "Marshal.4.8"
	indexer := &Indexer{destDir: gemfastDir, cfg: cfg, db: db}
	tmpdir, err := mkTempDir("gem_generate_index")
	if err != nil {
		return nil, err
	}
	indexer.dir = tmpdir
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
	indexer.files = append(indexer.files, indexer.specsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.specsIdx))
	indexer.files = append(indexer.files, indexer.latestSpecsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.latestSpecsIdx))
	indexer.files = append(indexer.files, indexer.prereleaseSpecsIdx)
	indexer.files = append(indexer.files, fmt.Sprintf("%s.gz", indexer.prereleaseSpecsIdx))
	log.Info().Str("detail", indexer.dir).Msg("indexer initialized")
	shouldIndex := false
	if _, err := os.Stat(indexer.destSpecsIdx); errors.Is(err, os.ErrNotExist) {
		log.Info().Str("detail", indexer.destSpecsIdx).Msg("index file not found - generating the index")
		shouldIndex = true
	}
	if _, err := os.Stat(indexer.destLatestSpecsIdx); errors.Is(err, os.ErrNotExist) {
		log.Info().Str("detail", indexer.destLatestSpecsIdx).Msg("index file not found - generating the index")
		shouldIndex = true
	}
	if _, err := os.Stat(indexer.destPrereleaseSpecsIdx); errors.Is(err, os.ErrNotExist) {
		log.Info().Str("detail", indexer.destPrereleaseSpecsIdx).Msg("index file not found - generating the index")
		shouldIndex = true
	}
	if shouldIndex {
		indexer.GenerateIndex()
	}
	return indexer, nil
}

func (indexer *Indexer) GenerateIndex() error {
	mkDirs(indexer.quickMarshalDir)
	mkDirs(indexer.cfg.GemDir)
	mkDirs(indexer.cfg.DBDir)
	_, specsMissing := os.Stat(fmt.Sprintf("%s/specs.4.8.gz", indexer.cfg.Dir))
	_, prereleaseSpecsMissing := os.Stat(fmt.Sprintf("%s/prerelease_specs.4.8.gz", indexer.cfg.Dir))
	_, latestSpecsMissing := os.Stat(fmt.Sprintf("%s/latest_specs.4.8.gz", indexer.cfg.Dir))
	if specsMissing != nil || prereleaseSpecsMissing != nil || latestSpecsMissing != nil {
		indexer.buildIndicies()
		indexer.installIndicies()
	}
	return nil
}

func mkTempDir(name string) (string, error) {
	dir, err := os.MkdirTemp("/tmp", name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmpdir")
		return dir, err
	}
	log.Trace().Msg(fmt.Sprintf("created tmpdir %s", dir))
	err = os.Chmod(dir, 0700)
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmpdir")
	}
	return dir, err
}

func mkDirs(dir string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create dir %s: %w", dir, err)
	}
	return nil
}

func (indexer *Indexer) gemList() ([]string, error) {
	var gems []string
	gemDir := indexer.cfg.GemDir
	err := filepath.WalkDir(gemDir, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ".gem" && filepath.Base(s) != ".gem" {
			gems = append(gems, s)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk gem dir %s: %w", gemDir, err)
	}
	return gems, nil
}

func mapGemsToSpecs(gems []string) ([]*spec.Spec, error) {
	var specs []*spec.Spec
	var s *spec.Spec
	for _, g := range gems {
		fi, err := os.Stat(g)
		if err != nil {
			log.Error().Err(err).Str("detail", g).Msg("failed to stat gem")
			return nil, err
		}
		if fi.Size() == 0 {
			log.Info().Str("detail", g).Msg("skipping zero-length gem")
			continue
		} else {
			log.Trace().Str("detail", g).Msg("extracting spec from gem")
			s, err = spec.FromFile(g)
			if err != nil {
				log.Error().Err(err).Str("detail", g).Msg("failed to extract spec from gem")
				return nil, err
			}
			specs = append(specs, s)
		}
	}
	return specs, nil
}

func (indexer *Indexer) buildMarshalGemspecs(specs []*spec.Spec, update bool) error {
	for _, s := range specs {
		specFName := fmt.Sprintf("%s.gemspec.rz", s.OriginalName)
		var marshalName string
		if update {
			marshalName = fmt.Sprintf("%s/%s/%s", indexer.destDir, indexer.quickMarshalDirBase, specFName)
			if _, err := os.Stat(marshalName); err == nil {
				log.Trace().Str("detail", marshalName).Msg("skipping marshal gemspec dump")
				continue
			}
		} else {
			marshalName = fmt.Sprintf("%s/%s", indexer.quickMarshalDir, specFName)
		}

		dump := marshal.DumpGemspecGemfast(s.GemMetadata)
		var b bytes.Buffer
		rz := zlib.NewWriter(&b)
		defer rz.Close()
		if _, err := rz.Write(dump); err != nil {
			return fmt.Errorf("failed to write zlib dump: %w", err)
		}
		err := rz.Close()
		if err != nil {
			return fmt.Errorf("failed to close zlib writer: %w", err)
		}
		err = os.WriteFile(marshalName, b.Bytes(), 0666)
		if err != nil {
			return fmt.Errorf("failed to write marshal file: %w", err)
		}
	}
	return nil
}

func buildModernIndex(specs []*spec.Spec, idxFile string, name string) error {
	file, err := os.OpenFile(
		idxFile,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", idxFile, err)
	}
	defer file.Close()

	dump := marshal.DumpSpecs(specs)
	_, err = file.Write(dump)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", idxFile, err)
	}
	return nil
}

func (indexer *Indexer) compressIndicies() error {
	tmpIndicies := []string{indexer.prereleaseSpecsIdx, indexer.latestSpecsIdx, indexer.specsIdx}
	for _, index := range tmpIndicies {
		if _, err := os.Stat(index); err == nil {
			err = gzipFile(index)
			if err != nil {
				return fmt.Errorf("failed to gzip file %s: %w", index, err)
			}
		}
	}
	return nil
}

func gzipFile(src string) error {
	content, err := os.ReadFile(src) // just pass the file name
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", src, err)
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	defer gz.Close() //NOT SUFFICIENT, DON'T DEFER WRITER OBJECTS
	if _, err := gz.Write(content); err != nil {
		return fmt.Errorf("failed to write gzip file %s: %w", src, err)
	}
	// NEED TO CLOSE EXPLICITLY
	err = gz.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	err = os.WriteFile(fmt.Sprintf("%s.gz", src), b.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("failed to write gzip file %s: %w", src, err)
	}
	return nil
}

func (indexer *Indexer) buildIndicies() error {
	gl, err := indexer.gemList()
	if err != nil {
		return fmt.Errorf("failed to get gem list: %w", err)
	}
	specs, err := mapGemsToSpecs(gl)
	if err != nil {
		log.Error().Err(err).Msg("failed to map gems to specs")
		return err
	}
	indexer.buildMarshalGemspecs(specs, false)
	pre, rel, latest := spec.PartitionSpecs(specs)
	buildModernIndex(rel, indexer.specsIdx, "specs")
	buildModernIndex(latest, indexer.latestSpecsIdx, "latest specs")
	buildModernIndex(pre, indexer.prereleaseSpecsIdx, "prerelease specs")
	indexer.compressIndicies()
	return nil
}

func (indexer *Indexer) installIndicies() error {
	destName := fmt.Sprintf("%s/%s", indexer.destDir, indexer.quickMarshalDirBase)
	err := os.MkdirAll(filepath.Dir(destName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destName, err)
	}
	err = os.RemoveAll(destName)
	if err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", destName, err)
	}
	err = os.Rename(indexer.quickMarshalDir, destName)
	if err != nil {
		return fmt.Errorf("failed to rename directory %s to %s: %w", indexer.quickMarshalDir, destName, err)
	}
	reg := regexp.MustCompile(fmt.Sprintf("^%s/?", indexer.dir))
	for _, file := range indexer.files {
		file = reg.ReplaceAllString(file, "${1}")
		srcName := fmt.Sprintf("%s/%s", indexer.dir, file)
		if _, err := os.Stat(srcName); errors.Is(err, os.ErrNotExist) {
			continue
		}
		destName = fmt.Sprintf("%s/%s", indexer.destDir, file)
		err = os.RemoveAll(destName)
		if err != nil {
			return fmt.Errorf("failed to remove file %s: %w", destName, err)
		}
		err = os.Rename(srcName, destName)
		if err != nil {
			return fmt.Errorf("failed to rename file %s to %s: %w", srcName, destName, err)
		}
	}
	err = os.RemoveAll(indexer.dir)
	if err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", indexer.dir, err)
	}
	return nil
}

func (indexer *Indexer) updateSpecsIndex(updated []*spec.Spec, src string, dest string, ch chan<- int) {
	if len(updated) == 0 {
		log.Info().Str("detail", src).Msg("no new gems for index")
		ch <- 0
		return
	}
	var specsIdx []*spec.Spec
	file, err := os.Open(src)
	if err != nil {
		log.Error().Err(err).Str("detail", src).Msg("failed to open index")
		ch <- 0
		return
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	out, err := io.ReadAll(fileReader)
	if err != nil {
		log.Error().Err(err).Str("detail", src).Msg("failed to read index")
		ch <- 0
		return
	}
	buff := bytes.NewBuffer(out)

	specsIdx = marshal.LoadSpecs(buff)
	log.Debug().Str("detail", src).Int("len", len(specsIdx)).Msg("loaded index")
	for _, spec := range updated {
		platform := spec.OriginalPlatform
		if platform == "" {
			spec.OriginalPlatform = RUBY_PLATFORM
		}
		specsIdx = append(specsIdx, spec)
	}

	var uniqSpecsIdx []*spec.Spec
	if src == indexer.destLatestSpecsIdx {
		specMap := make(map[string]*spec.Spec)
		for _, spec := range specsIdx {
			if specMap[spec.Name] != nil {
				if specMap[spec.Name].Version < spec.Version {
					specMap[spec.Name] = spec
				}
			} else {
				specMap[spec.Name] = spec
			}
		}
		for _, v := range specMap {
			uniqSpecsIdx = append(uniqSpecsIdx, v)
		}
	} else {
		shaMap := make(map[string]int)
		for _, spec := range specsIdx {
			sha1 := sha1.New()
			specId := spec.Name + spec.Version + spec.OriginalPlatform
			sha1.Write([]byte(specId))
			sha1Str := string(sha1.Sum(nil))
			shaMap[sha1Str] += 1
			if shaMap[sha1Str] == 1 {
				uniqSpecsIdx = append(uniqSpecsIdx, spec)
			}
		}
	}

	sort.Slice(uniqSpecsIdx, func(i, j int) bool {
		l := uniqSpecsIdx[i].Name + uniqSpecsIdx[i].Version + uniqSpecsIdx[i].OriginalPlatform
		r := uniqSpecsIdx[j].Name + uniqSpecsIdx[j].Version + uniqSpecsIdx[j].OriginalPlatform
		return l < r
	})

	file, err = os.OpenFile(
		dest,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		log.Error().Err(err).Str("detail", dest).Msg("failed to create destination spec index file")
		panic(err)
	}
	defer file.Close()

	dump := marshal.DumpSpecs(uniqSpecsIdx)
	bytesWritten, err := file.Write(dump)
	if err != nil {
		log.Error().Err(err).Str("detail", dest).Msg("failed to write destination spec index file")
		panic(err)
	}
	log.Info().Str("detail", src).Int("len", len(uniqSpecsIdx)).Msg("updated index")
	ch <- bytesWritten
}

// TODO: refactor this to not reopen the gemspec files that have been uploaded
func (indexer *Indexer) UpdateIndex(updatedGems []string) error {
	lock.Lock()
	defer lock.Unlock()
	defer os.RemoveAll(indexer.dir)
	mkDirs(indexer.quickMarshalDir)

	specs, err := mapGemsToSpecs(updatedGems)
	if err != nil {
		log.Error().Err(err).Msg("failed to update index - unable to map gems to specs")
		return err
	}

	err = indexer.db.SaveGemVersions(specs)
	if err != nil {
		log.Error().Err(err).Msg("failed to update index - unable to save gem dependencies")
		return err
	}

	indexer.buildMarshalGemspecs(specs, true)

	pre, rel, latest := spec.PartitionSpecs(specs)
	// TODO: capture errors from these goroutines
	ch := make(chan int, 3)
	go indexer.updateSpecsIndex(rel, indexer.destSpecsIdx, indexer.specsIdx, ch)
	go indexer.updateSpecsIndex(latest, indexer.destLatestSpecsIdx, indexer.latestSpecsIdx, ch)
	go indexer.updateSpecsIndex(pre, indexer.destPrereleaseSpecsIdx, indexer.prereleaseSpecsIdx, ch)

	<-ch
	<-ch
	<-ch

	return indexer.compressAndMoveIndices()
}

func (indexer *Indexer) AddGemToIndex(gem string) error {
	return indexer.UpdateIndex([]string{gem})
}

func (indexer *Indexer) Reindex() error {
	var updatedGems []string
	fi, err := os.Stat(indexer.destSpecsIdx)
	if err != nil {
		log.Error().Err(err).Str("detail", indexer.destSpecsIdx).Msg("destination specs index file does not exist")
		return err
	}
	specsMtime := fi.ModTime()
	newestMtime, err := time.Parse(time.RFC3339, EPOCH)
	if err != nil {
		log.Error().Err(err).Msg("unable to parse epoch time")
		return err
	}
	gl, err := indexer.gemList()
	if err != nil {
		log.Error().Err(err).Msg("failed to reindex - unable to list gems")
		return err
	}
	for _, gem := range gl {
		fi, err := os.Stat(gem)
		if err != nil {
			log.Error().Err(err).Msg("")
			return err
		}
		gemMtime := fi.ModTime()
		if gemMtime.Unix() > newestMtime.Unix() {
			newestMtime = gemMtime
		}
		if gemMtime.Unix() > specsMtime.Unix() {
			updatedGems = append(updatedGems, gem)
		}
	}

	if len(updatedGems) == 0 {
		log.Trace().Msg("no new gems")
		return nil
	}
	log.Trace().Str("detail", strings.Join(updatedGems, ",")).Msg("updated gems")

	return indexer.UpdateIndex(updatedGems)
}

func (indexer *Indexer) compressAndMoveIndices() error {
	indexer.compressIndicies()

	reg := regexp.MustCompile(fmt.Sprintf("^%s/?", indexer.dir))
	for _, file := range indexer.files {
		file = reg.ReplaceAllString(file, "${1}")
		srcName := fmt.Sprintf("%s/%s", indexer.dir, file)
		destName := fmt.Sprintf("%s/%s", indexer.destDir, file)
		if _, err := os.Stat(srcName); err == nil {
			err = os.RemoveAll(destName)
			if err != nil {
				log.Error().Err(err).Str("detail", destName).Msg("failed to remove existing file")
				return err
			}
			err = os.Rename(srcName, destName)
			if err != nil {
				log.Error().Err(err).Str("detail", destName).Msg("failed to move file")
				return err
			}
			newestMtime, _ := time.Parse(time.RFC3339, EPOCH)
			err = os.Chtimes(destName, newestMtime, newestMtime)
			if err != nil {
				log.Error().Err(err).Str("detail", destName).Msg("failed to update file mtime")
				return err
			}
		}
	}
	return nil
}

func (indexer *Indexer) RemoveGemFromIndex(name string, version string, platform string) error {
	lock.Lock()
	defer lock.Unlock()
	if platform == "" {
		platform = "ruby"
	}
	gl, err := indexer.gemList()
	if err != nil {
		log.Error().Err(err).Msg("failed to remove gem from index - unable to list gems")
		return err
	}
	specs, err := mapGemsToSpecs(gl)
	if err != nil {
		log.Error().Err(err).Msg("failed to map gems to specs")
		return err
	}
	var toDelete spec.Spec
	toDelete.Name = name
	toDelete.Version = version
	toDelete.OriginalPlatform = platform
	i := spec.FindIndexOf(specs, &toDelete)
	if i == -1 {
		return fmt.Errorf("unable to find gem in specs index")
	}
	specs = slices.Delete(specs, i, i+1)
	pre, rel, latest := spec.PartitionSpecs(specs)
	err = buildModernIndex(rel, indexer.destSpecsIdx, "specs")
	if err != nil {
		log.Error().Err(err).Msg("failed to remove gem from index - unable to build specs index")
		return err
	}
	err = buildModernIndex(latest, indexer.destLatestSpecsIdx, "latest specs")
	if err != nil {
		log.Error().Err(err).Msg("failed to remove gem from index - unable to build latest specs index")
		return err
	}
	err = buildModernIndex(pre, indexer.destPrereleaseSpecsIdx, "prerelease specs")
	if err != nil {
		log.Error().Err(err).Msg("failed to remove gem from index - unable to build prerelease specs index")
		return err
	}
	err = indexer.compressAndMoveIndices()
	if err != nil {
		log.Error().Err(err).Msg("failed to remove gem from index - unable to compress and move indices")
		return err
	}
	return nil
}
