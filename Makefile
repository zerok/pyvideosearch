all: pyvideosearch

pyvideosearch-linux: $(shell find . -name '*.go')
	cd cmd/pyvideosearch && GOOS=linux GOARCH=amd64 go build -o ../../pyvideosearch-linux

pyvideosearch: $(shell find . -name '*.go')
	cd cmd/pyvideosearch && go build -o ../../pyvideosearch

test:
	go test -v $(shell go list ./... | grep -v /vendor/)

clean:
	rm -rf pyvideosearch pyvideosearch-linux dist

snapshot:
	goreleaser --snapshot --skip-publish

docker: pyvideosearch-linux Dockerfile
	docker build -t pyvideosearch:latest .

.PHONY: all clean docker
