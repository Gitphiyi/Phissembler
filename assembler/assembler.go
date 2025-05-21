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

var symbolTable = make(map[string]*Symbol) //symbol mapping
var valueTable = make(map[string]ilen)
var sectionTable = make(map[string]*Section)
var quoted = regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)

func Print_Info() {
	fmt.Println("All .equ values: ")
	for key, val := range valueTable {
		fmt.Printf("  %s: %d\n", key, val)
	}
	for key, val := range sectionTable {
		fmt.Printf("Section: %s (addr, sz) = (0x%X, %d bytes)\n", key, val.addr, val.sz)
		for _, val := range symbolTable {
			if val.section.name == key {
				fmt.Printf("  (%s) offset from section: 0x%X\n", val.name, val.offset)
			}
		}
	}
}

// minimal cleaning and just returns slice of all lines that aren't empty or start with comments
func ParseFile(filename string) []string {
	fmt.Printf("Parsing Assembly File %s...\n", filename)
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

// Loop through every directve/instruction. Record which section each one is in and the offset address it is in each respective section
func FirstPass(instructions []string) {
	var section = ""
	var addr ilen = 0x0

	for i := 0; i < len(instructions); i++ {
		new_addr, err := FirstPassLine(instructions[i], ilen(addr), &section)
		if err != nil {
			log.Fatal(err)
		}
		addr = new_addr
	}
}

// loop through every instruction and plug in addresses of .words, .dword, .equ etc into instructions. Fill in actual value into memory for .words and such
func SecondPass(instructions []string) {
	for i := 0; i < len(instructions); i++ {

	}
}

// cleans every line of code getting rid of comments and ensuring everything is in the correct format. Returns (instruction, addr, error)
func FirstPassLine(line string, curr_addr ilen, section *string) (ilen, error) {
	var next_addr = curr_addr
	var str_slice = strings.SplitAfterN(line, " ", 5)

	str_slice[0] = strings.TrimSpace(strings.ToLower(str_slice[0]))
	if len(str_slice) > 1 && str_slice[1][0] != '#' {
		str_slice[1] = strings.TrimSuffix(strings.TrimSpace(str_slice[1]), ",")
		if len(str_slice) > 2 && str_slice[2][0] != '#' {
			str_slice[2] = strings.TrimSuffix(strings.TrimSpace(str_slice[2]), ",")
			if len(str_slice) > 3 && str_slice[3][0] != '#' {
				str_slice[3] = strings.TrimSpace(str_slice[3]) //Risc-V can have at most 3 operands so trim comma off of them if possible
				//Checks if there are any more arguments and return error. 4th index can only be comments
				if len(str_slice) > 4 && str_slice[4][0] != '#' {
					return next_addr, errors.New("extra arguments given")
				}
			}
		}
	}

	//line is a directive
	if line[0] == '.' {
		switch str_slice[0] {
		case ".org": //set location counter to absolute offset line[1]
			val, err := strconv.ParseUint(str_slice[1], 0, 32)
			if err != nil {
				log.Fatal("immediate given is not decimal, hex, nor binary")
			}
			next_addr = ilen(val)

		case ".align": //align to specified boundary
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
			}
			alignment, err := strconv.ParseUint(str_slice[1], 0, 32)
			if err != nil {
				log.Fatal("immediate given is not decimal, hex, nor binary")
			}
			byte_align := alignment / uint64(BYTE_SZ)
			remainder := ilen(byte_align) - (curr_addr % ilen(byte_align))
			next_addr += remainder //aligns address
			sec := sectionTable[*section]
			sec.sz += remainder

		case ".globl":
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
			}

			secBase := sectionTable[*section].addr
			symbolTable[str_slice[1]] = &Symbol{section: sectionTable[*section], name: str_slice[1], offset: curr_addr - secBase, global: true}

		case ".local":
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
			}
			secBase := sectionTable[*section].addr
			symbolTable[str_slice[1]] = &Symbol{section: sectionTable[*section], name: str_slice[1], offset: curr_addr - secBase, global: false}

		case ".equ": //only for instruction length sized stuff
			//don't need to populate .equ since it doesn't matter what section or address it is at
			valueTable[str_slice[1]] = handle_equ(str_slice[2])

		case ".section":
			*section = str_slice[1]
			sectionTable[str_slice[1]] = &Section{name: str_slice[1], addr: curr_addr, sz: 0}
			//next_addr += ILEN_BYTES

		case ".text", ".data", ".bss", ".rodata":
			*section = str_slice[0]
			sectionTable[str_slice[0]] = &Section{name: str_slice[0], addr: curr_addr, sz: 0}
			//next_addr += ILEN_BYTES

		case ".asciz":
			asciz, err := strconv.Unquote(quoted.FindString(line))
			if err != nil {
				log.Fatalf("unquoting asciz failed")
			}
			str_sz := ilen(len(asciz) + 1) //in bytes including /0
			sec := sectionTable[*section]
			sec.sz += align_addr(ilen(str_sz))
			next_addr += align_addr(ilen(str_sz)) //increases address by aligned amount

		case ".zero":
			zero_sz, err := strconv.Atoi(str_slice[1])
			if err != nil {
				log.Fatalf("atoi .zero failed")
			}
			sec := sectionTable[*section]
			sec.sz += ilen(zero_sz)
			next_addr += align_addr(ilen(zero_sz))

		case ".half": // 16 bit words
			words := strings.Split(line[5:], ",")
			word_sz := ilen(len(words)) * ILEN_BYTES / 4
			sec := sectionTable[*section]
			sec.sz += word_sz
			next_addr += word_sz

		case ".word": // 32 bit words
			words := strings.Split(line[5:], ",")
			word_sz := ilen(len(words)) * ILEN_BYTES / 2
			sec := sectionTable[*section]
			sec.sz += word_sz
			next_addr += align_addr(word_sz)

		case ".dword": // 64 bit words
			words := strings.Split(line[6:], ",")
			word_sz := ilen(len(words)) * ILEN_BYTES
			sec := sectionTable[*section]
			sec.sz += word_sz
			next_addr += align_addr(word_sz)

		default:
			log.Fatal("unknown assembler directive!")
		}
		fmt.Printf("(0x%X) %s\n", curr_addr, line)
		return next_addr, nil
	} else {
		//default is .text
		if *section == "" {
			*section = ".text"
			sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
		}
		sec := sectionTable[*section]
		//is label
		if strings.HasSuffix(str_slice[0], ":") {
			str_slice[0] = strings.TrimSuffix(str_slice[0], ":")
			symbol, ok := symbolTable[str_slice[0]]
			if ok {
				symbol.offset = curr_addr - sec.addr
			} else {
				symbolTable[str_slice[0]] = &Symbol{section: sectionTable[*section], name: str_slice[0], offset: curr_addr - sec.addr, global: false}
			}

		} else {
			//is instruction
			if *section != ".text" {
				fmt.Printf("section is %s\n", *section)
				return 0, errors.New("instructions must be in text")
			}
		}
		next_addr += ILEN_BYTES
		sec.sz += ILEN_BYTES
		fmt.Printf("(0x%X) %s\n", curr_addr, line)
		return next_addr, nil
	} // Instruction & labels
}

