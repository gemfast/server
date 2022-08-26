build:
	go build -o bin/gemfast ./cmd/gemfast

build-ctl:
	go build -o bin/gemfast-ctl ./cmd/gemfast-ctl

run:
	go run cmd/gemfast/main.go

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	go clean
	rm -f bin/gemfast
	rm -f bin/gemfast-ctl

all: clean fmt test build build-ctl