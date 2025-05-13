package main

import (
	"phissembler/assembler"
)

func main() {
	filename := "assembly/asm_example.s"
	var file_lines = assembler.ParseFile(filename)
	assembler.FirstPass(file_lines)

}
