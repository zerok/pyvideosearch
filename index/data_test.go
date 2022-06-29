package index

import (
	"context"
	"testing"
	"time"
)

func TestNewIndexedSession(t *testing.T) {
	col := &Collection{
		Title: "Conference",
		Slug:  "conf",
	}
	ses := &Session{
		Title: "My Session",
		Slug:  "my-session",
	}
	res := newIndexedSession(context.Background(), ses, col)
	if res.Title != ses.Title {
		t.Error("Title wasn't copied over from the session")
	}
	if res.URL != "/conf/my-session.html" {
		t.Errorf("Unexpected URL: %s", res.URL)
	}
}

func TestRecordedFormats(t *testing.T) {
	europeVienna, _ := time.LoadLocation("Europe/Vienna")
	testcases := []struct {
		Message  string
		Datetime string
		Expected time.Time
	}{
		{
			Message:  "RFC3339 date-only",
			Datetime: "2016-02-05",
			Expected: time.Date(2016, 2, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			Message:  "RFC3339 datetime",
			Datetime: "2016-02-05T18:30:00",
			Expected: time.Date(2016, 2, 5, 18, 30, 0, 0, time.UTC),
		},
		{
			Message:  "RFC3339 datetime + timezone",
			Datetime: "2016-02-05T18:30:00+01:00",
			Expected: time.Date(2016, 2, 5, 18, 30, 0, 0, europeVienna),
		},
	}
	col := &Collection{
		Title: "Conference",
		Slug:  "conf",
	}

	for _, testcase := range testcases {
		session := &Session{
			Recorded: testcase.Datetime,
		}
		result := newIndexedSession(context.Background(), session, col)
		if !result.Recorded.Equal(testcase.Expected) {
			t.Errorf("%s: Expected: %s got %s", testcase.Message, testcase.Expected, result.Recorded)
		}
	}
}
