.PHONY: all ytfetcher ytfetcher_test

all:
	make ytfetcher
	make ytfetcher_test

ytfetcher:
	go build -i -o ytfetcher main.go

ytfetcher_test:
	go build -i -o ytfetcher_test testing/*.go

lint:
	gometalinter --vendor --disable-all --enable=vet --enable=goimports --enable=vetshadow --enable=golint --enable=ineffassign --enable=goconst --deadline=120s --tests ./...

clean:
	rm -f ytfetcher
	rm -f ytfetcher_test
