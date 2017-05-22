package main

import (
	"net/http"

	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"encoding/json"

	"github.com/blevesearch/bleve"
	"github.com/julienschmidt/httprouter"
)

func runHTTPD(idx bleve.Index, addr string, allowedOrigins []string) error {
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

	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})

	log.Printf("Starting server on %s (allowing XHR from %s)", addr, allowedOrigins)
	return http.ListenAndServe(addr, c.Handler(router))
}
