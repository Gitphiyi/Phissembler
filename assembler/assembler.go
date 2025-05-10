package assembler

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// minimal cleaning and just returns slice of all lines that aren't empty or start with comments
func ParseFile(filename string) []string {
	fmt.Printf("Parse Assembly File %s...\n", filename)
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
		if line == "" || (len(line) >= 1 && line[0:1] == "#") {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return lines
}

// cleans every line of code getting rid of comments and ensuring everything is in the correct format
func ParseLine(line string) {
	fmt.Println(line)

	//directive
	if line[0] == '.' {
		fmt.Println("directive")
	} else {
		var str_slice = strings.SplitAfterN(line, " ", 5)

		//is label
		if strings.HasSuffix(str_slice[0], ":") {
			fmt.Println("label")
			return
		}
		str_slice[0] = strings.ToLower(str_slice[0])
		str_slice[1] = strings.TrimSuffix(str_slice[1], ",")
		str_slice[2] = strings.TrimSuffix(str_slice[2], ",") //Risc-V can have at most 3 operands so trim comma off of them if possible

		itype := InstrTable[str_slice[0]]
		switch itype {
		// case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7

		// case I: // immediate / loads / jalr: rd, rs1, imm  OR  lw rd, offset(rs1)

		// case S: // store: rs2, offset(rs1)

		// case B: // branch: rs1, rs2, label

		// case U: // upper-immediate: rd, imm

		// case J: // jump: rd, label

		// case C:

		default:
			log.Fatalf("unsupported instruction format %q", itype.fmt)
		}
		for _, i := range str_slice {
			fmt.Println(i)
		}
	} // Instruction & labels
}

// transforms the assembly to binary
func transform_to_bin() {

}
