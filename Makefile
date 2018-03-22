# Default config
VERBOSE		?= false
VERSION		?= $(shell echo `git describe --tags --dirty 2>/dev/null || echo devel`)
COMMIT_HASH	?= $(shell echo `git rev-parse --short HEAD 2>/dev/null`)
DATE		 = $(shell echo `date "+%Y-%m-%d"`)

# Go configuration
GOBIN		:= $(shell which go)
GOENV		?=
GOOPT		?=
GOLDF		 = -s -w -X main.commitHash=$(COMMIT_HASH) -X main.buildDate=$(DATE) -X main.version=$(VERSION)

# Docker configuration
DOCKBIN		:= $(shell which docker)
DOCKIMG		:= blippar/aragorn
DOCKOPTS	+= --build-arg VERSION="$(VERSION)" --build-arg COMMIT_HASH="$(COMMIT_HASH)"
DOCKFILE	:= Dockerfile

# If run as 'make VERBOSE=true', it will pass the '-v' option to GOBIN
ifeq ($(VERBOSE),true)
GOOPT		+= -v
else
DOCKOPTS	+= -q
endif

# Binary targets configuration
TARGET		 = ./bin/aragorn
GOPKGDIR	 = ./cmd/aragorn

# Local meta targets
all: $(TARGET)

# Build binaries with GOBIN using target name & GOPKGDIR
$(TARGET):
	$(info >>> Building $@ from $(GOPKGDIR) using $(GOBIN))
	$(if $(GOENV),$(info >>> with $(GOENV) and GOOPT=$(GOOPT)),)
	$(GOENV) $(GOBIN) build $(GOOPT) -ldflags '$(GOLDF)' -o $(TARGET) $(GOPKGDIR)

## Install in binary in /usr/local/bin/aragorn
install: $(TARGET)
	$(info >>> Install aragorn to /usr/local/bin/aragorn)
	install -m755 $(TARGET) /usr/local/bin/aragorn

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
	$(DOCKBIN) build $(DOCKOPTS) -f $(DOCKFILE) -t $(DOCKIMG):$(VERSION:v%=%) -t $(DOCKIMG):latest .

## Lint the source code
check:
	$(info >>> Linting source code using gometalinter)
	@gometalinter \
		--deadline 10m \
		--vendor \
		--sort="path" \
		--aggregate \
		--enable-gc \
		--disable-all \
		--enable goimports \
		--enable misspell \
		--enable vet \
		--enable deadcode \
		--enable varcheck \
		--enable ineffassign \
		--enable structcheck \
		--enable unconvert \
		--enable gofmt \
		./...

# Clean
clean:
	$(info >>> Cleaning up binaries and distribuables)
	rm -rv $(TARGET)

# Always execute these targets
.PHONY: all $(TARGET) install test static docker check clean
