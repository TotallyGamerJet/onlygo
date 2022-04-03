package main

import (
	"fmt"
	"io"
	"math"
)

// is Half Floating Point (float16)
func isHFP(_ *Type) bool {
	return false
}

// is Single Floating Point (float32)
func isSFP(ty *Type) bool {
	return ty.kind == F32
}

// is Double Floating Point (float64)
func isDFP(ty *Type) bool {
	return ty.kind == F64
}

// is Quad Floating Point (float128)
func isQFP(_ *Type) bool {
	return false
}

// Short-Vector – A data type directly representable in SIMD,
// a vector of 8 bytes or 16 bytes worth of elements. It's
// aligned to its size, either 8 bytes or 16 bytes, where
// each element can be 1, 2, 4, or 8 bytes.
func isSVT(_ *Type) bool {
	return false
}

// isHFA returns true if ty is a Homogeneous Floating-point Aggregate
//– A data type with 2 to 4 identical floating-point members, either floats or doubles.
func isHFA(ty *Type) bool {
	switch ty.kind {
	case ARRAY:
		if !isHFP(ty.underlyingType) && !isSFP(ty.underlyingType) && !isDFP(ty.underlyingType) && !isQFP(ty.underlyingType) {
			return false
		}
		if ty.length >= 2 && ty.length <= 4 {
			return true
		}
		return false
	default:
		return false
	}
}

// isHVA returns true if ty is a (Homogeneous Short-Vector Aggregate)
// – A data type with 2 to 4 identical Short-Vector members.
// A Short-Vector is a data type directly representable in SIMD,
// a vector of 8 bytes or 16 bytes worth of elements. It's aligned
// to its length, either 8 bytes or 16 bytes, where each element can
// be 1, 2, 4, or 8 bytes.
func isHVA(_ *Type) bool {
	return false
}

func isInteger(ty *Type) bool {
	switch ty.kind {
	case I8, I16, I32, I64, INT, U8, U16, U32, U64, UINT:
		return true
	default:
		return false
	}
}

func isPointer(ty *Type) bool {
	return ty.kind == PTR
}

func isComposite(ty *Type) bool {
	return ty.kind == STRUCT
}

// sizeof returns the size in bytes of a type
func sizeof(ty *Type) (ret int) {
	defer func() {
		ret += ty.padding
	}()
	if ty.kind == ARRAY {
		return sizeof(ty.underlyingType) * ty.length
	}
	if ty.kind == STRUCT {
		var total = 0
		for _, t := range ty.fields {
			total += sizeof(t)
		}
		return total
	}
	switch ty.kind {
	case U8, I8:
		return 1
	case U16, I16:
		return 2
	case U32, I32, F32:
		return 4
	case U64, UINT, I64, INT, PTR, F64:
		return 8
	default:
		panic(ty.kind)
	}
}

