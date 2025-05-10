package main

import (
	"phissembler/assembler"
)

func main() {
	filename := "assembly/asm_example.s"
	var file_lines = assembler.ParseFile(filename)
	for i := 0; i < len(file_lines); i++ {
		assembler.ParseLine(file_lines[i])
	}

}
