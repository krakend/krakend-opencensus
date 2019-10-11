OS := $(shell uname | tr '[:upper:]' '[:lower:]')

all: deps test

deps:
	go get .

test:
	go test -cover ./...