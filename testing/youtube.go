package main

type Youtube struct {
	Apikey string `json:"apikey"`
	Id string `json:"id"`
}

type YoutubeSearch struct {
	Apikey string `json:"apikey"`
	Searchquery string `json:"searchquery"`
}
