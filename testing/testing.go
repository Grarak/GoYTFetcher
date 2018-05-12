package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
)

var port int

func main() {
	flag.IntVar(&port, "p", 6713, "Which port to use")
	flag.Parse()
}

func createUsers() {
	var wait sync.WaitGroup
	signup := func(i int) {
		signupUser(fmt.Sprintf("someUser%d", i),
			"12345")
		wait.Done()
	}
	for i := 0; i < 100; i++ {
		wait.Add(1)
		go signup(i)
	}
	wait.Wait()
}

func signupUser(name, password string) error {
	user := User{
		Name:     name,
		Password: Encode(password),
	}

	res, err := http.Post(
		getUrl("v1", "users/signup"),
		"application/json",
		bytes.NewBuffer(user.ToJson()))
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("signUp: " + string(b))
	return nil
}

func loginUser(name, password string) error {
	user := User{
		Name:     name,
		Password: Encode(password),
	}

	res, err := http.Post(getUrl(
		"v1", "users/login?"),
		"application/json",
		bytes.NewBuffer(user.ToJson()))
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("login: " + string(b))
	return nil
}

func listUsers(apiKey string) error {
	user := User{ApiKey: apiKey}
	queries := url.Values{}
	queries.Set("page", "2")

	res, err := http.Post(
		getUrl("v1", "users/list?")+queries.Encode(),
		"application/json",
		bytes.NewBuffer(user.ToJson()))
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("list users: " + string(b))
	return nil
}

func createPlaylist(apiKey, name string) error {
	playlist := PlaylistName{
		apiKey, name,
	}

	b, err := json.Marshal(playlist)
	if err != nil {
		return err
	}

	res, err := http.Post(
		getUrl("v1", "users/playlist/create"),
		"application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	b, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("create playlist: " + string(b))
	return nil
}

func searchYoutube(apiKey, searchQuery string) error {
	youtubeSearch := YoutubeSearch{
		apiKey, searchQuery,
	}

	b, err := json.Marshal(youtubeSearch)
	if err != nil {
		return err
	}

	res, err := http.Post(
		getUrl("v1", "youtube/search"),
		"application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	b, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("search youtube: " + string(b))
	return nil
}

func getChartsYoutube(apiKey string) error {
	youtubeSearch := YoutubeSearch{
		Apikey: apiKey,
	}

	b, err := json.Marshal(youtubeSearch)
	if err != nil {
		return err
	}

	res, err := http.Post(
		getUrl("v1", "youtube/getcharts"),
		"application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	b, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("charts youtube: " + string(b))
	return nil
}

func fetchYoutube(apiKey, id string) error {
	youtubeSearch := Youtube{
		Apikey: apiKey,
		Id:     id,
	}

	b, err := json.Marshal(youtubeSearch)
	if err != nil {
		return err
	}

	res, err := http.Post(
		getUrl("v1", "youtube/fetch"),
		"application/json",
		bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	b, err = ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	fmt.Println("fetch youtube: " + string(b))
	return nil
}

func testDatastructures() {
	ranking := &rankingTree{}

	var datas []YoutubeSong
	for i := 0; i < 100; i++ {
		datas = append(datas, YoutubeSong{
			id:    fmt.Sprintf("someid%d", i),
			count: rand.Intn(1)})
	}

	var wait sync.WaitGroup
	for _, youtube := range datas {
		wait.Add(1)
		go func(youtube YoutubeSong) {
			ranking.insert(youtube)
			wait.Done()
		}(youtube)
	}
	wait.Wait()

	ranking.delete(datas[9])
	fmt.Println(fmt.Sprintf("size: %d", ranking.getSize()))

	startNode := ranking.start
	startNode.print("", true, "root")
}

func getUrl(apiVersion, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/api/%s/%s", port, apiVersion, path)
}
