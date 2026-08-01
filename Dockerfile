# Reproducible build & test for the Port Mortem minimatch → Go port.
#
# Build from this repository (minimatch-go/):
#   docker build -t minimatch-port .
#
#   docker run --rm minimatch-port test
#   docker run --rm minimatch-port vet
#   docker run --rm minimatch-port differential
#   docker run --rm minimatch-port bench
#   docker run --rm minimatch-port fuzz
#   docker run --rm minimatch-port check     # default
#   docker run --rm minimatch-port all
#
# JSON differential tests use committed fixtures (no Node required).
# Live oracle (TestFuzzDifferentialBatch) skips unless a reference tree is
# available; mount a built isaacs/minimatch checkout if needed:
#   docker run --rm -v /path/to/minimatch:/minimatch:ro minimatch-port oracle
# (entrypoint sets up ../minimatch relative to the module when /minimatch exists.)

FROM golang:1.24-bookworm AS go-deps

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM golang:1.24-bookworm

# Node optional: used only by live oracle when a reference tree is mounted.
RUN apt-get update \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY --from=go-deps /go/pkg/mod /go/pkg/mod
COPY . /src/

COPY scripts/docker-entrypoint.sh /usr/local/bin/minimatch-port-entry
RUN chmod +x /usr/local/bin/minimatch-port-entry

ENV CGO_ENABLED=0

ENTRYPOINT ["/usr/local/bin/minimatch-port-entry"]
CMD ["check"]
