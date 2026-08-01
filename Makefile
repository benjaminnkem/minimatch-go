.PHONY: help test vet differential check fuzz bench fixtures tidy \
	docker docker-test docker-check docker-bench docker-fuzz all

.DEFAULT_GOAL := help

help:
	@echo "minimatch-go — Port Mortem / library"
	@echo ""
	@echo "  make test           go test ./..."
	@echo "  make vet            go vet ./..."
	@echo "  make differential   Node-oracle JSON differential tests"
	@echo "  make check          vet + test + differential"
	@echo "  make fuzz           native fuzz (~10s)"
	@echo "  make bench          go test -bench"
	@echo "  make fixtures       regenerate JSON oracles (needs ../minimatch)"
	@echo "  make tidy           go mod tidy"
	@echo "  make docker         build judge image (minimatch-port)"
	@echo "  make docker-test    run tests inside Docker"
	@echo "  make docker-check   run check inside Docker"
	@echo "  make docker-bench   run benchmarks inside Docker"
	@echo "  make docker-fuzz    run short fuzz inside Docker"

test:
	go test ./...

vet:
	go vet ./...

differential:
	go test -run 'Differential' -v -count=1

fuzz:
	go test -fuzz=FuzzMatchNoPanic -fuzztime=10s

bench:
	go test -bench=. -benchmem -count=1

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
	@echo "Optional: make fuzz && make bench"
