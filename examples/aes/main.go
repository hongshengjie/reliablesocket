package main

import (
	"fmt"
	"reliablesocket/aesutil"
)

func main() {
	key := "hongshengjiehenshuai"
	plaintext := "今天是个好天气"
	chphertext, err := aesutil.Encrypt(aesutil.AES_CBC, (key), []byte(plaintext))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(chphertext))

	dd, err := aesutil.Decrypt(aesutil.AES_CBC, (key), []byte(chphertext))
	fmt.Println(string(dd))
}
