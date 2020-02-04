PLATFORM=$(shell uname -s | tr '[:upper:]' '[:lower:]')
VERSION=v$(shell date -u +"%Y.%m.%d").1

.PHONY: all build build-api build-apitest build-localdevproxy docker release dist test

all: build

version:
	@go run ./internal/version/ $(VERSION)

build: version build-api build-apitest build-localdevproxy


build-api:
ifneq ($(TRAVIS_OS_NAME),osx)
	docker build --pull -t moov/api:$(VERSION) -f Dockerfile .
	docker tag moov/api:$(VERSION) moov/api:latest
else
	@echo "Skipping Docker builds on TravisCI"
endif

build-apitest:
ifneq ($(TRAVIS_OS_NAME),osx)
	CGO_ENABLED=0 go build -o bin/apitest ./cmd/apitest/
	docker build --pull -t moov/apitest:$(VERSION) -f Dockerfile-apitest ./
	docker tag moov/apitest:$(VERSION) moov/apitest:latest
else
	@echo "Skipping Docker builds on TravisCI"
endif

build-localdevproxy:
ifneq ($(TRAVIS_OS_NAME),osx)
	CGO_ENABLED=0 go build -o bin/localdevproxy ./cmd/localdevproxy/
	docker build --pull -t moov/localdevproxy:$(VERSION) -f Dockerfile-localdevproxy ./
	docker tag moov/localdevproxy:$(VERSION) moov/localdevproxy:latest
else
	@echo "Skipping Docker builds on TravisCI"
endif

.PHONY: generate
generate:
	wget -O site/rapidoc-min.js https://raw.githubusercontent.com/mrin9/RapiDoc/7.2.1/dist/rapidoc-min.js

serve:
	@echo Load http://localhost:8000 in a web browser...
	@docker run --read-only -p '8000:8080' -v $(shell pwd)/nginx/cache/:/var/cache/nginx -v $(shell pwd)/nginx/run/:/var/run -it moov/api:latest

dist: build
ifeq ($(OS),Windows_NT)
	CGO_ENABLED=1 GOOS=windows go build -o bin/apitest-windows-amd64.exe github.com/moov-io/api/cmd/apitest
else
	CGO_ENABLED=1 GOOS=$(PLATFORM) go build -o bin/apitest-$(PLATFORM)-amd64 github.com/moov-io/api/cmd/apitest
endif

test:
ifeq ($(OS),Linux)
	docker run moov/apitest:latest
	docker run moov/apitest:latest -oauth
else
	@echo "No tests to run"
endif

# From https://github.com/genuinetools/img
.PHONY: AUTHORS
AUTHORS:
	@$(file >$@,# This file lists all individuals having contributed content to the repository.)
	@$(file >>$@,# For how it is generated, see `make AUTHORS`.)
	@echo "$(shell git log --format='\n%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf)" >> $@

release: docker AUTHORS
	go vet ./...
	go test ./...
	git tag -f $(VERSION)

release-push:
	docker push moov/api:$(VERSION)
	docker push moov/apitest:$(VERSION)
	docker push moov/localdevproxy:$(VERSION)
	docker push moov/localdevproxy:latest

.PHONY: tag
tag:
	git tag $(VERSION)
	git push origin $(VERSION)
