VERSION := $(shell grep -Eo '(v[0-9]+[\.][0-9]+[\.][0-9]+(-[a-zA-Z0-9]*)?)' internal/version/version.go)

.PHONY: build docker release

build:
	go fmt ./...
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

serve:
	@echo Load http://localhost:8000 in a web browser...
	@docker run -p '8000:8080' -it moov/api:latest

serve-apps:
	@echo Load http://localhost:8000 in a web browser...
	@docker run -p '8000:8080' -it moov/api-apps:latest

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
