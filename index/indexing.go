package index

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"

	"io/ioutil"

	"github.com/Flaque/filet"
	log "github.com/sirupsen/logrus"
)

const categoryFile = "category.json"
const videosFolder = "videos"
const stateFile = ".state"

func WatchForUpdates(ctx context.Context, idxChan chan bleve.Index, indexPath string, dataPath string, interval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		log.Info("Checking upstream for new commits")

		if err := updateRepo(dataPath); err != nil {
			return errors.Wrapf(err, "Failed to update git repository at %s", dataPath)
		}

		ref, err := getRepoState(dataPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to get data repo state of %s", dataPath)
		}

		idxRef, err := getIndexState(indexPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to get index state of %s", indexPath)
		}

		if idxRef != ref {
			log.Info("New commits found. Will rebuild index")
			tmpIndex := filet.TmpDir(nil, "")
			idx, err := LoadIndex(ctx, tmpIndex, dataPath, true)
			if err != nil {
				return errors.Wrap(err, "Failed to load the new index")
			}
			idx.Close()
			os.RemoveAll(indexPath)
			os.Rename(tmpIndex, indexPath)
			newIdx, err := bleve.Open(indexPath)
			if err != nil {
				return errors.Wrap(err, "Failed to open new index")
			}
			idxChan <- newIdx
		}

		time.Sleep(interval)
	}
}

// LoadIndex attempts to load an index from a given path or build it based
// on the data folder. If the index already exists then you can enforce a
// rebuild using the forceRebuild parameter.
func LoadIndex(ctx context.Context, indexPath string, dataFolder string, forceRebuild bool) (bleve.Index, error) {
	log.Info("Loading index")
	defer log.Info("Load complete")
	var create bool
	sessionIndexMapping := bleve.NewDocumentMapping()
	sessionIndexMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	sessionIndexMapping.AddFieldMappingsAt("description", bleve.NewTextFieldMapping())

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("session", sessionIndexMapping)

	if _, err := os.Stat(indexPath); err != nil {
		if os.IsNotExist(err) {
			create = true
			log.Infof("%s doesn't exist yet. Creating a new index there.", indexPath)
		} else {
			return nil, errors.Wrapf(err, "Failed to create new index at %s", indexPath)
		}
	}
	if forceRebuild || create {
		if err := os.RemoveAll(indexPath); err != nil {
			return nil, errors.Wrapf(err, "Failed to remove old index folder %s", indexPath)
		}
		idx, err := bleve.New(indexPath, mapping)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create new index in %s", indexPath)
		}
		if err := fillIndex(ctx, idx, dataFolder); err != nil {
			return nil, errors.Wrapf(err, "Failed to build index at %s", indexPath)
		}
		ref, err := getRepoState(dataFolder)
		if err != nil {
			return idx, err
		}
		if err := setIndexState(indexPath, ref); err != nil {
			return idx, err
		}
		return idx, err
	}
	log.Infof("%s already exists. Loading index from there.", indexPath)
	return bleve.Open(indexPath)
}

func parseCollection(ctx context.Context, p string) (*Collection, error) {
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
		select {
		case <-ctx.Done():
			return nil, errors.New("Canceled")
		default:
		}
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

func fillIndex(ctx context.Context, idx bleve.Index, dataFolder string) error {
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
	errs := make(chan error, len(categoryFolders))
	folderWait.Add(len(categoryFolders))

	parsedCollections := make(chan *Collection, len(categoryFolders))

	subContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, folder := range categoryFolders {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
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
			defer folderWait.Done()
			select {
			case <-subContext.Done():
				return
			default:
			}
			collection, err := parseCollection(subContext, p)
			if err != nil {
				cancel()
				errs <- errors.Wrapf(err, "Failed to load the collection data in %s", p)
				return
			}
			parsedCollections <- collection
		}(absPath)
	}

	processingWait.Add(1)

	go func() {
		var indexers sync.WaitGroup
		indexers.Add(5)
		for i := 0; i < 5; i++ {
			go func() {
				defer indexers.Done()
				for {
					select {
					case <-subContext.Done():
						return
					case collection, ok := <-parsedCollections:
						if !ok {
							return
						}
						log.Infof("Indexing %s", collection.Title)
						batch := idx.NewBatch()
						for _, session := range collection.Sessions {
							id := fmt.Sprintf("session:%s:%s", collection.Slug, session.Slug)
							batch.Index(id, newIndexedSession(session, collection))
						}
						idx.Batch(batch)
					}
				}
			}()
		}
		indexers.Wait()
		processingWait.Done()
	}()

	folderWait.Wait()
	close(parsedCollections)
	processingWait.Wait()

	// Drain the errors channel to make sure that we are not ignoring an error.
	select {
	case err := <-errs:
		return err
	default:
	}
	return nil
}

func updateRepo(p string) error {
	cmd := exec.Command("git", "pull", "origin", "master")
	cmd.Dir = p
	return cmd.Run()
}

func getRepoState(p string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = p
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), err
}

func getIndexState(p string) (string, error) {
	sp := filepath.Join(p, stateFile)
	data, err := ioutil.ReadFile(sp)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), err
}

func setIndexState(p string, ref string) error {
	sp := filepath.Join(p, stateFile)
	return ioutil.WriteFile(sp, []byte(ref), 0600)
}
