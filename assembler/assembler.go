package assembler

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var symbolTable = make(map[string]*Symbol) //symbol mapping
var valueTable = make(map[string]ilen)     //for equ
var sectionTable = make(map[string]*Section)
var instr_addresses = make([]ilen, 0, 10)
var quoted = regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"`)

func Print_Bin(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("error reading bin")
	}
	buf := make([]byte, 1)
	//offset := 0
	cnt := 0
	for {
		_, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln("failed to load byte")
		}
		if cnt%30 == 0 {
			fmt.Println()
		}
		cnt++
		//fmt.Printf("%08b ", buf[0])
		fmt.Printf("%02X ", buf[0])
	}
	fmt.Printf("\n# of bytes = %d\n", cnt)
	defer file.Close()

}
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
		// Remove comments, if any
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return lines
}

// Loop through every directve/instruction. Record which section each one is in and the offset address it is in each respective section. Returns binary file size
func FirstPass(instructions []string) ilen {
	var section = ""
	var addr ilen = 0x0

	for i := 0; i < len(instructions); i++ {
		instr_addresses = append(instr_addresses, addr)
		new_addr, err := FirstPassLine(instructions[i], ilen(addr), &section)
		if err != nil {
			log.Fatal(err)
		}
		addr = new_addr
	}
	return addr
}

