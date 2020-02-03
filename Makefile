all: deps test

deps:
	go get .

test:
	go test -cover ./...
