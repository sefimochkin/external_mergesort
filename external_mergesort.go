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

/*
 * getMoreLines() reads buffer_size of bytes from the designated file,
 * extracts complete lines from the string of bytes, and saves the "tail"
 * of the last incomplete line, if there is one.
 */

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
	}

	return lines, nil
}

func flush_buffer(file *os.File, buffer *[]byte) {
	_, err := file.Write(*buffer)
	if err != nil {
		panic(err)
	}
	*buffer = (*buffer)[:0]
}

func flush_and_sort_lines(output_filename string, lines []string) {
	sort.Strings(lines)

	f, err := os.Create(output_filename)
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

		tmp_filename := "tmp_" + strconv.Itoa(tmp_file_i) + ".txt"
		flush_and_sort_lines(tmp_filename, lines)
		split_filenames = append(split_filenames, tmp_filename)
		tmp_file_i += 1
	}

	return split_filenames
}

/*
 * merge_k_files reads sorted lines from filenames using bufferedReader, finds the minimum one
 * and saves it into write_buffer, which is flushed into output_file when full.
 * Deletes the temporary files that are merged.
 */

func merge_k_files(filenames []string, output_filename string, memory_size, max_len_size int) {
	n_buffers := len(filenames)
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

		min_line_bytes := []byte(min_line)
		if cap(write_buffer)-len(write_buffer) < len(min_line_bytes) {
			flush_buffer(output_file, &write_buffer)
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

	for _, filename := range filenames {
		err := os.Remove(filename)
		if err != nil {
			panic(err)
		}
	}
}

func merge_all(filenames []string, output_filename string, memory_size, max_len_size int) {
	max_k_acceptable := 8
	for {
		// check that there's enough space for all buffers
		buffer_size := memory_size/(max_k_acceptable+1) - max_len_size
		if buffer_size >= max_len_size {
			break
		}
		max_k_acceptable = max_k_acceptable / 2
	}

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
	// checks that at least two files would be able to be merged while writing to
	// a third one
	buffer_size := memory_size/(2+1) - max_len_size
	if buffer_size < max_len_size {
		panic("max_len_size is bigger than the smallest possible buffer size!")
	}

	tmp_filenames := split_file(filename, memory_size, max_len_size)
	output_filename := "sorted_" + filename
	merge_all(tmp_filenames, output_filename, memory_size, max_len_size)
}

/*
func main() {
	mergeSort("bigfile.txt", 10000, 100)
}
*/
