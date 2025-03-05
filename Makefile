.PHONY: test build clean

build:
	./build/all.sh

test:
	go test ./...

testv:
	go test -v ./...

clean:
	rm -rf release/

lint:
	go fmt ./...

# Build the templ files
templ:
	templ generate ./cmd/dashboard/ui/templates/...