package utils

import (
	"encoding/base64"
	"crypto/rand"
	"crypto/aes"
	"io"
	"crypto/cipher"
	"fmt"
)

func Encode(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func Decode(text string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(text)
}

func ToURLBase64(buf []byte) string {
	return base64.URLEncoding.EncodeToString(buf)
}

func FromURLBase64(text string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(text)
}

func Encrypt(key []byte, text string) string {
	plaintext := []byte(text)

	if len(plaintext) < aes.BlockSize {
		panic("Text too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return ToURLBase64(ciphertext)
}

func Decrypt(key []byte, cryptoText string) (string, error) {
	ciphertext, err := FromURLBase64(cryptoText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("text too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext), nil
}

func GenerateRandom(length int) []byte {
	buf := make([]byte, length)
	_, err := rand.Read(buf)
	Panic(err)

	return buf
}
