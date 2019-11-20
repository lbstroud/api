PLATFORM=$(shell uname -s | tr '[:upper:]' '[:lower:]')
VERSION=v$(shell date +"%Y.%m.%d").2

.PHONY: build docker release dist test

build:
	go fmt ./...
ifneq ($(TRAVIS_OS_NAME),osx)
# api.moov.io docker file
	docker build --pull -t moov/api:$(VERSION) -f Dockerfile .
	docker tag moov/api:$(VERSION) moov/api:latest
# api.moov.io/apps/ docker file
	docker build --pull -t moov/api-apps:$(VERSION) -f Dockerfile-apps .
	docker tag moov/api-apps:$(VERSION) moov/api-apps:latest
# apitest binary
	CGO_ENABLED=0 go build -o bin/apitest ./cmd/apitest/
	docker build --pull -t moov/apitest:$(VERSION) -f Dockerfile-apitest ./
	docker tag moov/apitest:$(VERSION) moov/apitest:latest
# localdevproxy binary
	CGO_ENABLED=0 go build -o bin/localdevproxy ./cmd/localdevproxy/
	docker build --pull -t moov/localdevproxy:$(VERSION) -f Dockerfile-localdevproxy ./
	docker tag moov/localdevproxy:$(VERSION) moov/localdevproxy:latest
else
	@echo "Skipping Docker builds on TravisCI"
endif

serve:
	@echo Load http://localhost:8000 in a web browser...
	@docker run -p '8000:8080' -it moov/api:latest

serve-apps:
	@echo Load http://localhost:8000 in a web browser...
	@docker run -p '8000:8080' -it moov/api-apps:latest

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
	docker push moov/api-apps:$(VERSION)
	docker push moov/apitest:$(VERSION)
	docker push moov/localdevproxy:$(VERSION)
	docker push moov/localdevproxy:latest

.PHONY: tag
tag:
	git tag $(VERSION)
	git push origin $(VERSION)
