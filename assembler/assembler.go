package assembler

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
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

// cleans every line of code getting rid of comments and ensuring everything is in the correct format. Returns (instruction, error)
func ParseLine(line string) (uint32, error) {
	fmt.Println(line)
	//directive
	if line[0] == '.' {
		fmt.Println("directive")
	} else {
		var str_slice = strings.SplitAfterN(line, " ", 5)
		//is label
		if strings.HasSuffix(str_slice[0], ":") {
			fmt.Println("label")
			return 0, nil
		}

		str_slice[0] = strings.TrimSpace(strings.ToLower(str_slice[0]))
		str_slice[1] = strings.TrimSuffix(strings.TrimSpace(str_slice[1]), ",")
		str_slice[2] = strings.TrimSuffix(strings.TrimSpace(str_slice[2]), ",")
		if len(str_slice) > 3 {
			str_slice[3] = strings.TrimSpace(str_slice[3]) //Risc-V can have at most 3 operands so trim comma off of them if possible
		}
		//Checks if there are any more arguments and return error. 4th index can only be comments
		if len(str_slice) > 4 && len(str_slice[4]) > 0 && str_slice[4][0] != '#' {
			return 0, errors.New("extra arguments given")
		}

		itype := InstrTable[str_slice[0]]
		instruction := uint32(0x00000000)
		switch itype.fmt {
		case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7
			rd, inRd := regMap[str_slice[1]]
			rs1, inRs1 := regMap[str_slice[2]]
			rs2, inRs2 := regMap[str_slice[3]]
			if !inRd || !inRs1 || !inRs2 {
				return 0, errors.New("invalid registers")
			}
			instruction |= uint32(itype.Opcode)
			instruction |= uint32(rd) << 7
			instruction |= uint32(itype.funct3) << 12
			instruction |= uint32(rs1) << 15
			instruction |= uint32(rs2) << 20
			//fmt.Printf("%032b\n", instruction)
			return uint32(instruction), nil
		case I: // immediate / loads / jalr: rd, rs1, imm  OR  lw rd, offset(rs1)
			rd, inRd := regMap[str_slice[1]]
			rs1, inRs1 := regMap[str_slice[2]]
			immediate, err := strconv.Atoi(str_slice[3])
			if err != nil {
				log.Fatal(err)
			}
			if str_slice[0] == "ssli" || str_slice[0] == "srli" || str_slice[0] == "srai" {
				immediate &= 0x1F
				if str_slice[0] == "srai" {
					immediate |= 0x20 << 5
				}
			}
			if !inRd || !inRs1 {
				return 0, errors.New("invalid registers")
			}
			instruction |= uint32(itype.Opcode)
			instruction |= uint32(rd) << 7
			instruction |= uint32(itype.funct3) << 12
			instruction |= uint32(rs1) << 15
			instruction |= uint32(immediate) << 20
			fmt.Printf("%032b\n", instruction)
			return uint32(instruction), nil
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
	return 0, nil
}
