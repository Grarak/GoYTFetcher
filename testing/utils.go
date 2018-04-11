package main

import "encoding/base64"

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func Encode(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func Decode(text string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(text)
}
