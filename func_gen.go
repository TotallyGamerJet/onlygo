package main

type FuncGen struct {
	PreCall  func()
	PostCall func()
	MovInst  func(*Type, int)
	RetInst  func(*Type)
	GenCall  func(string)
}
