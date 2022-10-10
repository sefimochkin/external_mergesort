package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func readAllLines(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func writeAllLines(filename string, lines []string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	for _, line := range lines {

		//line = line + "\n"
		_, err = file.WriteString(line + "\n")
		if err != nil {
			panic(err)
		}
	}
}

func main() {

	for i := 0; i < 100; i++ {
		filename := "bigfile.txt"

		max_string_len := 3 + rand.Intn(1000)
		number_of_lines := 1000 + rand.Intn(100000)
		memory_size := max_string_len*6 + rand.Intn(50000)

		fmt.Printf("number of lines: %v, max string len: %v, memory_size: %v\n", number_of_lines, max_string_len, memory_size)

		GenerateFile(filename, number_of_lines, max_string_len)

		MergeSort(filename, memory_size, max_string_len)

		left_lines := readAllLines(filename)
		sort.Strings(left_lines)

		writeAllLines("correct_"+filename, left_lines)

		right_lines := readAllLines("sorted_" + filename)

		if len(left_lines) != len(right_lines) {
			fmt.Println("Number of lines differs!")
		}

		break_everything := false
		for i := 0; i < len(left_lines); i++ {
			left_line := left_lines[i]
			right_line := right_lines[i]

			if left_line != right_line {
				fmt.Printf("different lines on i: %v\n", i)
				break_everything = true
				break
			}
		}

		if break_everything {
			fmt.Println("BREAK EVERYTHING")
			break
		}

		fmt.Println("everything is correct!")
	}

}
