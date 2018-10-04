VERSION := v0.2.3-dev

.PHONY: build

build: AUTHORS
	docker build -t moov/api:$(VERSION) -f Dockerfile .
	docker tag moov/api:$(VERSION) moov/api:latest

# From https://github.com/genuinetools/img
.PHONY: AUTHORS
AUTHORS:
	@$(file >$@,# This file lists all individuals having contributed content to the repository.)
	@$(file >>$@,# For how it is generated, see `make AUTHORS`.)
	@echo "$(shell git log --format='\n%aN <%aE>' | LC_ALL=C.UTF-8 sort -uf)" >> $@

release-push:
	docker push moov/api:$(VERSION)
