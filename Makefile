.PHONY: help test vet differential check fuzz fuzz-diff bench bench-compare fixtures tidy \
	docker docker-test docker-check docker-bench docker-fuzz all

.DEFAULT_GOAL := help

help:
	@echo "minimatch-go — Port Mortem / library"
	@echo ""
	@echo "  make test            go test (library packages)"
	@echo "  make vet             go vet"
	@echo "  make differential    Node-oracle JSON differential tests"
	@echo "  make check           vet + test + differential"
	@echo "  make fuzz            native Go fuzz (~10s)"
	@echo "  make fuzz-diff       60s differential fuzz → fuzz/log.txt (needs Node+../minimatch)"
	@echo "  make bench           go test -bench (library only)"
	@echo "  make bench-compare   Node vs Go → bench/results.json"
	@echo "  make fixtures        regenerate JSON oracles (needs ../minimatch)"
	@echo "  make tidy            go mod tidy"
	@echo "  make docker          build judge image (minimatch-port)"
	@echo "  make docker-check    run check inside Docker"
	@echo ""
	@echo "One-command build:  docker build -t minimatch-port ."

# Library packages only (exclude package-main harnesses under fuzz/ and bench/).
PKGS := $(shell go list ./... | grep -v '/fuzz$$' | grep -v '/bench$$')

test:
	go test $(PKGS)

vet:
	go vet $(PKGS)

differential:
	go test -run 'Differential' -v -count=1 .

fuzz:
	go test -fuzz=FuzzMatchNoPanic -fuzztime=10s .

fuzz-diff:
	go run ./fuzz -duration=60s -out=fuzz/log.txt

bench:
	go test -bench=. -benchmem -count=1 .

bench-compare:
	chmod +x bench/run.sh
	./bench/run.sh

# Regenerate Node oracle fixtures (requires ../minimatch built with npm).
fixtures:
	cd ../minimatch && npm run prepare
	node testdata/generate_patterns.mjs

tidy:
	go mod tidy

check: vet test differential

docker:
	docker build -t minimatch-port .

docker-test: docker
	docker run --rm minimatch-port test

docker-check: docker
	docker run --rm minimatch-port check

docker-bench: docker
	docker run --rm minimatch-port bench

docker-fuzz: docker
	docker run --rm minimatch-port fuzz

all: check
	@echo "Optional: make fuzz-diff && make bench-compare"
