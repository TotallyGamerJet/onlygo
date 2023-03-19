This project is not maintained. My effort has moved towards [ebitengine/purego](https://github.com/ebitengine/purego). It supports features that this project doesn't like Objective-C runtime bindings and callbacks that can be called from C.

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
starts with `onlygo:open` then followed by the os, architecture and library name.
Do this comment for each OS and arch combo you want to be generated.
```go
//onlygo:open darwin arm64 libSystem.dylib
```
Next, write stub functions for each C function you want to call.
You MUST match the signature exactly so that onlygo can
call the C function properly. Then above the function add a
comment starting with `//onlygo:linkname` that will give the C
name of the function as found in the dynamic library. This directive
is only needed if the go function name doesn't match the C function.

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

OnlyGo will generate a file ending in `*_init.go`. This file contains a function
with the signature `func Init() error` that MUST be called before calling any of the
dynamically linked to functions. This function links the go function to the 
C function.

If you want OnlyGo to resolve the functions at execution time instead of
requiring a call to an init function use the directive: `//onlygo:resolve_with_cgo`.
NOTE: using the directive does NOT hinder the cross-complication benefits of using
OnlyGo. The reason this is not the default is that it is likely to be more unstable.

## Type Guide
TODO:

## Support
Currently only a few OSs and Architectures are partially supported but it
should be easy enough to add more. Look at the [implementations](amd64_impl.go).
 - [x] MacOS
   - [x] AMD64
   - [x] ARM64
 - [x] iOS
   - [x] ARM64
 - [ ] Linux
   - [ ] AMD64
   - [x] ARM64
 - [ ] Windows
   - [ ] AMD64
   - [ ] ARM64

## License
[MIT](LICENSE)
