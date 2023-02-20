build:
	go build
	mv server bin/gemfast-server
	chmod +x bin/gemfast-server

run:
	go run main.go server

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	go clean
	rm -f bin/gemfast-server

all: clean fmt test build