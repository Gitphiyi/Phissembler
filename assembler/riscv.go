package assembler

import "fmt"

type ilen uint64 //instruction length
const ILEN_BYTES ilen = 8
const BYTE_SZ ilen = 8 //bits per byte

type Section struct {
	name string
	addr ilen
	sz   ilen //byte buffer size in BYTES. initialize this to 0.
}
type Symbol struct {
	section *Section
	name    string
	offset  ilen // offset to section base address
	global  bool
}

// register mapping
var regMap = make(map[string]uint8, 64)

// instruction mappings
var InstrTable = make(map[string]InstrDesc)

// direct map to Opcode if no extension
type InstrFmt uint8

const (
	R InstrFmt = 0b0110011 // register–register
	I InstrFmt = 0b0010011 // immediate / loads / jalr
	S InstrFmt = 0b0100011 // stores
	B InstrFmt = 0b1100011 // branches
	U InstrFmt = 0b0110111 // lui, auipc
	J InstrFmt = 0b1101111 // jumps
	C InstrFmt = 0b1110011 // system
)

type InstrExt uint8

const (
	ExtNone InstrExt = 0b0
)

type InstrDesc struct {
	fmt    InstrFmt
	Opcode uint8
	funct3 uint8
	funct7 uint8
	ext    InstrExt
}

func populate_regMap() {
	abiNames := []string{
		"zero", "ra", "sp", "gp", "tp",
		"t0", "t1", "t2",
		"s0", "s1",
		"a0", "a1", "a2", "a3", "a4", "a5", "a6", "a7",
		"s2", "s3", "s4", "s5", "s6", "s7", "s8", "s9", "s10", "s11",
		"t3", "t4", "t5", "t6",
	}
	for i, reg := range abiNames {
		regMap[reg] = uint8(i)
		regMap[fmt.Sprintf("x%d", i)] = uint8(i)
	}
}
func populate_instrTable() {
	//R Instructions
	InstrTable["add"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x0, funct7: 0x00, ext: ExtNone}
	InstrTable["sub"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x0, funct7: 0x20, ext: ExtNone}
	InstrTable["xor"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x4, funct7: 0x00, ext: ExtNone}
	InstrTable["or"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x6, funct7: 0x00, ext: ExtNone}
	InstrTable["and"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x7, funct7: 0x00, ext: ExtNone}
	InstrTable["sll"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x1, funct7: 0x00, ext: ExtNone}
	InstrTable["srl"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x5, funct7: 0x20, ext: ExtNone}
	InstrTable["sra"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x2, funct7: 0x00, ext: ExtNone}
	InstrTable["slt"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x3, funct7: 0x00, ext: ExtNone}
	InstrTable["sltu"] = InstrDesc{fmt: R, Opcode: uint8(R), funct3: 0x7, funct7: 0x00, ext: ExtNone}
	//I Instructions
	InstrTable["addi"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x0, funct7: 0, ext: ExtNone}
	InstrTable["xori"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x4, funct7: 0, ext: ExtNone}
	InstrTable["ori"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x6, funct7: 0, ext: ExtNone}
	InstrTable["andi"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x7, funct7: 0, ext: ExtNone}
	InstrTable["slli"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x1, funct7: 0, ext: ExtNone}
	InstrTable["srli"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x5, funct7: 0, ext: ExtNone}
	InstrTable["srai"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x5, funct7: 0, ext: ExtNone}
	InstrTable["slti"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x2, funct7: 0, ext: ExtNone}
	InstrTable["sltiu"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x3, funct7: 0, ext: ExtNone}
	InstrTable["lb"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x0, funct7: 0, ext: ExtNone}
	InstrTable["lh"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x1, funct7: 0, ext: ExtNone}
	InstrTable["lw"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x2, funct7: 0, ext: ExtNone}
	InstrTable["lbu"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x4, funct7: 0, ext: ExtNone}
	InstrTable["lhu"] = InstrDesc{fmt: I, Opcode: uint8(I), funct3: 0x5, funct7: 0, ext: ExtNone}
	//S Instructions
	InstrTable["sb"] = InstrDesc{fmt: S, Opcode: uint8(S), funct3: 0x0, funct7: 0, ext: ExtNone}
	InstrTable["sh"] = InstrDesc{fmt: S, Opcode: uint8(S), funct3: 0x1, funct7: 0, ext: ExtNone}
	InstrTable["sw"] = InstrDesc{fmt: S, Opcode: uint8(S), funct3: 0x2, funct7: 0, ext: ExtNone}
	//B Instructions
	InstrTable["beq"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x0, funct7: 0, ext: ExtNone}
	InstrTable["bne"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x1, funct7: 0, ext: ExtNone}
	InstrTable["blt"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x4, funct7: 0, ext: ExtNone}
	InstrTable["bge"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x5, funct7: 0, ext: ExtNone}
	InstrTable["bltu"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x6, funct7: 0, ext: ExtNone}
	InstrTable["bgeu"] = InstrDesc{fmt: B, Opcode: uint8(B), funct3: 0x7, funct7: 0, ext: ExtNone}
	//J Instructions
	InstrTable["jal"] = InstrDesc{fmt: J, Opcode: 0b1101111, funct3: 0, funct7: 0, ext: ExtNone}
	InstrTable["jalr"] = InstrDesc{fmt: I, Opcode: 0b1100111, funct3: 0x0, funct7: 0, ext: ExtNone}
	//U Instructions
	InstrTable["lui"] = InstrDesc{fmt: U, Opcode: 0b0110111, funct3: 0, funct7: 0, ext: ExtNone}
	InstrTable["auipc"] = InstrDesc{fmt: U, Opcode: 0b0010111, funct3: 0, funct7: 0, ext: ExtNone}
	//Transfer Instructions
	InstrTable["ecall"] = InstrDesc{fmt: I, Opcode: 0b1110011, funct3: 0x0, funct7: 0, ext: ExtNone}
	InstrTable["ebreak"] = InstrDesc{fmt: I, Opcode: 0b1110011, funct3: 0x0, funct7: 0, ext: ExtNone}

}

func init() {
	populate_regMap()
	populate_instrTable()
}
