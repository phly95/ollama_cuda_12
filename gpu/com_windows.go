package gpu

import (
	"reflect"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Inspired by github.com/kirides/go-d3d

type IUnknownVtbl struct {
	// every COM object starts with these three
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// _QueryInterface2 uintptr
}

func ReflectQueryInterface(self interface{}, method uintptr, interfaceID *windows.GUID, obj interface{}) int32 {
	selfValue := reflect.ValueOf(self).Elem()
	objValue := reflect.ValueOf(obj).Elem()

	hr, _, _ := syscall.SyscallN(
		method,
		selfValue.UnsafeAddr(),
		uintptr(unsafe.Pointer(interfaceID)),
		objValue.Addr().Pointer())

	return int32(hr)
}
