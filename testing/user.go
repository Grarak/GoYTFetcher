package main

import "encoding/json"
import "github.com/Grarak/GoYTFetcher/utils"

type User struct {
	ApiKey   string `json:"apikey,omitempty"`
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

func (user User) ToJson() []byte {
	b, err := json.Marshal(user)
	utils.Panic(err)
	return b
}
