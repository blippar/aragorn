# blippar/aragorn: requires go 1.8+ (time.Until)
FROM golang:1.8-alpine AS builder

RUN apk add --no-cache make

ARG VERSION="unknown"
ENV ARAGORN_VERSION="$VERSION"

COPY .  /go/src/github.com/blippar/aragorn
WORKDIR /go/src/github.com/blippar/aragorn

RUN make VERSION="${MGOEXPORT_VERSION}" static

FROM alpine:3.6 AS runtime

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/blippar/aragorn/bin/aragorn /usr/bin/aragorn

ENTRYPOINT ["/usr/bin/aragorn"]
