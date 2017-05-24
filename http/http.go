package http

import (
	"context"
	"net/http"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"encoding/json"

	"expvar"

	"sync"

	"github.com/blevesearch/bleve"
	"github.com/julienschmidt/httprouter"
)

var searchQueries = expvar.NewInt("pyvideo.search_count")

// RunHTTPD starts the API server on the given addr serving the index.
// If you need to support XHRs, make sure to pass respective allowedOrigin
// hosts like http://domain.com:5000.
func RunHTTPD(ctx context.Context, idxChan chan bleve.Index, addr string, allowedOrigins []string) error {
	router := httprouter.New()

	idxLock := sync.RWMutex{}
	idx, _ := bleve.NewMemOnly(bleve.NewIndexMapping())

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case i := <-idxChan:
				idxLock.Lock()
				idx.Close()
				idx = i
				idxLock.Unlock()
				log.Info("Index updated for HTTPD")
			}
		}
	}()

	router.Handler(http.MethodGet, "/api/v1/metrics", expvar.Handler())

	router.GET("/api/v1/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		searchQueries.Add(1)
		qs := r.FormValue("q")
		q := bleve.NewQueryStringQuery(qs)
		req := bleve.NewSearchRequest(q)
		req.Fields = []string{"title", "url", "conference", "speakers.name", "speakers.slug", "thumbnail_url", "collection_title", "collection_url", "recorded", "recorded_formatted"}
		req.Size = 100
		req.IncludeLocations = true
		collectionFacet := bleve.NewFacetRequest("collection_title", 10)
		speakerFacet := bleve.NewFacetRequest("speakers", 10)
		req.AddFacet("speaker", speakerFacet)
		req.AddFacet("collection", collectionFacet)
		idxLock.RLock()
		defer idxLock.RUnlock()
		res, err := idx.Search(req)
		if err != nil {
			http.Error(w, "Query failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})

	log.Printf("Starting server on %s (allowing XHR from %s)", addr, allowedOrigins)
	return http.ListenAndServe(addr, c.Handler(router))
}
