VERSION="1.0"

.PHONY: build
build:
	go build -o bin/api cmd/main.go

.PHONY: run
run: build
	./bin/api

.PHONY: image
image:
	docker build . -t waf-api:$(VERSION)
