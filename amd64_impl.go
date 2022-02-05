package main

import (
	"fmt"
	"io"
)

func NewAmd64FuncGen(w io.Writer, fn Function) FuncGen {
	var GPRL = [...]string{"DI", "SI", "DX", "CX", "R8", "R9"}
	var FPRL = [...]string{"X0", "X1", "X2", "X3"}
	return FuncGen{
		PreCall: func() {
			fmt.Fprintf(w, "\tCALL runtime·entersyscall(SB)\n")
		},
		PostCall: func() {
			fmt.Fprintf(w, "\tCALL runtime·exitsyscall(SB)\n")
		},
		MovInst: func() func(*Type, int) {
			var offset = 8 // current offset so far
			return func(ty *Type, index int) {
				pad := func(to int) {
					for offset%to != 0 {
						offset++
					}
				}
				switch ty.kind {
				case U8, I8:
					fmt.Fprintf(w, "\tMOVBLZX _%s+%d(SP), %s\n", ty.name, offset, GPRL[index])
					offset += 1
					return
				case PTR, INT, UINT:
					pad(8)
					fmt.Fprintf(w, "\tMOVQ _%s+%d(SP), %s\n", ty.name, offset, GPRL[index])
					offset += 8
					return
				case I32, U32:
					pad(4)
					fmt.Fprintf(w, "\tMOVL _%s+%d(SP), %s\n", ty.name, offset, GPRL[index])
					offset += 4
					return
				case F32:
					fmt.Fprintf(w, "\tMOVSS _%s+%d(SP), %s\n", ty.name, offset, FPRL[index])
					offset += 4
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
			case PTR:
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
