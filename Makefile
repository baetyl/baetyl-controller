HOMEDIR := $(shell pwd)
OUTDIR  := $(HOMEDIR)/output

GIT_TAG:=$(shell git tag --contains HEAD)
GIT_REV:=git-$(shell git rev-parse --short HEAD)
VERSION:=$(if $(GIT_TAG),$(GIT_TAG),$(GIT_REV))

GO           = go
GO_MOD       = $(GO) mod
GOTEST       = $(GO) test
GOPKGS       = $$($(GO) list ./... | grep -vE "vendor")

test: test-case
test-case:
	$(GOTEST) -race -cover -coverprofile=coverage.out $(GOPKGS)

clean:
	rm -rf $(OUTDIR)
	rm -rf $(HOMEDIR)/$(MODULE)
	rm -rf $(HOMEDIR)/$(TEST_MODULE)

fmt:
	go fmt ./...

.PHONY: all prepare compile test package clean build
