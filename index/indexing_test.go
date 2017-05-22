package index

import (
	"fmt"
	"path/filepath"
	"testing"

	"io/ioutil"
	"os"

	"github.com/Flaque/filet"
	"github.com/blevesearch/bleve"
)

var sessionContent = `{
	"title": "Some title",
	"speakers": []
}`

var categoryContent = `{
	"title": "My Conference"
}`

func TestFillIndex(t *testing.T) {
	defer filet.CleanUp(t)
	root, _ := createConference(t, "conf-2017", []string{"my-session", "my-other-session"})
	idx, _ := bleve.NewMemOnly(bleve.NewIndexMapping())

	if err := fillIndex(idx, root); err != nil {
		t.Fatalf("Unexpected error when filling the index: %s", err.Error())
	}

	count, _ := idx.DocCount()
	if count != 2 {
		t.Fatalf("Expected 2 document in the index. Got %d.", count)
	}
}

// TestFillIndexBorkenCategoryJSON checks the behaviour of the fillIndex
// function if the category.json file is not actually JSON. In that case
// the program shouldn't panic but just return an error.
func TestFillIndexBrokenCategoryJSON(t *testing.T) {
	defer filet.CleanUp(t)
	root, confFolder := createConference(t, "conf-2017", []string{})
	ioutil.WriteFile(filepath.Join(confFolder, "category.json"), []byte("not valid json"), 0600)

	idx, _ := bleve.NewMemOnly(bleve.NewIndexMapping())

	if err := fillIndex(idx, root); err == nil {
		t.Fatal("Expected error not returned")
	}
}

func TestParseSession(t *testing.T) {
	defer filet.CleanUp(t)
	_, confPath := createConference(t, "conf-2017", []string{"my-session"})

	s, err := parseSession(getVideoPath(confPath, "my-session"))
	if err != nil {
		t.Fatalf("Parsing session file returned an unexpected error: %s", err.Error())
	}

	// The session slug is pretty much the filename without the extension.
	if s.Slug != "my-session" {
		t.Errorf("Unexpected value for slug: %v", s.Slug)
	}
}

func createConference(t *testing.T, slug string, sessions []string) (string, string) {
	root := filet.TmpDir(t, "")
	confPath := filepath.Join(root, slug)
	videos := filepath.Join(confPath, videosFolder)
	catPath := filepath.Join(confPath, categoryFile)
	os.MkdirAll(videos, 0755)

	for _, session := range sessions {
		videoPath := filepath.Join(videos, fmt.Sprintf("%s.json", session))
		ioutil.WriteFile(videoPath, []byte(sessionContent), 0600)
	}

	ioutil.WriteFile(catPath, []byte(categoryContent), 0600)
	return root, confPath
}

func getVideoPath(root string, slug string) string {
	return filepath.Join(root, "videos", fmt.Sprintf("%s.json", slug))
}
