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

var symbolTable = make(map[string]Symbol) //symbol mapping
var sectionTable = make(map[string]Section)

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

func FirstPass(instructions []string) {
	var section = ""
	var addr ilen = 0x0

	for i := 0; i < len(instructions); i++ {
		_, new_addr, err := ParseLine(instructions[i], ilen(addr), &section)
		if err != nil {
			log.Fatal(err)
		}
		addr = new_addr
	}
}

// cleans every line of code getting rid of comments and ensuring everything is in the correct format. Returns (instruction, addr, error)
func ParseLine(line string, prev_addr ilen, section *string) (ilen, ilen, error) {
	fmt.Println(line)
	//var curr_sec = sectionTable[*section]
	var addr = prev_addr + ILEN_BYTES
	var str_slice = strings.SplitAfterN(line, " ", 5)

	str_slice[0] = strings.TrimSpace(strings.ToLower(str_slice[0]))
	str_slice[1] = strings.TrimSuffix(strings.TrimSpace(str_slice[1]), ",")
	str_slice[2] = strings.TrimSuffix(strings.TrimSpace(str_slice[2]), ",")
	if len(str_slice) > 3 {
		str_slice[3] = strings.TrimSpace(str_slice[3]) //Risc-V can have at most 3 operands so trim comma off of them if possible
	}
	//Checks if there are any more arguments and return error. 4th index can only be comments
	if len(str_slice) > 4 && len(str_slice[4]) > 0 && str_slice[4][0] != '#' {
		return 0, addr, errors.New("extra arguments given")
	}

	//line is a directive
	if line[0] == '.' {
		fmt.Println("directive")
		switch line {
		case ".org": //set location counter to absolute offset line[1]
			val, err := strconv.ParseUint(str_slice[1], 0, 32)
			if err != nil {
				log.Fatal("immediate given is not decimal, hex, nor binary")
			}
			addr = ilen(val)

		case ".align": //align to specified boundary
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = Section{name: "text", addr: addr, sz: 0}
			}
			val, err := strconv.ParseUint(str_slice[1], 0, 32)
			if err != nil {
				log.Fatal("immediate given is not decimal, hex, nor binary")
			}
			remainder := ilen(val) - (addr % ilen(val))
			addr += remainder //aligns address
			sec := sectionTable[*section]
			sec.sz += remainder

		case ".globl":
			symbolTable[str_slice[1]] = Symbol{name: str_slice[1], addr: addr, global: true}

		case ".local":
			symbolTable[str_slice[1]] = Symbol{name: str_slice[1], addr: addr, global: false}

		case ".equ":
			fmt.Println("do the equ shi")
		case ".section":
			fmt.Println("specifics for section")
		case ".text", ".data", ".bss", ".rodata":
			fmt.Println("do the section shi")
		default:
			log.Fatal("unknown assembler directive!")
		}
		return 0, addr, nil
	} else {
		//default is .text
		if *section == "" {
			*section = ".text"
			sectionTable[*section] = Section{name: "text", addr: addr, sz: 0}
		}

		//is label
		if strings.HasSuffix(str_slice[0], ":") {
			str_slice[0] = strings.TrimSuffix(str_slice[0], ":")
			symbolTable[str_slice[0]] = Symbol{name: str_slice[0], addr: addr, global: false}
			return 0, addr, nil
		}

		itype := InstrTable[str_slice[0]]
		instruction := ilen(0x000000000)
		switch itype.fmt {
		case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7
			rd, inRd := regMap[str_slice[1]]
			rs1, inRs1 := regMap[str_slice[2]]
			rs2, inRs2 := regMap[str_slice[3]]
			if !inRd || !inRs1 || !inRs2 {
				return 0, addr, errors.New("invalid registers")
			}
			instruction |= ilen(itype.Opcode)
			instruction |= ilen(rd) << 7
			instruction |= ilen(itype.funct3) << 12
			instruction |= ilen(rs1) << 15
			instruction |= ilen(rs2) << 20
			//fmt.Printf("%032b\n", instruction)
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
				return 0, addr, errors.New("invalid registers")
			}
			instruction |= ilen(itype.Opcode)
			instruction |= ilen(rd) << 7
			instruction |= ilen(itype.funct3) << 12
			instruction |= ilen(rs1) << 15
			instruction |= ilen(immediate) << 20
			fmt.Printf("%032b\n", instruction)
		// case S: // store: rs2, offset(rs1)

		// case B: // branch: rs1, rs2, label

		// case U: // upper-immediate: rd, imm

		// case J: // jump: rd, label

		// case C:

		default:
			log.Fatalf("unsupported instruction format %q", itype.fmt)
		}
		return ilen(instruction), addr, nil

	} // Instruction & labels
}
