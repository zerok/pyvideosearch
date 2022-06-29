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

	"github.com/blevesearch/bleve/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
	"github.com/zerok/pyvideosearch/slugify"
)

type Index struct {
	Index bleve.Index
	Path  string
}

func (i *Index) Close() error {
	return i.Index.Close()
}

func (i *Index) Destroy() error {
	if i.Path != "" {
		return os.RemoveAll(i.Path)
	}
	return nil
}

const categoryFile = "category.json"
const videosFolder = "videos"
const stateFile = ".state"

func WatchForUpdates(ctx context.Context, idxChan chan *Index, indexPath string, dataPath string, interval time.Duration, deleteOldIndex bool) error {
	logger := zerolog.Ctx(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		logger.Info().Msg("Checking upstream for new commits")

		if err := updateRepo(ctx, dataPath); err != nil {
			return errors.Wrapf(err, "Failed to update git repository at %s", dataPath)
		}

		ref, err := getRepoState(ctx, dataPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to get data repo state of %s", dataPath)
		}

		idxRef, err := getIndexState(ctx, indexPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to get index state of %s", indexPath)
		}

		logger.Info().Str("index", idxRef.Ref).Str("repo", ref).Msg("Comparing states")

		oldIdx, err := findIndex(indexPath)
		if err != nil {
			return errors.Wrapf(err, "Failed to find old index")
		}

		if idxRef.Ref != ref {
			logger.Info().Msg("New commits found. Will rebuild index")
			newIdxName := newIndexName(indexPath)
			idx, err := createNewIndex(ctx, filepath.Join(indexPath, newIdxName), dataPath)
			if err != nil {
				return errors.Wrap(err, "Failed to load the new index")
			}
			if err := setIndexState(ctx, indexPath, &State{Index: newIdxName, Ref: ref}); err != nil {
				return err
			}
			if oldIdx != "" && deleteOldIndex {
				os.RemoveAll(oldIdx)
			}
			idxChan <- idx
		}

		time.Sleep(interval)
	}
}

func readDir(path string) ([]os.FileInfo, error) {
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	return fp.Readdir(0)
}

func findIndex(root string) (string, error) {
	fp, err := os.Open(root)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer fp.Close()

	files, err := fp.Readdir(0)
	if err != nil {
		return "", err
	}
	for _, file := range files {
		if file.IsDir() {
			return filepath.Join(root, file.Name()), nil
		}
	}
	return "", nil
}

func newIndexName(root string) string {
	return uuid.NewV4().String()
}

func createNewIndex(ctx context.Context, indexPath string, dataPath string) (*Index, error) {
	sessionIndexMapping := bleve.NewDocumentMapping()
	sessionIndexMapping.AddFieldMappingsAt("title", bleve.NewTextFieldMapping())
	sessionIndexMapping.AddFieldMappingsAt("description", bleve.NewTextFieldMapping())

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("session", sessionIndexMapping)
	idx, err := bleve.New(indexPath, mapping)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new index in %s", indexPath)
	}
	if err := fillIndex(ctx, idx, dataPath); err != nil {
		return nil, errors.Wrapf(err, "Failed to build index at %s", indexPath)
	}
	return &Index{
		Index: idx,
		Path:  indexPath,
	}, nil
}

// LoadIndex attempts to load an index from a given path or build it based
// on the data folder. If the index already exists then you can enforce a
// rebuild using the forceRebuild parameter.
func LoadIndex(ctx context.Context, indexPath string, dataFolder string, forceRebuild bool, deleteOld bool) (*Index, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Loading index")
	defer logger.Info().Msg("Load complete")
	var create bool

	idxPath, err := findIndex(indexPath)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new index at %s", indexPath)
	}
	if idxPath == "" {
		create = true
		logger.Info().Msgf("%s doesn't exist yet. Creating a new index.", indexPath)
	}

	if forceRebuild || create {
		if idxPath != "" && deleteOld {
			if err := os.RemoveAll(idxPath); err != nil {
				return nil, errors.Wrapf(err, "Failed to remove old index folder %s", indexPath)
			}
		}
		if err := os.MkdirAll(indexPath, 0700); err != nil {
			return nil, errors.Wrapf(err, "Failed to create index root folder in %s", indexPath)
		}
		idxName := newIndexName(indexPath)
		idxPath = filepath.Join(indexPath, idxName)
		idx, err := createNewIndex(ctx, idxPath, dataFolder)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create new index in %s", indexPath)
		}
		ref, err := getRepoState(ctx, dataFolder)
		if err != nil {
			return nil, err
		}
		if err := setIndexState(ctx, indexPath, &State{Index: idxName, Ref: ref}); err != nil {
			return nil, err
		}
		return idx, err
	}
	logger.Info().Msgf("%s already exists. Loading index from there.", idxPath)
	idx, err := bleve.Open(idxPath)
	if err != nil {
		return nil, err
	}
	return &Index{
		Index: idx,
		Path:  idxPath,
	}, err
}

