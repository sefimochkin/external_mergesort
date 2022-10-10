package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
)

type bufferedReader struct {
	reader          *bufio.Reader
	buffer_size     int
	unfinished_line string
	finished        bool
}

func buildBufferedReader(file *os.File, buffer_size int) bufferedReader {
	reader := bufio.NewReader(file)

	return bufferedReader{reader, buffer_size, "", false}
}

func (br *bufferedReader) getMoreLines() ([]string, error) {
	if br.finished {
		return nil, io.EOF
	}

	var err error

	buf := make([]byte, 0, br.buffer_size)

	n, err := io.ReadFull(br.reader, buf[:cap(buf)])
	buf = buf[:n]

	if err != nil {
		if err == io.EOF {
			br.finished = true
			if br.unfinished_line != "" {
				ans := []string{br.unfinished_line}
				br.unfinished_line = ""

				return ans, nil
			}
		}
		if err != io.ErrUnexpectedEOF {
			//fmt.Fprintln(os.Stderr, err)
			return nil, err
		}
	}

	var lines []string

	for i := 0; i < len(buf); i++ {
		if buf[i] == '\n' {
			line_cap := i + 1
			if len(buf) < i+1 {
				line_cap = len(buf)
			}

			new_line := string(buf[0:line_cap])
			if br.unfinished_line != "" {
				//fmt.Printf("restoring unfinished line: %v, with: %v\n", br.unfinished_line, new_line)
				new_line = br.unfinished_line + new_line
				br.unfinished_line = ""
			}
			buf = buf[line_cap:]
			i = 0
			lines = append(lines, new_line)
		}
	}

	if len(buf) > 0 {
		br.unfinished_line = string(buf)
		//fmt.Printf("saving unfinished line: %v\n", br.unfinished_line)
	}

	return lines, nil
}

func flush_buffer(file *os.File, buffer *[]byte) {
	//fmt.Printf("flushing_buffer: %v\n", string(*buffer))
	_, err := file.Write(*buffer)
	if err != nil {
		panic(err)
	}
	*buffer = (*buffer)[:0]
	//fmt.Printf("buffer after flush inside: %v\n", string(*buffer))

}

func flush_sorted(filename_i int, lines []string) string {
	sort.Strings(lines)

	filename := "tmp_" + strconv.Itoa(filename_i) + ".txt"

	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	for _, line := range lines {
		_, err = f.WriteString(line)
		if err != nil {
			panic(err)
		}
	}
	return filename
}

func split_file(filename string, memory_size int, max_len_size int) []string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	buffer_size := memory_size - max_len_size
	buffered_reader := buildBufferedReader(file, buffer_size)

	tmp_file_i := 0
	var split_filenames []string

	for {
		lines, err := buffered_reader.getMoreLines()

		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		tmp_filename := flush_sorted(tmp_file_i, lines)
		split_filenames = append(split_filenames, tmp_filename)
		tmp_file_i += 1
	}

	return split_filenames
}

func merge_k_files(filenames []string, output_filename string, memory_size, max_len_size int) {
	n_buffers := len(filenames)
	//fmt.Printf("n_buffers: %v\n", n_buffers)
	buffer_size := memory_size/(n_buffers+1) - max_len_size // (memory_size - max_len_size*(n_buffers+1)) / (n_buffers + 1)
	// n_buffers + 1 because of write_buffer

	readers := make([]bufferedReader, 0, n_buffers)
	all_lines := make([][]string, 0, n_buffers)
	finished := make([]bool, n_buffers)
	n_finished := 0

	for i, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		buffered_reader := buildBufferedReader(file, buffer_size)
		readers = append(readers, buffered_reader)
		lines, err := readers[i].getMoreLines()
		if err != nil {
			if err == io.EOF {
				all_lines = append(all_lines, []string{})
				finished[i] = true
				n_finished++
				continue
			} else {
				panic(err)
			}
		}

		all_lines = append(all_lines, lines)
	}

	output_file, err := os.Create(output_filename)
	if err != nil {
		log.Fatal(err)
	}
	defer output_file.Close()
	write_buffer := make([]byte, 0, buffer_size)

	for {
		if n_finished == n_buffers {
			break
		}

		var min_set bool
		var min_line string
		var min_line_i int

		for i := 0; i < n_buffers; i++ {
			if finished[i] {
				continue
			}

			line := all_lines[i][0]

			if !min_set || line < min_line {
				min_line_i, min_line = i, line
				min_set = true
			}
		}

		//fmt.Printf("min line: %v\n", min_line)
		min_line_bytes := []byte(min_line)
		if cap(write_buffer)-len(write_buffer) < len(min_line_bytes) {
			flush_buffer(output_file, &write_buffer)
			//fmt.Printf("buffer after flush outside: %v\n", string(write_buffer))
		}
		write_buffer = append(write_buffer, min_line_bytes...)

		if len(all_lines[min_line_i]) > 1 {
			all_lines[min_line_i] = all_lines[min_line_i][1:]
		} else {
			lines, err := readers[min_line_i].getMoreLines()
			if err != nil {
				if err == io.EOF {
					all_lines[min_line_i] = []string{}
					finished[min_line_i] = true
					n_finished++
					continue
				} else {
					panic(err)
				}
			}

			all_lines[min_line_i] = lines
		}
	}

	if len(write_buffer) > 0 {
		flush_buffer(output_file, &write_buffer)
	}

	skip_delete := false
	if !skip_delete {
		for _, filename := range filenames {
			err := os.Remove(filename)
			if err != nil {
				panic(err)
			}
		}
	}

}

func merge_all(filenames []string, output_filename string, memory_size, max_len_size int) {
	max_k_acceptable := 8
	for {
		buffer_size := memory_size/(max_k_acceptable+1) - max_len_size
		if buffer_size >= max_len_size {
			break
		}
		max_k_acceptable = max_k_acceptable / 2
		if max_k_acceptable == 1 {
			panic("max_len_size is bigger than the smallest possible buffer size!")
		}
	}

	//fmt.Println(max_k_acceptable)

	i_pass := 0
	for {
		if len(filenames) <= max_k_acceptable {
			merge_k_files(filenames, output_filename, memory_size, max_len_size)
			return
		}

		var new_filenames []string
		for i := 0; i < len(filenames)-1; i += 2 {
			tmp_filename := "merge_tmp_" + strconv.Itoa(i_pass) + "_" + strconv.Itoa(i/2) + ".txt"
			new_filenames = append(new_filenames, tmp_filename)

			merge_k_files(filenames[i:i+2], tmp_filename, memory_size, max_len_size)
		}

		if len(filenames)%2 == 1 {
			new_filenames = append(new_filenames, filenames[len(filenames)-1])
		}

		filenames = new_filenames
		i_pass++
	}
}

func MergeSort(filename string, memory_size, max_len_size int) {
	tmp_filenames := split_file(filename, memory_size, max_len_size)
	output_filename := "sorted_" + filename
	merge_all(tmp_filenames, output_filename, memory_size, max_len_size)
}

/*
func main() {
	mergeSort("bigfile.txt", 10000, 100)
}
*/
