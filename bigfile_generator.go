package main

import (
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateString(n int) string {
	b := make([]rune, n)
	for i := range b {
		if i == len(b)-1 {
			break
		}
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	b[len(b)-1] = '\n'
	return string(b)
}

func GenerateFile(filename string, num_of_strings, max_string_len int) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	for i := 0; i < num_of_strings; i++ {
		string_len := 2 + rand.Intn(max_string_len-2)
		line := generateString(string_len)

		_, err = f.WriteString(line)
		if err != nil {
			panic(err)
		}
	}
}

/*
func main() {
	generateFile("bigfile.txt", 1000, 100)
}
*/
