# GOYTFetcher [![Build Status][travis-url]][travis-svg] [![][license-svg]][license-url]

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
$ git clone https://github.com/Grarak/GoYTFetcher
$ cd GoYTFetcher
$ make
```

## Usage

```
$ ./ytfetcher [-p Port] [-yt Youtube API key] [-host Hostname]
```

All flags are optional. When no port is given it will use 6713 and when there is no hostname
then it will use the local IPv4 address.

Youtube API key is used for searching and getting video information. When no key is given then
it will rely on youtube-dl. Only feature which totally depends on the Youtube API is getting
popular videos.

## Clients

* **Android:** [YTFetcher](https://github.com/Grarak/YTFetcher)

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
