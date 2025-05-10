package assembler

import "fmt"

// register mapping
var regMap = make(map[string]uint8, 64)

// instruction mappings
var InstrTable = make(map[string]InstrDesc)

// direct map to Opcode includes extensions
type InstrFmt uint8

const (
	R InstrFmt = 0b0110011 // register–register
	I InstrFmt = 0b0010011 // immediate / loads / jalr
	S InstrFmt = 0b0100011 // stores
	B InstrFmt = 0b1100011 // branches
	U InstrFmt = 0b0110111 // lui, auipc
	J InstrFmt = 0b1101111 // jumps
	C InstrFmt = 0b1110011 // system instructions
)

type InstrExt uint8

type InstrDesc struct {
	fmt    InstrFmt
	funct3 uint8
	funct7 uint8
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

func init() {
	populate_regMap()
}