// func SecondPassLine(line string, prev_addr ilen, section *string) {
// 	fmt.Println(line)
// 	var addr = prev_addr
// 	var str_slice = strings.SplitAfterN(line, " ", 5)

// 	str_slice[0] = strings.TrimSpace(strings.ToLower(str_slice[0]))
// 	str_slice[1] = strings.TrimSuffix(strings.TrimSpace(str_slice[1]), ",")
// 	str_slice[2] = strings.TrimSuffix(strings.TrimSpace(str_slice[2]), ",")
// 	if len(str_slice) > 3 {
// 		str_slice[3] = strings.TrimSpace(str_slice[3]) //Risc-V can have at most 3 operands so trim comma off of them if possible
// 	}
// 	//Checks if there are any more arguments and return error. 4th index can only be comments
// 	if len(str_slice) > 4 && len(str_slice[4]) > 0 && str_slice[4][0] != '#' {
// 		return 0, addr, errors.New("extra arguments given")
// 	}
// 	// eventually add functionality to account for when the immediate is too big
// 	itype := InstrTable[str_slice[0]]
// 	instruction := ilen(0x000000000)
// 	switch itype.fmt {
// 	case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7
// 		rd, inRd := regMap[str_slice[1]]
// 		rs1, inRs1 := regMap[str_slice[2]]
// 		rs2, inRs2 := regMap[str_slice[3]]
// 		if !inRd || !inRs1 || !inRs2 {
// 			return 0, addr, errors.New("invalid registers")
// 		}
// 		instruction |= ilen(itype.Opcode)
// 		instruction |= ilen(rd) << 7
// 		instruction |= ilen(itype.funct3) << 12
// 		instruction |= ilen(rs1) << 15
// 		instruction |= ilen(rs2) << 20
// 		//fmt.Printf("%032b\n", instruction)
// 	case I: // immediate / loads / jalr: rd, rs1, imm  OR  lw rd, offset(rs1)
// 		rd, inRd := regMap[str_slice[1]]
// 		rs1, inRs1 := regMap[str_slice[2]]
// 		immediate, err := strconv.Atoi(str_slice[3])
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		if str_slice[0] == "ssli" || str_slice[0] == "srli" || str_slice[0] == "srai" {
// 			immediate &= 0x1F
// 			if str_slice[0] == "srai" {
// 				immediate |= 0x20 << 5
// 			}
// 		}
// 		if !inRd || !inRs1 {
// 			return 0, addr, errors.New("invalid registers")
// 		}
// 		instruction |= ilen(itype.Opcode)
// 		instruction |= ilen(rd) << 7
// 		instruction |= ilen(itype.funct3) << 12
// 		instruction |= ilen(rs1) << 15
// 		instruction |= ilen(immediate) << 20
// 		fmt.Printf("%032b\n", instruction)

// 	case S: // store: rs2, offset(rs1)
// 		open := strings.Index(str_slice[2], "(")
// 		close := strings.Index(str_slice[2], ")")
// 		if open < 0 || close < 0 || close <= open {
// 			return 0, 0, errors.New("invalid format, expected imm(reg)")
// 		}
// 		imm := str_slice[2][:open]
// 		reg := str_slice[2][open+1 : close]
// 		immediate, err := strconv.Atoi(imm)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		first_imm := immediate & 0b11111
// 		sec_imm := immediate & 0b11111110000

// 		rs1, inRs1 := regMap[reg]
// 		rs2, inRs2 := regMap[str_slice[3]]
// 		if !inRs1 || !inRs2 {
// 			return 0, addr, errors.New("invalid registers")
// 		}
// 		instruction |= ilen(itype.Opcode)
// 		instruction |= ilen(first_imm) << 7
// 		instruction |= ilen(itype.funct3) << 12
// 		instruction |= ilen(rs1) << 15
// 		instruction |= ilen(rs2) << 20
// 		instruction |= ilen(sec_imm) << 25

// 	// case B: // branch: rs1, rs2, label

// 	// case U: // upper-immediate: rd, imm

// 	// case J: // jump: rd, label

// 	// case C:

// 	default:
// 		log.Fatalf("unsupported instruction format %q", itype.fmt)
// 	}
// 	addr += ILEN_BYTES
// 	return ilen(instruction), addr, nil

// }

func handle_equ(expression string) ilen {
	return 0
}

// rounds v up to instruction length
func align_addr(v ilen) ilen {
	return (v + (ILEN_BYTES - 1)) &^ (ILEN_BYTES - 1)
}
