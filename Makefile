.PHONY: fmt test lint build cross-build package-deb checksums clean

fmt:
	gofmt -w cmd internal

test:
	go test ./...

lint:
	go vet ./...

build:
	mkdir -p dist
	go build -buildvcs=false -o dist/teams-quiet-hours ./cmd/teams-quiet-hours

cross-build:
	./scripts/build.sh

package-deb: cross-build
	./packaging/deb/build-deb.sh

checksums:
	./scripts/checksums.sh

clean:
	rm -rf dist
