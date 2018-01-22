FROM golang:alpine AS builder

RUN apk add --no-cache make

ARG VERSION="unknown"
ENV ARAGORN_VERSION="$VERSION"

COPY .  /go/src/github.com/blippar/aragorn
WORKDIR /go/src/github.com/blippar/aragorn

RUN make VERSION="${ARAGORN_VERSION}" static

FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/blippar/aragorn/bin/aragorn /usr/bin/aragorn

ENTRYPOINT ["/usr/bin/aragorn"]
