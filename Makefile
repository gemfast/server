.PHONY: all test clean
	
build:
	go build -o bin/gemfast-server main.go

run:
	go run main.go start

fmt:
	go fmt ./...

test:
	go test ./...
	
vet:
	go vet ./...

clean:
	go clean
	rm -f bin/gemfast-server

all: clean fmt test build
