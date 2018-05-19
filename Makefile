.PHONY: all install test

all:
	go build -ldflags "-s -w" -i -o GoYTFetcher main.go

install:
	go install -ldflags "-s" -i

test:
	go build -i -o ytfetcher_test testing/*.go

clean:
	rm -f ytfetcher
	rm -f ytfetcher_test
