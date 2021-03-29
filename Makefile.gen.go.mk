# DO NOT EDIT. Generated with:
#
#    devctl@4.4.0
#

PACKAGE_DIR    := ./bin-dist

APPLICATION    := $(shell go list -m | cut -d '/' -f 3)
BUILDTIMESTAMP := $(shell date -u '+%FT%TZ')
GITSHA1        := $(shell git rev-parse --verify HEAD)
MODULE         := $(shell go list -m)
OS             := $(shell go env GOOS)
SOURCES        := $(shell find . -name '*.go')
VERSION        := $(shell architect project version)
ifeq ($(OS), linux)
EXTLDFLAGS := -static
endif
LDFLAGS        ?= -w -linkmode 'auto' -extldflags '$(EXTLDFLAGS)' \
  -X '$(shell go list -m)/pkg/project.buildTimestamp=${BUILDTIMESTAMP}' \
  -X '$(shell go list -m)/pkg/project.gitSHA=${GITSHA1}'

.DEFAULT_GOAL := build

.PHONY: build build-darwin build-darwin-64 build-linux build-linux-arm64
## build: builds a local binary
build: $(APPLICATION)
	@echo "====> $@"
## build-darwin: builds a local binary for darwin/amd64
build-darwin: $(APPLICATION)-darwin
	@echo "====> $@"
## build-darwin-arm64: builds a local binary for darwin/arm64
build-darwin-arm64: $(APPLICATION)-darwin-arm64
	@echo "====> $@"
## build-linux: builds a local binary for linux/amd64
build-linux: $(APPLICATION)-linux
	@echo "====> $@"
## build-linux-arm64: builds a local binary for linux/arm64
build-linux-arm64: $(APPLICATION)-linux-arm64
	@echo "====> $@"

$(APPLICATION): $(APPLICATION)-v$(VERSION)-$(OS)-amd64
	@echo "====> $@"
	cp -a $< $@

$(APPLICATION)-darwin: $(APPLICATION)-v$(VERSION)-darwin-amd64
	@echo "====> $@"
	cp -a $< $@

$(APPLICATION)-darwin-arm64: $(APPLICATION)-v$(VERSION)-darwin-arm64
	@echo "====> $@"
	cp -a $< $@

$(APPLICATION)-linux: $(APPLICATION)-v$(VERSION)-linux-amd64
	@echo "====> $@"
	cp -a $< $@

$(APPLICATION)-linux-arm64: $(APPLICATION)-v$(VERSION)-linux-arm64
	@echo "====> $@"
	cp -a $< $@

$(APPLICATION)-v$(VERSION)-%-amd64: $(SOURCES)
	@echo "====> $@"
	CGO_ENABLED=0 GOOS=$* GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $@ .

$(APPLICATION)-v$(VERSION)-%-arm64: $(SOURCES)
	@echo "====> $@"
	CGO_ENABLED=0 GOOS=$* GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $@ .

.PHONY: package-darwin-amd64 package-darwin-arm64 package-linux-amd64 package-linux-arm64
## package-darwin-amd64: prepares a packaged darwin/amd64 version
package-darwin-amd64: $(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-darwin-amd64.tar.gz
	@echo "====> $@"
## package-darwin-arm64: prepares a packaged darwin/arm64 version
package-darwin-arm64: $(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-darwin-arm64.tar.gz
	@echo "====> $@"
## package-linux-amd64: prepares a packaged linux/amd64 version
package-linux-amd64: $(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-linux-amd64.tar.gz
	@echo "====> $@"
## package-linux-arm64: prepares a packaged linux/arm64 version
package-linux-arm64: $(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-linux-arm64.tar.gz
	@echo "====> $@"

$(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-%-amd64.tar.gz: DIR=$(PACKAGE_DIR)/$<
$(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-%-amd64.tar.gz: $(APPLICATION)-v$(VERSION)-%-amd64
	@echo "====> $@"
	mkdir -p $(DIR)
	cp $< $(DIR)/$(APPLICATION)
	cp README.md LICENSE $(DIR)
	tar -C $(PACKAGE_DIR) -cvzf $(PACKAGE_DIR)/$<.tar.gz $<
	rm -rf $(DIR)
	rm -rf $<

$(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-%-arm64.tar.gz: DIR=$(PACKAGE_DIR)/$<
$(PACKAGE_DIR)/$(APPLICATION)-v$(VERSION)-%-arm64.tar.gz: $(APPLICATION)-v$(VERSION)-%-arm64
	@echo "====> $@"
	mkdir -p $(DIR)
	cp $< $(DIR)/$(APPLICATION)
	cp README.md LICENSE $(DIR)
	tar -C $(PACKAGE_DIR) -cvzf $(PACKAGE_DIR)/$<.tar.gz $<
	rm -rf $(DIR)
	rm -rf $<

.PHONY: install
## install: install the application
install:
	@echo "====> $@"
	go install -ldflags "$(LDFLAGS)" .

.PHONY: run
## run: runs go run main.go
run:
	@echo "====> $@"
	go run -ldflags "$(LDFLAGS)" -race .

.PHONY: clean
## clean: cleans the binary
clean:
	@echo "====> $@"
	rm -f $(APPLICATION)*
	go clean

.PHONY: imports
## imports: runs goimports
imports:
	@echo "====> $@"
	goimports -local $(MODULE) -w .

.PHONY: lint
## lint: runs golangci-lint
lint:
	@echo "====> $@"
	golangci-lint run -E gosec -E goconst --timeout=15m ./...

.PHONY: test
## test: runs go test with default values
test:
	@echo "====> $@"
	go test -ldflags "$(LDFLAGS)" -race ./...

.PHONY: build-docker
## build-docker: builds docker image to registry
build-docker: build-linux
	@echo "====> $@"
	cp -a $(APPLICATION)-linux $(APPLICATION)
	docker build -t ${APPLICATION}:${VERSION} .
