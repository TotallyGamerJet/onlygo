package main

import (
	"fmt"
	"io"
)

func NewArm64FuncGen(w io.Writer, fn Function) FuncGen {
	var GPRL = []string{"R0", "R1", "R2", "R3", "R4", "R5", "R6"}
	var FPRL = []string{"F0", "F1", "F2", "F3"}
	return FuncGen{
		PreCall: func() {
			fmt.Fprintf(w, "\tBL runtime·entersyscall(SB)\n")
		},
		PostCall: func() {
			fmt.Fprintf(w, "\tBL runtime·exitsyscall(SB)\n")
		},
		MovInst: func() func(*Type, int) {
			var offset int // current offset so far
			return func(ty *Type, index int) {
				pad := func(to int) {
					for offset%to != 0 {
						offset++
					}
				}
				switch ty.kind {
				case U8, I8:
					fmt.Fprintf(w, "\tMOVBU _%s+%d(FP), %s\n", ty.name, offset, GPRL[index])
					offset += 1
					return
				case PTR, INT, UINT:
					pad(8)
					fmt.Fprintf(w, "\tMOVD _%s+%d(FP), %s\n", ty.name, offset, GPRL[index])
					offset += 8
					return
				case U32, I32:
					fmt.Fprintf(w, "\tMOVW _%s+%d(FP), %s\n", ty.name, offset, GPRL[index])
					offset += 4
					return
				case F32:
					fmt.Fprintf(w, "\tFMOVD _%s+%d(FP), %s\n", ty.name, offset, FPRL[index])
					offset += 4
					return
				default:
					panic(ty.kind)
				}
			}
		}(),
		RetInst: func(ty *Type) {
			switch ty.kind {
			case U32:
				fmt.Fprintf(w, "\tMOVW R0, ret+8(FP)\n") // TODO: calculate where return is
			case PTR:
				fmt.Fprintf(w, "\tMOVD R0, ret+8(FP)\n")
			default:
				panic(ty.kind)
			}
		},
		GenCall: func(name string) {
			fmt.Fprintf(w, "\tMOVD ·_%s(SB), R16\n\tCALL R16\n", name)
		},
	}
}
