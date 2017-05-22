# PyVideo Search

This is a the backend component for the search functionality available on
pyvideo.org. It uses [bleve][] in the background and for now supports only
fulltext-searching.

## Usage

```
# You need to have Go installed for that:
$ go get github.com/zerok/pyvideosearch

$ ./pyvideosearch --data-path /path/to/pyvideo-data \
  --index-path /path/to/search.bleve \
  --http-addr 0.0.0.0:8080
```

This will index the data in the data-path folder and create a search index
in the index-path if that folder doesn't exist yet. Afterwards, a HTTP server
is started listening on 0.0.0.0:8080. You can then query the index with the
`/api/v1/search?q=<your search>` endpoint.

By default, pyvideosearch only allows XHRs from `http://localhost:8000`. To
change that, use the `--allowed-origin` flag (you can pass that multiple times
to set multiple allowed origins).


## How to build

You need to have Go installed in order to build this project:

```
$ mkdir -p $GOPATH/src/github.com/zerok
$ cd $GOPATH/src/github.com/zerok
$ git clone git@github.com:zerok/pyvideosearch.git
$ cd pyvideosearch
$ go build
```

[bleve]: http://www.blevesearch.com/