// loop through every instruction and plug in addresses of .words, .dword, .equ etc into instructions. Fill in actual value into memory for .words and such
func SecondPass(instructions []string, bin_sz ilen) {
	fmt.Printf("\n Second pass starting: \n\n")
	bin_file, err := os.Create("assembly.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer bin_file.Close()

	var section = ""
	var byte_arr = make([]byte, bin_sz)
	for i := 0; i < len(instructions); i++ {
		BinGenerationLine(i, byte_arr, instructions[i], &section)
	}
	err = binary.Write(bin_file, binary.LittleEndian, byte_arr)
	if err != nil {
		log.Fatalf("could not write to binary file")
	}
}

// cleans every line of code getting rid of comments and ensuring everything is in the correct format. Returns (instruction, addr, error)
func FirstPassLine(line string, curr_addr ilen, section *string) (ilen, error) {
	var next_addr = curr_addr
	var op_split = strings.SplitN(line, " ", 2)
	//line is a directive
	if line[0] == '.' {
		switch op_split[0] {
		case ".org": //set location counter to absolute offset line[1]
			val, err := strconv.ParseUint(strings.TrimSpace(op_split[1]), 0, 32)
			if err != nil {
				log.Fatal("immediate given is not decimal, hex, nor binary")
			}
			sz := align_size(reg(val), 4)
			next_addr = ilen(sz)

		case ".align": //align to specified boundary
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
			}
			alignment, err := strconv.ParseUint(strings.TrimSpace(op_split[1]), 0, 32)
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
			symbolTable[op_split[1]] = &Symbol{section: sectionTable[*section], name: op_split[1], offset: curr_addr - secBase, global: true}

		case ".local":
			if *section == "" {
				*section = ".text"
				sectionTable[*section] = &Section{name: ".text", addr: curr_addr, sz: 0}
			}
			secBase := sectionTable[*section].addr
			symbolTable[op_split[1]] = &Symbol{section: sectionTable[*section], name: op_split[1], offset: curr_addr - secBase, global: false}

		case ".equ": //only for instruction length sized stuff
			//don't need to populate .equ since it doesn't matter what section or address it is at
			args := strings.SplitN(op_split[1], ",", 2)
			valueTable[args[0]] = handle_equ(strings.TrimSpace(args[1]))

		case ".section":
			*section = op_split[1]
			sectionTable[op_split[1]] = &Section{name: op_split[1], addr: curr_addr, sz: 0}
			//next_addr += ILEN_BYTES

		case ".text", ".data", ".bss", ".rodata":
			*section = op_split[0]
			sectionTable[op_split[0]] = &Section{name: op_split[0], addr: curr_addr, sz: 0}
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
			zero_sz, err := strconv.Atoi(strings.TrimSpace(op_split[1]))
			if err != nil {
				log.Fatalf("atoi .zero failed")
			}
			sec := sectionTable[*section]
			sec.sz += ilen(zero_sz)
			next_addr += align_addr(ilen(zero_sz))

		case ".half": // 16 bit words
			words := strings.Split(line[5:], ",")
			word_sz := ilen(len(words)) * 2
			sec := sectionTable[*section]
			sec.sz += word_sz
			next_addr += word_sz

		case ".word": // 32 bit words
			words := strings.Split(line[5:], ",")
			word_sz := ilen(len(words)) * 4
			sec := sectionTable[*section]
			sec.sz += word_sz
			next_addr += align_addr(word_sz)

		case ".dword": // 64 bit words
			words := strings.Split(line[6:], ",")
			word_sz := ilen(len(words)) * 8
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
		if strings.HasSuffix(op_split[0], ":") {
			op_split[0] = strings.TrimSuffix(op_split[0], ":")
			symbol, ok := symbolTable[op_split[0]]
			if ok {
				symbol.offset = curr_addr - sec.addr
			} else {
				symbolTable[op_split[0]] = &Symbol{section: sectionTable[*section], name: op_split[0], offset: curr_addr - sec.addr, global: false}
			}

		} else {
			//is instruction
			if *section != ".text" {
				fmt.Printf("section is %s\n", *section)
				return 0, errors.New("instructions must be in text")
			}
			next_addr += ILEN_BYTES
			sec.sz += ILEN_BYTES
		}
		fmt.Printf("(0x%X) %s\n", curr_addr, line)
		return next_addr, nil
	} // Instruction & labels
}

func BinGenerationLine(curr_idx int, bin_arr []byte, line string, section *string) (ilen, error) {
	//default is .text
	if *section == "" {
		*section = ".text"
		sectionTable[*section] = &Section{name: ".text", addr: instr_addresses[curr_idx], sz: 0}
	}

	var next_addr = instr_addresses[curr_idx]
	var op_split = strings.SplitN(line, " ", 2)

	//line is a directive
	if line[0] == '.' {
		switch op_split[0] {
		case ".org": //set location counter to absolute offset line[1]
			//ignore if it is last instruction
			if len(instr_addresses) == curr_idx+1 {
				break
			}
			pad_size := instr_addresses[curr_idx+1] - instr_addresses[curr_idx]
			targAddr := instr_addresses[curr_idx] + ilen(pad_size)
			for i := instr_addresses[curr_idx]; i < targAddr; i += 4 {
				if *section == ".text" {
					bin_arr[i] = 0x13 // NOP instruction
				} else {
					bin_arr[i] = 0x00
				}
				bin_arr[i+1] = 0x00
				bin_arr[i+2] = 0x00
				bin_arr[i+3] = 0x00
			}
		case ".align": //align to specified boundary
			//ignore if it is last instruction
			if len(instr_addresses) == curr_idx+1 {
				break
			}
			pad_size := instr_addresses[curr_idx+1] - instr_addresses[curr_idx]
			targAddr := instr_addresses[curr_idx] + ilen(pad_size)
			for i := instr_addresses[curr_idx]; i < targAddr; i += 1 {
				if *section == ".text" {
					bin_arr[i] = 0x13 // NOP instruction
				} else {
					bin_arr[i] = 0x00
				}
			}
		case ".globl", ".local", ".equ", ".zero":
			break
		case ".section":
			*section = op_split[1]
		case ".text", ".data", ".bss", ".rodata":
			*section = op_split[0]
		case ".asciz":
			asciz, err := strconv.Unquote(quoted.FindString(line))
			if err != nil {
				log.Fatalf("unquoting asciz failed")
			}
			i := instr_addresses[curr_idx]
			for ; i < instr_addresses[curr_idx]+ilen(len(asciz)); i++ {
				bin_arr[i] = asciz[i-instr_addresses[curr_idx]]
			}
			bin_arr[i] = 0 //automatic terminator
		case ".half": // 16 bit words
			words := strings.Split(op_split[1], ",")
			i := instr_addresses[curr_idx]
			cnt := 0
			for ; i < instr_addresses[curr_idx]+ilen(2*len(words)); i += 2 {
				num, err := strconv.ParseUint(strings.TrimSpace(words[cnt]), 0, 16)
				if err != nil {
					fmt.Println(err)
					log.Fatalf("converted half word to integer failure") //autmatically checks bounds
				}
				bin_arr[i] = byte((num & 0xFF00) >> 8)
				bin_arr[i+1] = byte(num & 0xFF)
				//fmt.Printf("num = 0x%02X \t 0x%02X\t 0x%02X \n", num, bin_arr[i], bin_arr[i+1])
				cnt++
			}

		case ".word": // 32 bit words
			words := strings.Split(op_split[1], ",")
			i := instr_addresses[curr_idx]
			cnt := 0
			for ; i < instr_addresses[curr_idx]+ilen(4*len(words)); i += 4 {
				num, err := strconv.ParseUint(strings.TrimSpace(words[cnt]), 0, 32)
				if err != nil {
					fmt.Println(err)
					log.Fatalf("converted word to integer failure") //autmatically checks bounds
				}
				bin_arr[i] = byte((num & 0xFF000000) >> 24)
				bin_arr[i+1] = byte((num & 0xFF0000) >> 16)
				bin_arr[i+2] = byte((num & 0xFF00) >> 8)
				bin_arr[i+3] = byte(num & 0xFF)
				//fmt.Printf("num = 0x%02X \t 0x%02X\t 0x%02X \t 0x%02X\t 0x%02X \n", num, bin_arr[i], bin_arr[i+1], bin_arr[i+2], bin_arr[i+3])
				cnt++
			}

		case ".dword": // 64 bit words
			words := strings.Split(op_split[1], ",")
			i := instr_addresses[curr_idx]
			cnt := 0
			for ; i < instr_addresses[curr_idx]+ilen(8*len(words)); i += 8 {
				num, err := strconv.ParseUint(strings.TrimSpace(words[cnt]), 0, 64)
				if err != nil {
					fmt.Println(err)
					log.Fatalf("converted word to integer failure") //autmatically checks bounds
				}
				bin_arr[i] = byte((num & 0xFF00000000000000) >> 56)
				bin_arr[i+1] = byte((num & 0xFF000000000000) >> 48)
				bin_arr[i+2] = byte((num & 0xFF0000000000) >> 40)
				bin_arr[i+3] = byte(num & 0xFF00000000 >> 32)
				bin_arr[i+4] = byte((num & 0xFF000000) >> 24)
				bin_arr[i+5] = byte((num & 0xFF0000) >> 16)
				bin_arr[i+6] = byte((num & 0xFF00) >> 8)
				bin_arr[i+7] = byte(num & 0xFF)
				//fmt.Printf("num = 0x%02X \t 0x%02X\t 0x%02X \t 0x%02X\t 0x%02X \n", num, bin_arr[i], bin_arr[i+1], bin_arr[i+2], bin_arr[i+3])
				cnt++
			}
		default:
			log.Fatal("unknown assembler directive!")
		}
		fmt.Printf("(0x%X) %s\n", instr_addresses[curr_idx], line)
		return next_addr, nil
	} else {
		//sec := sectionTable[*section]
		return next_addr, nil
	} // Instruction & labels
	// eventually add functionality to account for when the immediate is too big
	// itype := InstrTable[str_slice[0]]
	// instruction := ilen(0x000000000)
	// switch itype.fmt {
	// case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7
	// 	rd, inRd := regMap[str_slice[1]]
	// 	rs1, inRs1 := regMap[str_slice[2]]
	// 	rs2, inRs2 := regMap[str_slice[3]]
	// 	if !inRd || !inRs1 || !inRs2 {
	// 		return 0, addr, errors.New("invalid registers")
	// 	}
	// 	instruction |= ilen(itype.Opcode)
	// 	instruction |= ilen(rd) << 7
	// 	instruction |= ilen(itype.funct3) << 12
	// 	instruction |= ilen(rs1) << 15
	// 	instruction |= ilen(rs2) << 20
	// 	//fmt.Printf("%032b\n", instruction)
	// case I: // immediate / loads / jalr: rd, rs1, imm  OR  lw rd, offset(rs1)
	// 	rd, inRd := regMap[str_slice[1]]
	// 	rs1, inRs1 := regMap[str_slice[2]]
	// 	immediate, err := strconv.Atoi(str_slice[3])
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	if str_slice[0] == "ssli" || str_slice[0] == "srli" || str_slice[0] == "srai" {
	// 		immediate &= 0x1F
	// 		if str_slice[0] == "srai" {
	// 			immediate |= 0x20 << 5
	// 		}
	// 	}
	// 	if !inRd || !inRs1 {
	// 		return 0, addr, errors.New("invalid registers")
	// 	}
	// 	instruction |= ilen(itype.Opcode)
	// 	instruction |= ilen(rd) << 7
	// 	instruction |= ilen(itype.funct3) << 12
	// 	instruction |= ilen(rs1) << 15
	// 	instruction |= ilen(immediate) << 20
	// 	fmt.Printf("%032b\n", instruction)

	// case S: // store: rs2, offset(rs1)
	// 	open := strings.Index(str_slice[2], "(")
	// 	close := strings.Index(str_slice[2], ")")
	// 	if open < 0 || close < 0 || close <= open {
	// 		return 0, 0, errors.New("invalid format, expected imm(reg)")
	// 	}
	// 	imm := str_slice[2][:open]
	// 	reg := str_slice[2][open+1 : close]
	// 	immediate, err := strconv.Atoi(imm)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	first_imm := immediate & 0b11111
	// 	sec_imm := immediate & 0b11111110000

	// 	rs1, inRs1 := regMap[reg]
	// 	rs2, inRs2 := regMap[str_slice[3]]
	// 	if !inRs1 || !inRs2 {
	// 		return 0, addr, errors.New("invalid registers")
	// 	}
	// 	instruction |= ilen(itype.Opcode)
	// 	instruction |= ilen(first_imm) << 7
	// 	instruction |= ilen(itype.funct3) << 12
	// 	instruction |= ilen(rs1) << 15
	// 	instruction |= ilen(rs2) << 20
	// 	instruction |= ilen(sec_imm) << 25

	// // case B: // branch: rs1, rs2, label

	// // case U: // upper-immediate: rd, imm

	// // case J: // jump: rd, label

	// // case C:

	// default:
	// 	log.Fatalf("unsupported instruction format %q", itype.fmt)
	// }
	// addr += ILEN_BYTES
	// return ilen(instruction), addr, nil

}

func handle_equ(expression string) ilen {
	fmt.Printf("Expression inside equ: %s \n", expression)
	return 0
}

// rounds v up to instruction length
func align_addr(v ilen) ilen {
	return (v + (ILEN_BYTES - 1)) &^ (ILEN_BYTES - 1)
}

// rounds v up to size param length
func align_size(v reg, sz reg) reg {
	if sz == 0 {
		return v
	}
	return (v + (sz - 1)) &^ (sz - 1)
}
