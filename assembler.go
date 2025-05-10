package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// All instructions are 32 bits long, 7 bit opcode,
type Instruction_Info struct {
	instruction uint32
	itype       string
}

func parseFile(filename string) []string {
	fmt.Printf("Reading Assembly File %s...\n", filename)
	data, err := os.Open(filename)
	if err != nil {
		data.Close()
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(data)
	lines := make([]string, 0)
	var spaceCollapse = regexp.MustCompile(`\s+`)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = spaceCollapse.ReplaceAllString(line, " ")
		if line == "" || line[0:2] == "//" {
			continue
		}
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return lines
}

func main() {
	filename := "asm_example.S"
	parseFile(filename)

}
