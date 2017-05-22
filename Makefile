all: pyvideosearch

pyvideosearch: $(shell find . -name '*.go')
	cd cmd/pyvideosearch && go build -o ../../pyvideosearch

test:
	go test $(shell go list ./... | grep -v /vendor/)

clean:
	rm -f pyvideosearch

.PHONY: all clean