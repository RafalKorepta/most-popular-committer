GIT_HASH = $(shell git rev-parse HEAD | tr -d "\n")
VERSION = $(shell git describe --tags --always --dirty --match=*.*.*)
GO_PKGS= \
    github.com/RafalKorepta/most-popular-committer \
    github.com/RafalKorepta/most-popular-committer/pkg/...

OUT=dist/most-popular-committer

define linker_flags
-X github.com/RafalKorepta/most-popular-committer/cmd.Version=$(VERSION) \
-X github.com/RafalKorepta/most-popular-committer/cmd.Commit=$(GIT_HASH)
endef

all: backend
.PHONY: all

init:
	go get -d -u github.com/golang/dep
	go get -u github.com/hairyhenderson/gomplate
	go get -u github.com/tebeka/go2xunit
	go get -u github.com/axw/gocov/...
	go get -u github.com/AlekSi/gocov-xml
	go get -u github.com/onsi/ginkgo/ginkgo
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get github.com/jteeuwen/go-bindata/...
.PHONY: init

backend: lint test-backend build-backend
.PHONY: backend

lint:
	golangci-lint run
.PHONY: lint

test-backend:
	go vet $(GO_PKGS)
	echo "mode: set" > coverage-all.out
	$(foreach pkg,$(GO_PKGS),\
		go test -v -race -coverprofile=coverage.out $(pkg) | tee -a test-results.out || exit 1;\
		tail -n +2 coverage.out >> coverage-all.out || exit 1;)
	go tool cover -func=coverage-all.out
.PHONY: test-backend

build-container-locally: build-linux-backend build-container
.PHONY: build-container-locally

build-container:
	docker build -t rafalkorepta/most-popular-committer:local-latest .
.PHONY: build-container-locally

build-backend:
	go build -ldflags '$(linker_flags) -s' -o $(OUT) main.go
.PHONY: build-backend

build-linux-backend:
	env GOOS=linux GOARCH=amd64 go build -ldflags '$(linker_flags) -s' -o $(OUT) main.go
.PHONY: build-linux-backend

deploy:
	docker build -f Dockerfile -t $(DOCKER_REGISTRY)/most-popular-committer:$(VERSION) .
	docker push $(DOCKER_REGISTRY)/most-popular-committer:$(VERSION)
	docker logout
.PHONY: deploy

run:
	$(OUT) serve
.PHONY: run
