package main

import "io"

type FuncGen struct {
	PreCall  func()
	PostCall func()
	MovInst  func(*Type)
	RetInst  func(*Type)
	GenCall  func(string)
}

var generators = map[string]map[string]func(io.Writer, Function) FuncGen{
	"darwin": {
		"arm64": NewArm64FuncGen,
		"amd64": NewAmd64FuncGen,
	},
}
