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
			line = strings.TrimSpace(line[:idx])
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
			fmt.Println(args)
			key := strings.TrimSpace(args[0])
			val, err := strconv.ParseInt(strings.TrimSpace(args[1]), 0, 64)
			if err != nil {
				log.Fatalf("equ value is not valid")
			}
			valueTable[key] = ilen(val)

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
	}
	// eventually add functionality to account for when the immediate is too big
	if strings.HasSuffix(op_split[0], ":") {
		return next_addr, nil
	} // is a label
	itype := InstrTable[op_split[0]]
	instruction := ilen(0x0)
	switch itype.fmt {
	case R: // 3 operands: opcode, rd, funct3, rs1, rs2, funct7
		operands := strings.SplitN(op_split[1], ", ", 3)
		rd, inRd := regMap[operands[0]]
		rs1, inRs1 := regMap[operands[1]]
		rs2, inRs2 := regMap[operands[2]]
		if !inRd || !inRs1 || !inRs2 {
			log.Fatalln("invalid registers")
		}
		instruction |= ilen(itype.Opcode)
		instruction |= ilen(rd) << 7
		instruction |= ilen(itype.funct3) << 12
		instruction |= ilen(rs1) << 15
		instruction |= ilen(rs2) << 20
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)
		fmt.Printf("R instr: %032b \n", instruction)
	case I: // immediate / loads / jalr rd, rs1, imm  OR  lw rd, offset(rs1)
		var operands = strings.SplitN(op_split[1], ", ", 3)
		var rd, inRd = regMap[operands[0]]
		var rs1 uint8
		var inRs1 = true
		var immediate ilen

		if len(operands) == 3 {
			rs1, inRs1 = regMap[operands[1]]
			if !inRd && !inRs1 {
				log.Fatalln("invalid I registers")
			}
			val, err := strconv.Atoi(operands[2])
			immediate = ilen(val)
			if err != nil {
				log.Fatalf("cannot convert val into immediate")
			}
		} else {
			open := strings.Index(operands[1], "(")
			close := strings.Index(operands[1], ")")
			if open < 0 || close < 0 || close <= open {
				log.Fatalln("invalid format, expected imm(reg)")
			}
			offset := operands[1][:open]
			addr := operands[1][open+1 : close]
			rs1, inRs1 = regMap[addr]
			if !inRd && !inRs1 {
				log.Fatalln("invalid I registers")
			}

			//check for equ
			val, ok := valueTable[offset]
			if ok {
				immediate = val
			} else {
				val, err := strconv.Atoi(offset)
				immediate = ilen(val)
				if err != nil {
					log.Fatalf("cannot convert val into immediate")
				}
			}
		} //immediate is an address
		if op_split[0] == "ssli" || op_split[0] == "srli" || op_split[0] == "srai" {
			immediate &= 0x1F
			if op_split[0] == "srai" {
				immediate |= 0x20 << 5
			}
		}
		fmt.Printf("%07b %05b %03b %05b %012b \n", itype.Opcode, rd, itype.funct3, rs1, immediate)
		instruction |= ilen(itype.Opcode)
		instruction |= ilen(rd) << 7
		instruction |= ilen(itype.funct3) << 12
		instruction |= ilen(rs1) << 15       //000000001010 01100 000 01000 1100111
		instruction |= ilen(immediate) << 20 //010001010110 01011 000 00110 0010011
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)
		fmt.Printf("I instr: %032b\n", instruction)
	case S: // store: rs2, offset(rs1)
		var operands = strings.SplitN(op_split[1], ", ", 3)
		var first_imm ilen
		var sec_imm ilen
		var rs1, inRs1 = regMap[operands[0]]
		var rs2, inRs2 = uint8(0), true

		open := strings.Index(operands[1], "(")
		close := strings.Index(operands[1], ")")
		if open < 0 || close < 0 || close <= open {
			log.Fatalln("invalid format, expected imm(reg)")
		}
		offset := operands[1][:open]
		addr := operands[1][open+1 : close]
		rs2, inRs2 = regMap[addr]
		if !inRs1 && !inRs2 {
			log.Fatalln("invalid S registers")
		}

		//check for equ
		immediate := ilen(0)
		val, ok := valueTable[offset]
		if ok {
			immediate = val
		} else {
			val, err := strconv.Atoi(offset)
			immediate = ilen(val)
			if err != nil {
				log.Fatalf("cannot convert val into immediate")
			}
		}
		first_imm = immediate & 0b11111
		sec_imm = immediate & 0b11111110000

		instruction |= ilen(itype.Opcode)
		instruction |= ilen(first_imm) << 7
		instruction |= ilen(itype.funct3) << 12
		instruction |= ilen(rs1) << 15
		instruction |= ilen(rs2) << 20
		instruction |= ilen(sec_imm) << 25 //0000000 01010 00110 000 01010 0100011
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)
		fmt.Printf("S instr: %032b\n", instruction)
	case B: // branch: rs1, rs2, label
		var operands = strings.SplitN(op_split[1], ", ", 3)
		var rs1, inRs1 = regMap[operands[0]]
		var rs2, inRs2 = regMap[operands[1]]
		var immediate uint32
		if !inRs1 && !inRs2 {
			log.Fatalln("B registers not valid")
		}
		val, err := strconv.ParseInt(operands[2], 0, 64)
		immediate = uint32(val)
		if err != nil {
			//check if immediate is equ
			val, ok := valueTable[operands[2]]
			if ok {
				immediate = uint32(val)
				if val > 4094 {
					log.Fatalln("immediate doesn't fit in 13 bits")
				}
				goto valid_b_immediate
			}
			//check if immediate is a label
			symbol, ok := symbolTable[operands[2]]
			if ok {
				symb_addr := symbol.offset + symbol.section.addr
				offset := int32(symb_addr) - int32(instr_addresses[curr_idx])
				immediate = uint32(offset)
				if offset < -4096 || offset > 4094 {
					log.Fatalln("immediate doesn't fit in 13 bits")
				}
				goto valid_b_immediate
			}
			log.Fatalf("J instr: %s\n", err)
		} else {
			//sign extend 2's complement to 16 bits
			bitmask := (1 << 13) - 1
			val := uint64(immediate) & uint64(bitmask) // keep lower 13 bits
			signBit13 := uint64(1 << (13 - 1))
			if val&signBit13 != 0 { // sign‑extend to 64 bits
				val |= ^uint64((1 << 13) - 1)
			}
			offset := int64(val)
			immediate = uint32(offset)
			if offset < -4096 || offset > 4094 {
				log.Fatalf("immediate 0b%016b(%d) does not fit in signed 13 bits", offset, offset)
			}
		}
	valid_b_immediate:
		//imm[0] can be dropped because all instructions are byte aligned
		imm_4_1 := (immediate & 0b0000000011110) >> 1
		imm_5_10 := (immediate & 0b0011111100000) >> 5
		imm_11 := (immediate & 0b0100000000000) >> 11
		imm_12 := (immediate & 0b1000000000000) >> 12
		//fmt.Printf("imm: %013b(%d) -> %01b %04b %06b %01b\n", uint16(immediate), int16(immediate), imm_11, imm_4_1, imm_5_10, imm_12)

		instruction |= ilen(itype.Opcode)
		instruction |= ilen(imm_11) << 7
		instruction |= ilen(imm_4_1) << 8
		instruction |= ilen(itype.funct3) << 12
		instruction |= ilen(rs1) << 15
		instruction |= ilen(rs2) << 20
		instruction |= ilen(imm_5_10) << 25 //0000000 01100 00101 001 1000 0 1100011
		instruction |= ilen(imm_12) << 31
		fmt.Printf("B instr: %032b\n", instruction)
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)
	case U: // upper-immediate: rd, imm
		var operands = strings.SplitN(op_split[1], ", ", 2)
		var rd, inRd = regMap[operands[0]]
		if !inRd {
			log.Fatalf("U register is wrong")
		}
		//check if immediate is equ
		immediate := uint32(0)
		val, ok := valueTable[operands[1]]
		if ok {
			immediate = uint32(val)
		} else {
			temp, err := strconv.ParseInt(operands[1], 0, 64)
			immediate = uint32(temp)
			if err != nil {
				log.Fatalf("U instr: %s\n", err)
			}
		}
		//fmt.Printf("imm: %032b(%d) -> %020b \n", immediate, immediate, immediate>>12)
		// only take 12 bits from immediate
		immediate = immediate >> 12
		instruction |= ilen(itype.Opcode)
		instruction |= ilen(rd) << 7
		instruction |= ilen(immediate) << 12
		fmt.Printf("U instr: %032b\n", instruction)
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)

	case J: // jump: rd, label
		var operands = strings.SplitN(op_split[1], ", ", 2)
		var rd, inRd = regMap[operands[0]]
		var immediate uint32
		if !inRd {
			log.Fatalf("J register is wrong")
		}
		val, err := strconv.ParseInt(operands[1], 0, 64)
		immediate = uint32(val)
		if err != nil {
			//check if immediate is equ
			val, ok := valueTable[operands[1]]
			if ok {
				immediate = uint32(val)
				goto valid_j_immediate
			}
			//check if immediate is a label
			symbol, ok := symbolTable[operands[1]]
			if ok {
				symb_addr := symbol.offset + symbol.section.addr
				offset := int32(symb_addr) - int32(instr_addresses[curr_idx])
				immediate = uint32(offset)
				goto valid_j_immediate
			}
			log.Fatalf("J instr: %s\n", err)
		}
	valid_j_immediate:
		//split immediate into J format
		imm_12_19 := (immediate >> 12) & 0xFF
		imm_11 := (immediate >> 11) & 0x1
		imm_1_10 := (immediate >> 1) & 0x3FF
		imm_20 := (immediate >> 20) & 0x1
		//fmt.Printf("imm: %032b(%d) %08b %01b %010b %01b\n", immediate, immediate, imm_12_19, imm_11, imm_1_10, imm_20)

		instruction |= ilen(itype.Opcode)
		instruction |= ilen(rd) << 7
		instruction |= ilen(imm_12_19) << 12
		instruction |= ilen(imm_11) << 20
		instruction |= ilen(imm_1_10) << 21
		instruction |= ilen(imm_20) << 31 //0 0111010101 1 00001010 00110 1101111
		fmt.Printf("J instr: %032b\n", instruction)
		populate_bin_instruction(instruction, instr_addresses[curr_idx], bin_arr)

	default:
		log.Fatalf("unsupported instruction format %q", itype.fmt)
	}
	return next_addr, nil
} // Instruction & labels

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

func populate_bin_instruction(instruction ilen, addr ilen, byte_arr []byte) {
	for i := ilen(0); i < ILEN_BYTES; i++ {
		ibyte := (instruction >> (8 * (ILEN_BYTES - i - 1))) & 0xFF
		//fmt.Printf("%08b, ", ibyte)
		byte_arr[addr+i] = byte(ibyte)
	}
}
