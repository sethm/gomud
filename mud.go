package main

import "bufio"
import "fmt"
import "io"
import "os"
import "strings"

func print(w io.Writer, msg string, args ...interface{}) {
	bw := bufio.NewWriter(w)
	bw.WriteString(fmt.Sprintf(msg, args...))
	bw.Flush()
}

func println(w io.Writer, msg string, args ...interface{}) {
	print(w, msg, args...)
	print(w, "\r\n")
}

func mainLoop(r io.Reader, w io.Writer) {

	bufReader := bufio.NewReader(r)

	for {
		print(w, "mud> ")

		line, err := bufReader.ReadString('\n')

		if err != nil {
			if err != io.EOF {
				println(w, "Error : %s", err)
			}
			// No more input
			break
		}

		// Clean whitespace off the ends of the string.
		line = strings.TrimSpace(line)

		println(w, "Huh?")
	}
}

func main() {
	in := os.Stdin
	out := os.Stdout

	mainLoop(in, out)
}
