package index

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

const categoryFile = "category.json"
const videosFolder = "videos"

func parseCollection(p string) (*Collection, error) {
	result := Collection{}
	categoryPath := filepath.Join(p, categoryFile)
	videosPath := filepath.Join(p, videosFolder)
	fp, err := os.Open(categoryPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open category.json of %s", p)
	}
	defer fp.Close()
	if err := json.NewDecoder(fp).Decode(&result); err != nil {
		return nil, errors.Wrapf(err, "Failed to decode %s", categoryPath)
	}

	result.Slug = filepath.Base(p)

	dir, err := os.Open(videosPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open videos folder %s", videosPath)
	}
	defer dir.Close()
	videoFiles, err := dir.Readdir(0)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read videos folder %s", videosPath)
	}
	result.Sessions = make([]*Session, 0, len(videoFiles))
	for _, videoFile := range videoFiles {
		videoPath := filepath.Join(videosPath, videoFile.Name())
		if !strings.HasSuffix(videoPath, ".json") {
			continue
		}
		session, err := parseSession(videoPath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to parse sesion file %s", videoPath)
		}
		result.Sessions = append(result.Sessions, session)
	}

	return &result, nil
}

func parseSession(p string) (*Session, error) {
	result := Session{}
	fp, err := os.Open(p)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open session file %s", p)
	}
	defer fp.Close()
	if err := json.NewDecoder(fp).Decode(&result); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse session file %s", p)
	}
	result.Slug = strings.TrimSuffix(filepath.Base(p), ".json")
	return &result, nil
}

// LoadIndex attempts to load an index from a given path or build it based
// on the data folder. If the index already exists then you can enforce a
// rebuild using the forceRebuild parameter.
func LoadIndex(indexPath string, dataFolder string, forceRebuild bool) (bleve.Index, error) {
	sessionIndexMapping := bleve.NewDocumentMapping()
	sessionIndexMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	sessionIndexMapping.AddFieldMappingsAt("description", bleve.NewTextFieldMapping())

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("session", sessionIndexMapping)

	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			log.Infof("%s doesn't exist yet. Creating a new index there.", indexPath)
			idx, err := bleve.New(indexPath, mapping)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to create new index in %s", indexPath)
			}
			if err := fillIndex(idx, dataFolder); err != nil {
				return nil, errors.Wrapf(err, "Failed to build index at %s", indexPath)
			}
			return idx, err
		}
		return nil, errors.Wrapf(err, "Failed to create new index at %s", indexPath)
	}
	// The index already exists, let's open it from here.
	if forceRebuild {
		if err := os.RemoveAll(indexPath); err != nil {
			return nil, errors.Wrapf(err, "Failed to remove old index folder %s", indexPath)
		}
		idx, err := bleve.New(indexPath, mapping)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create new index in %s", indexPath)
		}
		if err := fillIndex(idx, dataFolder); err != nil {
			return nil, errors.Wrapf(err, "Failed to build index at %s", indexPath)
		}
		return idx, err
	}
	log.Infof("%s already exists. Loading index from there.", indexPath)
	return bleve.Open(indexPath)
}

func fillIndex(idx bleve.Index, dataFolder string) error {
	root, err := os.Open(dataFolder)
	if err != nil {
		return errors.Wrapf(err, "Failed to open pyvideo data folder %s", dataFolder)
	}
	defer root.Close()

	categoryFolders, err := root.Readdir(0)
	if err != nil {
		return errors.Wrap(err, "Failed to read root category folders")
	}

	var folderWait sync.WaitGroup
	var processingWait sync.WaitGroup
	folderWait.Add(len(categoryFolders))

	parsedCollections := make(chan *Collection, len(categoryFolders))

	for _, folder := range categoryFolders {
		absPath := filepath.Join(dataFolder, folder.Name())
		categoryPath := filepath.Join(absPath, categoryFile)
		if strings.HasPrefix(folder.Name(), ".") {
			folderWait.Done()
			continue
		}
		if _, err := os.Stat(categoryPath); err != nil {
			folderWait.Done()
			continue
		}
		go func(p string) {
			collection, err := parseCollection(p)
			if err != nil {
				log.WithError(err).Fatalf("Failed to load the collection data in %s", p)
			}
			parsedCollections <- collection
			folderWait.Done()
		}(absPath)
	}

	processingWait.Add(1)

	go func() {
		var indexers sync.WaitGroup
		indexers.Add(5)
		for i := 0; i < 5; i++ {
			go func() {
				for collection := range parsedCollections {
					log.Infof("Indexing %s", collection.Title)
					batch := idx.NewBatch()
					for _, session := range collection.Sessions {
						id := fmt.Sprintf("session:%s:%s", collection.Title, session.Title)
						batch.Index(id, newIndexedSession(session, collection))
					}
					idx.Batch(batch)
				}
				indexers.Done()
			}()
		}
		indexers.Wait()
		processingWait.Done()
	}()

	folderWait.Wait()
	close(parsedCollections)
	processingWait.Wait()
	return nil
}
