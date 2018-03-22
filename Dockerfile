##
## Build phase
##
FROM golang:alpine AS builder

RUN apk add --no-cache make

ARG VERSION
ARG COMMIT_HASH

COPY .  /go/src/github.com/blippar/aragorn
WORKDIR /go/src/github.com/blippar/aragorn

RUN make VERSION="${VERSION}" COMMIT_HASH="${COMMIT_HASH}" static

##
## Runtime image
##
FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/blippar/aragorn/bin/aragorn /usr/bin/aragorn

ENTRYPOINT ["/usr/bin/aragorn"]
