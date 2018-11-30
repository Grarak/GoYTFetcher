# GOYTFetcher [![][travis-url]][travis-svg] [![][license-svg]][license-url] [![][report-card-svg]][report-card-url]

GoYTFetcher is a server application written in Golang to serve Youtube videos as audio files
through HTTP.

It also comes with a simple login system, so you can manage users easily.

## Installation

#### Dependencies

* **Golang** (You need Go 1.9 or higher)
* **FFmpeg** (Install with libvorbis enabled)
* **youtube-dl**

#### Build

```
$ go get github.com/Grarak/GoYTFetcher
$ cd $GOPATH/src/github.com/Grarak/GoYTFetcher
$ make install
```

Compiled executable should be in $GOPATH/bin.

## Usage

```
$ ./GoYTFetcher [-p Port] [-yt Youtube API key] [-i Webpage path (where index.html is stored)]
```

All flags are optional. When no port is given it will use 6713.

Youtube API key is used for searching and getting video information. When no key is given then
it will rely on youtube-dl. Only feature which totally depends on the Youtube API is getting
popular videos.

Webpage path is the directory where your index.html is stored, in case you want to host a website
on the same port.

The first user who sign ups will automatically promoted to administrator and can unlock other
users. When you request a video, then the server will first return the audio link from google
and start the downloading of the video at the same time. Once the download is finished and the
same video is requested again, it will serve the local audio file. Both the link from google
the local audio file are encoded in vorbis format. (Audio bitrate: 160kb/s)

## Clients

* **Android:** [YTFetcher](https://github.com/Grarak/YTFetcher)
* **Web** [YTFetcher-Web](https://github.com/Grarak/YTFetcher-Web)

If you want write your own client. Please let me know, then I can write up a documentation
for API calls.

## Libraries

* [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3)
* [pbkdf2](https://godoc.org/golang.org/x/crypto/pbkdf2)
* [PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery)
* [op/go-logging](https://github.com/op/go-logging)
* [rylio/ytdl](https://github.com/rylio/ytdl)

[travis-url]: https://travis-ci.org/Grarak/GoYTFetcher.svg?branch=master
[travis-svg]: https://travis-ci.org/Grarak/GoYTFetcher.svg

[license-url]: https://github.com/Grarak/GoYTFetcher/blob/master/LICENSE
[license-svg]: https://img.shields.io/badge/license-MIT-blue.svg

[report-card-url]: https://goreportcard.com/report/github.com/Grarak/GoYTFetcher
[report-card-svg]: https://goreportcard.com/badge/github.com/Grarak/GoYTFetcher
