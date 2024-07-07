all: pyvideosearch

pyvideosearch: $(shell find . -name '*.go')
	cd cmd/pyvideosearch && go build -o ../../pyvideosearch

test:
	go test -v $(shell go list ./... | grep -v /vendor/)

clean:
	rm -rf pyvideosearch pyvideosearch-linux dist

.PHONY: all clean
