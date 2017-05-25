FROM alpine:3.5

MAINTAINER Horst Gutmann <zerok@zerokspot.com>

RUN apk add --no-cache git
ADD pyvideosearch-linux /
VOLUME ["/var/lib/pyvideosearch"]

EXPOSE 8000
CMD ["--data-path", "/var/lib/pyvideosearch/data", "--index-path", "/var/lib/pyvideosearch/index", "--http", "--http-addr", "0.0.0.0:8000", "--allowed-origin", "http://pyvideo.org", "--allowed-origin", "http://www.pyvideo.org", "--allowed-origin", "http://localhost:8000", "--allowed-origin", "https://pyvideo.org", "--allowed-origin", "https://www.pyvideo.org", "--check-interval", "30s"]
ENTRYPOINT ["/pyvideosearch-linux"]