func parseCollection(ctx context.Context, p string) (Collection, error) {
	result := Collection{}
	categoryPath := filepath.Join(p, categoryFile)
	videosPath := filepath.Join(p, videosFolder)
	fp, err := os.Open(categoryPath)
	if err != nil {
		return result, errors.Wrapf(err, "Failed to open category.json of %s", p)
	}
	if err := json.NewDecoder(fp).Decode(&result); err != nil {
		fp.Close()
		return result, errors.Wrapf(err, "Failed to decode %s", categoryPath)
	}
	fp.Close()

	if result.Slug == "" {
		result.Slug = slugify.Slugify(result.Title)
	}

	videoFiles, err := readDir(videosPath)
	if err != nil {
		return result, errors.Wrapf(err, "Failed to read videos folder %s", videosPath)
	}
	result.Sessions = make([]Session, 0, len(videoFiles))
	for _, videoFile := range videoFiles {
		select {
		case <-ctx.Done():
			return result, errors.New("Canceled")
		default:
		}
		videoPath := filepath.Join(videosPath, videoFile.Name())
		if !strings.HasSuffix(videoPath, ".json") {
			continue
		}
		session, err := parseSession(videoPath)
		if err != nil {
			return result, errors.Wrapf(err, "Failed to parse session file %s", videoPath)
		}
		result.Sessions = append(result.Sessions, session)
	}

	return result, nil
}

func parseSession(p string) (Session, error) {
	result := Session{}
	fp, err := os.Open(p)
	if err != nil {
		return result, errors.Wrapf(err, "Failed to open session file %s", p)
	}
	defer fp.Close()
	if err := json.NewDecoder(fp).Decode(&result); err != nil {
		return result, errors.Wrapf(err, "Failed to parse session file %s", p)
	}
	if result.Slug == "" {
		result.Slug = slugify.Slugify(strings.TrimSpace(result.Title))
	}
	return result, nil
}

func runCollectionParser(ctx context.Context, wait *sync.WaitGroup, errs chan error, parsedCollections chan Collection, work <-chan string) {
	logger := zerolog.Ctx(ctx)
	defer wait.Done()
	defer logger.Info().Msg("Parser done")
	for {
		select {
		case <-ctx.Done():
			return
		case w, ok := <-work:
			if !ok {
				return
			}
			coll, err := parseCollection(ctx, w)
			if err != nil {
				errs <- err
				return
			}
			parsedCollections <- coll
		}
	}
}

func runCollectionGenerator(ctx context.Context, wait *sync.WaitGroup, errs chan error, work chan string, categoryFolders []os.FileInfo, dataFolder string) {
	logger := zerolog.Ctx(ctx)
	defer close(work)
	defer wait.Done()
	defer logger.Info().Msg("Generator done")
	for _, folder := range categoryFolders {
		select {
		case <-ctx.Done():
			return
		default:
		}
		absPath := filepath.Join(dataFolder, folder.Name())
		categoryPath := filepath.Join(absPath, categoryFile)
		if strings.HasPrefix(folder.Name(), ".") {
			continue
		}
		if _, err := os.Stat(categoryPath); err != nil {
			continue
		}
		work <- absPath
	}
}

func runIndexer(ctx context.Context, wait *sync.WaitGroup, errs chan error, idx bleve.Index, parsedCollections chan Collection) {
	logger := zerolog.Ctx(ctx)
	defer wait.Done()
	defer logger.Info().Msg("Indexer done")
	for {
		select {
		case <-ctx.Done():
			return
		case collection, ok := <-parsedCollections:
			if !ok {
				return
			}
			logger.Info().Msgf("Indexing %s", collection.Title)
			batch := idx.NewBatch()
			for _, session := range collection.Sessions {
				id := fmt.Sprintf("session:%s:%s", collection.Slug, session.Slug)
				batch.Index(id, newIndexedSession(ctx, &session, &collection))
			}
			idx.Batch(batch)
		}
	}
}

func fillIndex(ctx context.Context, idx bleve.Index, dataFolder string) error {
	logger := zerolog.Ctx(ctx)
	categoryFolders, err := readDir(dataFolder)
	if err != nil {
		return errors.Wrap(err, "Failed to read root category folders")
	}
	cctx, cancel := context.WithCancel(ctx)
	numParsers := 10
	work := make(chan string)
	parsedCollections := make(chan Collection, 10)
	wg := sync.WaitGroup{}
	errWg := sync.WaitGroup{}
	errWg.Add(1)
	errs := make(chan error)
	wg.Add(1) // For the generator
	wg.Add(1) // For the indexer
	go func() {
		defer errWg.Done()
		defer logger.Info().Msg("Error handler done")
		select {
		case <-cctx.Done():
			return
		case e := <-errs:
			err = e
			cancel()
		}
	}()
	// Before anything else, we should start the go-routine that
	// produces work for the collection parsers:
	go runCollectionGenerator(cctx, &wg, errs, work, categoryFolders, dataFolder)

	// First, let's start those routines that parse the category
	// collections:
	wgParsers := sync.WaitGroup{}
	wgParsers.Add(numParsers)
	for i := 0; i < numParsers; i++ {
		go runCollectionParser(cctx, &wgParsers, errs, parsedCollections, work)
	}

	// Finally, let's start another go-routine that indexes the
	// data:
	go runIndexer(cctx, &wg, errs, idx, parsedCollections)
	// Let's wait for all the parsers to be done before closing the collections channel
	wgParsers.Wait()
	close(parsedCollections)
	wg.Wait()
	cancel()
	errWg.Wait()
	return err
}

func updateRepo(ctx context.Context, p string) error {
	cmd := exec.CommandContext(ctx, "git", "pull", "origin", "master")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = p
	return cmd.Run()
}

func getRepoState(ctx context.Context, p string) (string, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msgf("Checking HEAD of %s", p)
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = p
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), err
}

func getIndexState(ctx context.Context, p string) (*State, error) {
	sp := filepath.Join(p, stateFile)
	state := State{}
	fp, err := os.Open(sp)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	if err := json.NewDecoder(fp).Decode(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

func setIndexState(ctx context.Context, p string, state *State) error {
	sp := filepath.Join(p, stateFile)
	fp, err := os.OpenFile(sp, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()
	return json.NewEncoder(fp).Encode(state)
}
