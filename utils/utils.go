package utils

import (
	"os"
	"strconv"
	"fmt"
	"net"
	"log"
)

func Panic(err error) {
	if err != nil {
		panic(err)
	}
}

func MkDir(dir string) error {
	return os.MkdirAll(dir, os.ModePerm)
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

func ReverseStringSlice(s []string) {
	for i, j := 0, len(s)-1; i < len(s)/2; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func InterfaceToString(val interface{}) string {
	return fmt.Sprintf("%v", val)
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
