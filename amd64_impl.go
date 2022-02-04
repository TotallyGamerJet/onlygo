package main

import "fmt"

func NewArm64FuncGen() FuncGen {
	var GPRL = []string{"R0", "R1", "R2", "R3", "R4", "R5", "R6"}
	var FPRL = []string{"F0", "F1", "F2", "F3"}
	return FuncGen{
		MovInst: func() func(*Type, int) string {
			var offset int // current offset so far
			return func(ty *Type, index int) (out string) {
				pad := func(to int) {
					for offset%to != 0 {
						offset++
					}
				}
				switch ty.kind {
				case PTR, INT, UINT:
					pad(8)
					out = fmt.Sprintf("\tMOVD _%s+%d(FP), %s\n", ty.name, offset, GPRL[index])
					offset += 8
					return out
				case U32, I32:
					out = fmt.Sprintf("\tMOVW _%s+%d(FP), %s\n", ty.name, offset, GPRL[index])
					offset += 4
					return out
				case F32:
					out = fmt.Sprintf("\tFMOVD _%s+%d(FP), %s\n", ty.name, offset, FPRL[index])
					offset += 4
					return out
				default:
					panic(ty.kind)
				}
			}
		}(),
		RetInst: func(ty *Type) string {
			switch ty.kind {
			case PTR, U32:
				return "MOVD R0, ret+8(FP)"
			default:
				panic(ty.kind)
			}
		},
	}
}
