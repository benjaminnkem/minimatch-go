.PHONY: test vet fuzz bench differential tidy

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
