VERSION := $(shell grep -Eo '(v[0-9]+[\.][0-9]+[\.][0-9]+(-[a-zA-Z0-9]*)?)' internal/version/version.go)

.PHONY: build docker release

build: AUTHORS
# api.moov.io docker file
	docker build -t moov/api:$(VERSION) -f Dockerfile .
	docker tag moov/api:$(VERSION) moov/api:latest
# apitest binary
	go fmt ./...
	CGO_ENABLED=0 go build -o bin/apitest ./cmd/apitest/
	docker build -t moov/apitest:$(VERSION) -f Dockerfile-apitest .
	docker tag moov/apitest:$(VERSION) moov/apitest:latest

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
	git push --tags origin $(VERSION)
