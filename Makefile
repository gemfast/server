.PHONY: all omnibus test clean
	
build:
	go build -mod=mod
	mv server bin/gemfast-server
	chmod +x bin/gemfast-server

omnibus:
	cd omnibus && bundle install
	cd omnibus && bundle exec omnibus build gemfast

run:
	go run -mod=mod main.go server

fmt:
	go fmt ./...

test:
	go test -mod=mod ./...

clean:
	go clean
	rm -f bin/gemfast-server

all: clean fmt test build