func newArm64FuncGen(w io.Writer, fn Function) FuncGen {
	var x = [...]string{"R0", "R1", "R2", "R3", "R4", "R5", "R6", "R7"}
	var v = [...]string{"F0", "F1", "F2", "F3", "F4", "F5", "F6", "F7"}
	return FuncGen{
		PreCall: func() {
			_, _ = fmt.Fprintf(w, "\tBL runtime·entersyscall(SB)\n")
		},
		PostCall: func() {
			_, _ = fmt.Fprintf(w, "\tBL runtime·exitsyscall(SB)\n")
		},
		MovInst: func() func(*Type) {
			var offset int // current offset so far
			var NGRN int   // A.1 - the number of ints put so far
			var NSRN int   // A.2 - the number of floats put so far
			var NSAA = 16  // A.3 - the current stack pointer
			pad := func(to int) {
				for offset%to != 0 {
					offset++
				}
			}
			writeFloat32 := func(ty *Type) {
				pad(4)
				_, _ = fmt.Fprintf(w, "\tFMOVD _%s+%d(FP), %s\n", ty.name, offset, v[NSRN])
				offset += 4
			}
			writeFloat64 := func(ty *Type) {
				pad(8)
				_, _ = fmt.Fprintf(w, "\tFMOVD _%s+%d(FP), %s\n", ty.name, offset, v[NSRN])
				offset += 8
			}
			writeU8 := func(ty *Type) {
				_, _ = fmt.Fprintf(w, "\tMOVBU _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 1
			}
			writeI8 := func(ty *Type) {
				_, _ = fmt.Fprintf(w, "\tMOVB _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 1
			}
			writeU16 := func(ty *Type) {
				pad(2)
				_, _ = fmt.Fprintf(w, "\tMOVHU _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 2
			}
			writeI16 := func(ty *Type) {
				pad(2)
				_, _ = fmt.Fprintf(w, "\tMOVH _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 2
			}
			writeU32 := func(ty *Type) {
				pad(4)
				_, _ = fmt.Fprintf(w, "\tMOVWU _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 4
			}
			writeI32 := func(ty *Type) {
				pad(4)
				_, _ = fmt.Fprintf(w, "\tMOVW _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 4
			}
			writeU64 := func(ty *Type) {
				pad(8)
				_, _ = fmt.Fprintf(w, "\tMOVDU _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 8
			}
			writeI64 := func(ty *Type) {
				pad(8)
				_, _ = fmt.Fprintf(w, "\tMOVD _%s+%d(FP), %s\n", ty.name, offset, x[NGRN])
				offset += 8
			}
			return func(ty *Type) {
				// B.1
				// If the argument type is a Composite Type whose size cannot be statically determined by
				// both the caller and the callee, the argument is copied to memory and the argument is
				// replaced by a pointer to the copy. (There are no such types in C/C++ but they exist in
				// other languages or in language extensions).
				// *** Nothing to do bc all types are statically known ***

				// B.2
				// If the argument type is an HFA or an HVA, then the argument is used unmodified.
				// *** HVAs and HFA are unmodified ***

				// B.3
				// If the argument type is a Composite Type that is larger than 16 bytes, then the argument is
				// copied to memory allocated by the caller and the argument is replaced by a pointer to the copy.
				if isComposite(ty) && sizeof(ty) > 16 {
					ty = &Type{
						kind:           PTR,
						underlyingType: ty,
					}
					panic("write pointer") // TODO: write pointer
				}

				// B.4
				// If the argument type is a Composite Type then the size of the argument is rounded
				// up to the nearest multiple of 8 bytes.
				if isComposite(ty) {
					ty.padding += sizeof(ty) % 8
				}

				// C.1
				// If the argument is a Half-, Single-, Double- or Quad- precision Floating-point or
				// Short Vector Type and the NSRN is less than 8, then the argument is allocated to
				// the least significant bits of register v[NSRN]. The NSRN is incremented by one.
				// The argument has now been allocated.
				if isHFP(ty) || isSFP(ty) || isDFP(ty) || isQFP(ty) || isSVT(ty) {
					if NSRN < 8 {
						switch ty.kind {
						case F32:
							writeFloat32(ty)
						case F64:
							writeFloat64(ty)
						default:
							panic(fmt.Sprintf("unknown type: %+v", ty))
						}
						NSRN++
						return
					}
				}

				// C.2
				// If the argument is an HFA or an HVA and there are sufficient unallocated SIMD
				// and Floating-point registers (NSRN + number of members ≤ 8), then the argument
				// is allocated to SIMD and Floating-point Registers (with one register per member
				// of the HFA or HVA). The NSRN is incremented by the number of registers used.
				// The argument has now been allocated.
				if isHFA(ty) || isHVA(ty) {
					if NSRN+ty.length < 8 {
						for i := 0; i < ty.length; i++ {
							switch ty.kind {
							case F32:
								writeFloat32(ty)
							case F64:
								writeFloat64(ty)
							default:
								panic(fmt.Sprintf("unknown type: %+v", ty))
							}
						}
						NSRN++
						return
					}

					// C.3
					// If the argument is an HFA or an HVA then the NSRN is set to 8 and the size of the
					// argument is rounded up to the nearest multiple of 8 bytes.
					NSRN = 8
					ty.padding += ty.length % 8
				}

				// C.4
				// HFA, an HVA, a Quad-precision Floating-point or Short Vector Type
				// then the NSAA is rounded up to the larger of 8 or the Natural Alignment of the argument’s type
				if isHFA(ty) || isHVA(ty) || isQFP(ty) || isSVT(ty) {
					alignTo := int(math.Max(8, float64(sizeof(ty))))
					for NSAA%alignTo != 0 {
						NSAA++
					}
				}

				// C.5
				// If the argument is a Half- or Single- precision Floating Point type, then the size of the
				// argument is set to 8 bytes. The effect is as if the argument had been copied to the least
				// significant bits of a 64-bit register and the remaining bits filled with unspecified values.
				if isHFA(ty) || isSFP(ty) {
					ty.padding = 8 - sizeof(ty)
				}

				// C.6
				// If the argument is an HFA, an HVA, a Half-, Single-, Double- or Quad- precision Floating-point
				// or Short Vector Type, then the argument is copied to memory at the adjusted NSAA. The NSAA is
				// incremented by the size of the argument. The argument has now been allocated.
				if isHFA(ty) || isHVA(ty) || isHFP(ty) || isSFP(ty) || isDFP(ty) || isQFP(ty) || isSVT(ty) {
					panic("TODO:") // TODO: write to stack
					return
				}

				// C.7
				// If the argument is an Integral or Pointer Type, the size of the argument is less than or
				// equal to 8 bytes and the NGRN is less than 8, the argument is copied to the least significant
				// bits in x[NGRN]. The NGRN is incremented by one. The argument has now been allocated.
				if isInteger(ty) || isPointer(ty) {
					if sizeof(ty) <= 8 && NGRN < 8 {
						switch ty.kind {
						case U8:
							writeU8(ty)
						case I8:
							writeI8(ty)
						case U16:
							writeU16(ty)
						case I16:
							writeI16(ty)
						case U32:
							writeU32(ty)
						case I32:
							writeI32(ty)
						case U64, UINT:
							writeU64(ty)
						case I64, INT, PTR:
							writeI64(ty)
						default:
							panic(fmt.Sprintf("unknown type: %+v", ty))
						}
						NGRN++
						return
					}
				}

				// C.8
				// If the argument has an alignment of 16 then the NGRN is rounded up to the next even number.
				if (sizeof(ty)/8)%16 == 0 {
					if NGRN%2 != 0 {
						NGRN++
					}
				}

				// C.9
				// If the argument is an Integral Type, the size of the argument is equal to 16 and the NGRN
				// is less than 7, the argument is copied to x[NGRN] and x[NGRN+1]. x[NGRN] shall contain the
				// lower addressed double-word of the memory representation of the argument. The NGRN is
				// incremented by two. The argument has now been allocated.
				if isInteger(ty) && sizeof(ty)/8 == 16 && NGRN < 7 {
					writeI64(&Type{name: ty.name + "a", kind: I64})
					NGRN++
					writeI64(&Type{name: ty.name + "b", kind: I64})
					NGRN++
				}

				// C.10
				// If the argument is a Composite Type and the size in double-words of the argument is not more
				// than 8 minus NGRN, then the argument is copied into consecutive general-purpose registers,
				// starting at x[NGRN]. The argument is passed as though it had been loaded into the registers
				// from a double-word- aligned address with an appropriate sequence of LDR instructions loading
				// consecutive registers from memory (the contents of any unused parts of the registers are
				// unspecified by this standard). The NGRN is incremented by the number of registers used.
				// The argument has now been allocated.
				if isComposite(ty) && sizeof(ty)/8 < 8-NGRN {
					for _, f := range ty.fields {
						switch f.kind {
						case U8:
							writeU8(ty)
						case I8:
							writeI8(ty)
						case U16:
							writeU16(ty)
						case I16:
							writeI16(ty)
						case U32:
							writeU32(ty)
						case I32, F32:
							writeI32(ty)
						case U64, UINT:
							writeU64(ty)
						case I64, INT, PTR, F64:
							writeI64(ty)
						default:
							panic(fmt.Sprintf("unknown type: %+v", ty))
						}
						NGRN++
					}
					return
				}

				// C.11
				// The NGRN is set to 8.
				NGRN = 8

				// TODO: C.12
				// The NSAA is rounded up to the larger of 8 or the Natural Alignment of the argument’s type..

				// TODO: C.13
				// If the argument is a composite type then the argument is copied to memory at the adjusted NSAA.
				// The NSAA is incremented by the size of the argument. The argument has now been allocated.
				if isComposite(ty) {
					NSAA += sizeof(ty)
				}

				// If the size of the argument is less than 8 bytes then the size of the argument is set to 8 bytes.
				// The effect is as if the argument was copied to the least significant bits of a 64-bit register
				// and the remaining bits filled with unspecified values.
				if sizeof(ty) < 8 {
					ty.padding = 8 - sizeof(ty)
				}

				// TODO: C.15
				// The argument is copied to memory at the adjusted NSAA. The NSAA is incremented by the size of
				// the argument. The argument has now been allocated.
				return
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
				_, _ = fmt.Fprintf(w, "\tMOVB R0, ret+%d(FP)\n", retLoc)
			case U32, I32:
				_, _ = fmt.Fprintf(w, "\tMOVW R0, ret+%d(FP)\n", retLoc)
			case PTR, INT, I64, U64:
				_, _ = fmt.Fprintf(w, "\tMOVD R0, ret+%d(FP)\n", retLoc)
			default:
				panic(ty.kind)
			}
		},
		GenCall: func(name string, resolveDL bool) {
			if resolveDL {
				_, _ = fmt.Fprintf(w, "\tMOVD ·_%s(SB), R16\n\tCALL R16\n", name)
			} else {
				_, _ = fmt.Fprintf(w, "\tCALL _%s(SB)\n", name)
			}
		},
	}
}
