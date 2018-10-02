VERSION := 0.1.0-dev

.PHONY: build

build:
	docker build -t moov/api.moov.io:$(VERSION) -f Dockerfile .
	docker tag moov/api.moov.io:$(VERSION) moov/api.moov.io:latest

release-push:
	docker push moov/api.moov.io:$(VERSION)
