package utils

import (
	"os"
	"strconv"
	"fmt"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func MkDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
}

type Error string

func (err Error) Error() string {
	return string(err)
}

func StringIsEmpty(data string) bool {
	return len(data) == 0
}

func StringArrayContains(array []string, item string) bool {
	for _, value := range array {
		if value == item {
			return true
		}
	}
	return false
}

func FormatMinutesSeconds(minutes, seconds int) string {
	m := strconv.Itoa(minutes)
	if len(m) == 1 {
		m = "0" + m
	}
	s := strconv.Itoa(seconds)
	if len(s) == 1 {
		s = "0" + s
	}
	return fmt.Sprintf("%s:%s", m, s)
}
