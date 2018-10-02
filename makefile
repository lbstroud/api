VERSION := v0.1.0-dev2018100201

.PHONY: build

build:
	docker build -t moov/api.moov.io:$(VERSION) -f Dockerfile .
	docker tag moov/api.moov.io:$(VERSION) moov/api.moov.io:latest

release-push:
	docker push moov/api.moov.io:$(VERSION)
