.PHONY: all ytfetcher ytfetcher_test

all:
	make ytfetcher
	make ytfetcher_test

ytfetcher:
	go get -u github.com/mattn/go-sqlite3
	go get -u golang.org/x/crypto/pbkdf2
	go get -u github.com/PuerkitoBio/goquery
	go build -i -o ytfetcher main.go

ytfetcher_test:
	go build -i -o ytfetcher_test testing/*.go

clean:
	rm -f ytfetcher
	rm -f ytfetcher_test
