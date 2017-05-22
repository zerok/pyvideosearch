package main

import "fmt"

type Video struct {
	Type string
	URL  string
}

type Session struct {
	Title       string
	Description string
	Speakers    []string
	Recorded    string
	Videos      []Video
	Slug        string
}

type Collection struct {
	Title    string `json:"title"`
	Slug     string
	Sessions []*Session
}

type IndexedSession struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Collection  string   `json:"collection"`
	Speakers    []string `json:"speakers"`
}

func (s IndexedSession) Type() string {
	return "session"
}

func newIndexedSession(session *Session, collection *Collection) IndexedSession {
	res := IndexedSession{
		Title:       session.Title,
		Description: session.Description,
		Speakers:    session.Speakers,
		URL:         fmt.Sprintf("/%s/%s.html", collection.Slug, session.Slug),
		Collection:  collection.Title,
	}
	return res
}
