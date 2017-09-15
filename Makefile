# Default config
VERSION   ?= $(shell echo `git describe --tag 2>/dev/null || git rev-parse --short HEAD` | sed -E 's|^v||g')
VERBOSE   ?= false
DISTFOLDER = dist

# Go configuration
GOBIN	  := $(shell which go)
GOENV	  ?=
GOOPT     ?=
GOLDF      = -X main.Version=$(VERSION)

# Docker configuration
DOCKBIN   := $(shell which docker)
DOCKIMG   := blippar/aragorn
DOCKOPTS  += --build-arg VERSION="$(VERSION)"

# If run as 'make VERBOSE=true', it will pass th '-v' option to GOBIN
ifeq ($(VERBOSE),true)
GOOPT     += -v
FPMFLAGS  += --verbose
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
	@$(GOBIN) test $(GOOPT) ./...

# Build binaries staticly
static: GOLDF += -extldflags "-static"
static: GOENV += CGO_ENABLED=0 GOOS=linux
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
.PHONY: all $(TARGET) static test docker clean
