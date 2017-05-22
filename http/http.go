package http

import (
	"net/http"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"encoding/json"

	"github.com/blevesearch/bleve"
	"github.com/julienschmidt/httprouter"
)

// RunHTTPD starts the API server on the given addr serving the index.
// If you need to support XHRs, make sure to pass respective allowedOrigin
// hosts like http://domain.com:5000.
func RunHTTPD(idx bleve.Index, addr string, allowedOrigins []string) error {
	router := httprouter.New()

	router.GET("/api/v1/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
