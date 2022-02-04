package main

type FuncGen struct {
	MovInst func(*Type, int) string
	RetInst func(*Type) string
}
