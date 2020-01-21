package index

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zerok/pyvideosearch/slugify"
)

var inputTimestampFormats = []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}

const outputTimestampFormat = "Mon Jan 2 2006"

type State struct {
	Ref   string
	Index string
}

type Video struct {
	Type string
	URL  string
}

type Session struct {
	Title        string
	Description  string
	Speakers     []string
	Recorded     string
	Videos       []Video
	Slug         string
	ThumbnailURL string `json:"thumbnail_url"`
}

type Speaker struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Collection struct {
	Title    string `json:"title"`
	Slug     string
	Sessions []Session
}

type IndexedSession struct {
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	URL               string    `json:"url"`
	CollectionTitle   string    `json:"collection_title"`
	CollectionURL     string    `json:"collection_url"`
	Speakers          []Speaker `json:"speakers"`
	ThumbnailURL      string    `json:"thumbnail_url"`
	Recorded          time.Time `json:"recorded"`
	RecordedFormatted string    `json:"recorded_formatted"`
}

func (s IndexedSession) Type() string {
	return "session"
}

func newIndexedSession(session Session, collection Collection) IndexedSession {
	speakers := make([]Speaker, 0, len(session.Speakers))
	for _, speaker := range session.Speakers {
		s := Speaker{
			Name: speaker,
			Slug: slugify.Slugify(speaker),
		}
		speakers = append(speakers, s)
	}

	res := IndexedSession{
		Title:           session.Title,
		Description:     session.Description,
		Speakers:        speakers,
		URL:             fmt.Sprintf("/%s/%s.html", collection.Slug, session.Slug),
		CollectionTitle: collection.Title,
		CollectionURL:   fmt.Sprintf("/events/%s.html", collection.Slug),
		ThumbnailURL:    session.ThumbnailURL,
	}

	if session.Recorded != "" {
		valid := false
		for _, fmt := range inputTimestampFormats {
			recorded, err := time.Parse(fmt, session.Recorded)
			if err == nil {
				res.Recorded = recorded
				valid = true
				break
			}
		}
		if !valid {
			log.Infof("Failed to parse %s", session.Recorded)
		}
		res.RecordedFormatted = res.Recorded.Format(outputTimestampFormat)
	}

	return res
}
