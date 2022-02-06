package main

import (
	"fmt"
	"io"
)

func NewArm64FuncGen(w io.Writer, fn Function) FuncGen {
	var GPRL = [...]string{"R0", "R1", "R2", "R3", "R4", "R5", "R6", "R7", "R16"}
	var FPRL = [...]string{"F0", "F1", "F2", "F3"}
	return FuncGen{
		PreCall: func() {
			fmt.Fprintf(w, "\tBL runtime·entersyscall(SB)\n")
		},
		PostCall: func() {
			fmt.Fprintf(w, "\tBL runtime·exitsyscall(SB)\n")
		},
		MovInst: func() func(*Type) {
			var offset int // current offset so far
			var intC int   // the number of ints put so far
			var floatC int // the number of floats put so far
			return func(ty *Type) {
				pad := func(to int) {
					for offset%to != 0 {
						offset++
					}
				}
				if intC >= len(GPRL)-1 {
					intC = len(GPRL) - 1 // push the value onto stack
					defer func() {
						fmt.Fprintf(w, "\tSTP (R16, R16), -16(RSP)\n")
					}()
				}
				switch ty.kind {
				case U8, I8:
					fmt.Fprintf(w, "\tMOVBU _%s+%d(FP), %s\n", ty.name, offset, GPRL[intC])
					offset += 1
					intC++
					return
				case PTR, INT, UINT, I64:
					pad(8)
					fmt.Fprintf(w, "\tMOVD _%s+%d(FP), %s\n", ty.name, offset, GPRL[intC])
					offset += 8
					intC++
					return
				case I32:
					pad(4)
					fmt.Fprintf(w, "\tMOVW _%s+%d(FP), %s\n", ty.name, offset, GPRL[intC])
					offset += 4
					intC++
					return
				case U32:
					pad(4)
					fmt.Fprintf(w, "\tMOVWU _%s+%d(FP), %s\n", ty.name, offset, GPRL[intC])
					offset += 4
					intC++
					return
				case F32:
					fmt.Fprintf(w, "\tFMOVD _%s+%d(FP), %s\n", ty.name, offset, FPRL[floatC])
					offset += 4
					floatC++
					return
				default:
					panic(ty.kind)
				}
			}
		}(),
		RetInst: func(ty *Type) {
			var retLoc int
			for _, a := range fn.args {
				switch a.kind {
				case PTR, INT, I64, F64:
					retLoc += 8
				case U32, I32, F32:
					retLoc += 4
				case I8, U8:
					retLoc++
				default:
					panic(fmt.Sprintf("%+v\n", a))
				}
			}
			for retLoc%8 != 0 {
				retLoc++
			}
			switch ty.kind {
			case I8, U8:
				fmt.Fprintf(w, "\tMOVB R0, ret+%d(FP)\n", retLoc)
			case U32, I32:
				fmt.Fprintf(w, "\tMOVW R0, ret+%d(FP)\n", retLoc)
			case PTR, INT, I64, U64:
				fmt.Fprintf(w, "\tMOVD R0, ret+%d(FP)\n", retLoc)
			default:
				panic(ty.kind)
			}
		},
		GenCall: func(name string) {
			fmt.Fprintf(w, "\tMOVD ·_%s(SB), R16\n\tCALL R16\n", name)
		},
	}
}
