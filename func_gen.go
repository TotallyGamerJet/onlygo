package main

import "io"

type FuncGen struct {
	PreCall  func()
	PostCall func()
	MovInst  func(*Type)
	RetInst  func(*Type)
	GenCall  func(string, bool)
}

var generators = map[string]map[string]func(io.Writer, Function) FuncGen{
	"darwin": {
		"arm64": newArm64FuncGen,
		"amd64": newAmd64FuncGen,
	},
	"ios": {
		"arm64": newArm64FuncGen,
	},
	"linux": {
		"arm64": newArm64FuncGen,
	},
}
