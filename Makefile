build:
	go build -o bin/gemfast ./cmd/gemfast

run:
	go run main.go server

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	go clean
	rm -f bin/gemfast

all: clean fmt test build