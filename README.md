# onlygo
OnlyGo generates go assembly wrappers into dynamically linked C code
without the use of CGO.

## Install
Install OnlyGo using the following command:

`go install github.com/totallygamerjet/onlygo@latest`


## Usage
Create a go file that will hold the function stubs.
First, create a comment that tells onlygo what dynamic libray
it should be linking to. This comment should peferably be near
the top of the go file for easy discovery. The format of this comment
starts with `onlygo:open` then followed by the os and library name.
Do this comment for each OS you want to be generated.
```go
//onlygo:open darwin libSystem.dylib
```
Next, write stub functions for each C function you want to call.
You MUST match the signature exactly so that onlygo can
call the C function properly. Then above the function add a
comment starting with `//onlygo:linkname` that will give the C
name of the function as found in the dynamic library.

```go
//onlygo:linkname malloc
func Malloc(size uintptr) unsafe.Pointer
```
Finally, just call `onlygo` with a list of go files you want to
generate wrappers for. You may also wish to use a `go:generate`
comment to make this process easier.

```go
//go:generate onlygo libc.go
```
