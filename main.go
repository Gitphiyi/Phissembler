package main

import (
	"fmt"
	"phissembler/assembler"
)

func main() {
	filename := "assembly/asm_example.s"
	var file_lines = assembler.ParseFile(filename)
	bin_sz := assembler.FirstPass(file_lines)
	fmt.Printf("\nbinary size: 0x%0X \n", bin_sz)
	assembler.SecondPass(file_lines, bin_sz)
	//assembler.Print_Info()
	assembler.Print_Bin("assembly.bin")
}
