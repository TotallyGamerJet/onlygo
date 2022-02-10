package main

import (
	"fmt"
	"io"
)

func newAmd64FuncGen(w io.Writer, fn Function) FuncGen {
	var GPRL = [...]string{"DI", "SI", "DX", "CX", "R8", "R9"}
	var FPRL = [...]string{"X0", "X1", "X2", "X3", "X4", "X5", "X6", "X7"}
	return FuncGen{
		PreCall: func() {
			fmt.Fprintf(w, "\tCALL runtime·entersyscall(SB)\n")
		},
		PostCall: func() {
			fmt.Fprintf(w, "\tCALL runtime·exitsyscall(SB)\n")
		},
		MovInst: func() func(*Type) {
			var offset = 8 // current offset so far
			var intC int   // the number of ints put so far
			var floatC int // the number of floats put so far
			return func(ty *Type) {
				pad := func(to int) {
					for offset%to != 0 {
						offset++
					}
				}
				switch ty.kind {
				case U8, I8:
					fmt.Fprintf(w, "\tMOVBLZX _%s+%d(SP), %s\n", ty.name, offset, GPRL[intC])
					offset += 1
					intC++
					return
				case PTR, INT, UINT, I64:
					pad(8)
					fmt.Fprintf(w, "\tMOVQ _%s+%d(SP), %s\n", ty.name, offset, GPRL[intC])
					offset += 8
					intC++
					return
				case I32, U32:
					pad(4)
					fmt.Fprintf(w, "\tMOVL _%s+%d(SP), %s\n", ty.name, offset, GPRL[intC])
					offset += 4
					intC++
					return
				case F32:
					fmt.Fprintf(w, "\tMOVSS _%s+%d(SP), %s\n", ty.name, offset, FPRL[floatC])
					offset += 4
					floatC++
					return
				default:
					panic(ty.kind)
				}
			}
		}(),
		RetInst: func(ty *Type) {
			var retLoc int = 8
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
				fmt.Fprintf(w, "\tMOVB AX, ret+%d(SP)\n", retLoc)
			case U32, I32:
				fmt.Fprintf(w, "\tMOVL AX, ret+%d(SP)\n", retLoc)
			case PTR, INT, I64, U64:
				fmt.Fprintf(w, "\tMOVQ AX, ret+%d(SP)\n", retLoc)
			default:
				panic(ty.kind)
			}
		},
		GenCall: func(name string) {
			fmt.Fprintf(w, "\tMOVD ·_%s(SB), AX\n\tCALL AX\n", name)
		},
	}
}
