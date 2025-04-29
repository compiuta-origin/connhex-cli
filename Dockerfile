FROM golang:1.24-alpine AS builder

ARG SVC
ARG TARGETARCH
ARG TARGETVARIANT

# Map TARGETARCH to GOARCH
ENV GOARCH=${TARGETARCH}
# If TARGETVARIANT is set (e.g., "v7" for armv7), extract just the number part
# to set GOARM (e.g., "7") - otherwise leave GOARM unset for non-ARM architectures
ENV GOARM=${TARGETVARIANT:+"${TARGETVARIANT#v}"}

WORKDIR /go/src/github.com/compiuta-origin/connhex

COPY . .
RUN apk update \
    && apk add make\
    && make $SVC \
    && mv build/connhex-$SVC /exe

FROM scratch
# Certificate are needed to make TLS secured API calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /exe /
ENTRYPOINT ["/exe"]
