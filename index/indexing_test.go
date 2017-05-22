package index

import (
	"path/filepath"
	"testing"

	"io/ioutil"
	"os"

	"github.com/Flaque/filet"
)

var sessionContent = `{
	"title": "Some title"	
}`

func TestParseSession(t *testing.T) {
	defer filet.CleanUp(t)

	root := filet.TmpDir(t, "")
	videos := filepath.Join(root, "conf-2017", "videos")
	videoPath := filepath.Join(videos, "my-session.json")
	os.MkdirAll(videos, 0755)

	ioutil.WriteFile(videoPath, []byte(sessionContent), 0600)

	s, err := parseSession(videoPath)
	if err != nil {
		t.Fatalf("Parsing session file returned an unexpected error: %s", err.Error())
	}

	// The session slug is pretty much the filename without the extension.
	if s.Slug != "my-session" {
		t.Errorf("Unexpected value for slug: %v", s.Slug)
	}
}
