# Default config
VERBOSE     ?= false
VERSION     ?= $(shell echo `git describe --tags --dirty  2>/dev/null || echo devel`)
COMMIT_HASH ?= $(shell echo `git rev-parse --short HEAD 2>/dev/null`)
DATE        = $(shell echo `date "+%Y-%m-%d"`)

# Go configuration
GOBIN	  := $(shell which go)
GOENV	  ?=
GOOPT     ?=
GOLDF      = -s -w -X main.commitHash=$(COMMIT_HASH) -X main.buildDate=$(DATE) -X main.version=$(VERSION)

# Docker configuration
DOCKBIN   := $(shell which docker)
DOCKIMG   := blippar/aragorn
DOCKOPTS  += --build-arg VERSION="$(VERSION)" --build-arg COMMIT_HASH="$(COMMIT_HASH)"

# If run as 'make VERBOSE=true', it will pass the '-v' option to GOBIN
ifeq ($(VERBOSE),true)
GOOPT     += -v
else
DOCKOPTS  += -q
endif

# Binary targets configuration
TARGET     = ./bin/aragorn
GOPKGDIR   = ./cmd/aragorn

# Local meta targets
all: $(TARGET)

# Build binaries with GOBIN using target name & GOPKGDIR
$(TARGET):
	$(info >>> Building $@ from $(GOPKGDIR) using $(GOBIN))
	$(if $(GOENV),$(info >>> with $(GOENV) and GOOPT=$(GOOPT)),)
	$(GOENV) $(GOBIN) build $(GOOPT) -ldflags '$(GOLDF)' -o $(TARGET) $(GOPKGDIR)

# Run tests using GOBIN
test:
	$(info >>> Testing ./... using $(GOBIN))
	@$(GOBIN) test -cover -race $(GOOPT) ./...

# Build binaries staticly
static: GOLDF += -extldflags "-static"
static: GOENV += CGO_ENABLED=0
static: $(TARGET)

# Docker
docker:
	$(info >>> Building docker image $(DOCKIMG) using $(DOCKBIN))
	$(DOCKBIN) build $(DOCKOPTS) -t $(DOCKIMG):$(VERSION) -t $(DOCKIMG):latest .

# Clean
clean:
	$(info >>> Cleaning up binaries and distribuables)
	rm -rv $(TARGET)

# Always execute these targets
.PHONY: all $(TARGET) test static docker clean
