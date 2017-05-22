package main

import "testing"

func TestNewIndexedSession(t *testing.T) {
	col := &Collection{
		Title: "Conference",
		Slug:  "conf",
	}
	ses := &Session{
		Title: "My Session",
		Slug:  "my-session",
	}
	res := newIndexedSession(ses, col)
	if res.Title != ses.Title {
		t.Error("Title wasn't copied over from the session")
	}
	if res.URL != "/conf/my-session.html" {
		t.Errorf("Unexpected URL: %s", res.URL)
	}
}
