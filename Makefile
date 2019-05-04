VERSION_MAJOR ?= 0
VERSION_MINOR ?= 1
VERSION_BUILD ?= 0
VERSION ?= v$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_BUILD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ORG := github.com
OWNER := kubedev
REPOPATH ?= $(ORG)/$(OWNER)/line-bot-operator

$(shell mkdir -p ./out)

.PHONY: build
build: out/controller

out/controller:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	  -ldflags="-s -w -X $(REPOPATH)/pkg/version.version=$(VERSION)" \
	  -a -o $@ cmd/main.go

.PHONY: dep 
dep:
	@dep ensure

.PHONY: test
test:
	./hack/test-go.sh

.PHONY: build_images
build_images:
	docker build -t $(OWNER)/line-bot-operator:$(VERSION) .

.PHONY: push_images
push_images:
	docker push $(OWNER)/line-bot-operator:$(VERSION)

.PHONY: clean
clean:
	rm -rf out/

