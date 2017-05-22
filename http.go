package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"encoding/json"

	"github.com/blevesearch/bleve"
	"github.com/julienschmidt/httprouter"
)

func runHTTPD(idx bleve.Index, addr string) error {
	router := httprouter.New()

	router.GET("/api/v1/search", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		qs := r.FormValue("q")
		q := bleve.NewQueryStringQuery(qs)
		req := bleve.NewSearchRequest(q)
		req.Fields = []string{"title", "url", "conference", "speakers"}
		req.Size = 100
		req.IncludeLocations = true
		res, err := idx.Search(req)
		if err != nil {
			http.Error(w, "Query failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(res)
	})

	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, router)
}
