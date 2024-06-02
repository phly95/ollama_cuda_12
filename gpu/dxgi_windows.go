package gpu

// TODO - insert MIT copyright and link to github.com/kirides/go-d3d

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modDXGI                = windows.NewLazySystemDLL("dxgi.dll")
	procCreateDXGIFactory1 = modDXGI.NewProc("CreateDXGIFactory1")
	procCreateDXGIFactory2 = modDXGI.NewProc("CreateDXGIFactory2")

	// iid_IDXGIDevice, _   = windows.GUIDFromString("{54ec77fa-1377-44e6-8c32-88fd5f44c84c}")
	IID_IDXGIDevice1, _ = windows.GUIDFromString("{77db970f-6276-48ba-ba28-070143b4392c}")
	// IID_IDXGIAdapter, _  = windows.GUIDFromString("{2411E7E1-12AC-4CCF-BD14-9798E8534DC0}")
	IID_IDXGIAdapter1, _ = windows.GUIDFromString("{29038f61-3839-4626-91fd-086879011a05}")
	IID_IDXGIAdapter3, _ = windows.GUIDFromString("{645967a4-1392-4310-a798-8053ce3e93fd}")
	IID_IDXGIAdapter4, _ = windows.GUIDFromString("{3c8d99d1-4fbf-4181-a82c-af66bf7bd24e}")
	// static const GUID IID_IDXGIAdapter    = { 0x2411e7e1, 0x12ac, 0x4ccf, { 0xbd, 0x14, 0x97, 0x98, 0xe8, 0x53, 0x4d, 0xc0 } };
	// static const GUID IID_IDXGIAdapter2   = { 0x0aa1ae0a, 0xfa0e, 0x4b84, { 0x86, 0x44, 0xe0, 0x5f, 0xf8, 0xe5, 0xac, 0xb5 } };
	// static const GUID IID_IDXGIAdapter3   = { 0x645967a4, 0x1392, 0x4310, { 0xa7, 0x98, 0x80, 0x53, 0xce, 0x3e, 0x93, 0xfd } };
	// static const GUID IID_IDXGIAdapter4   = { 0x3c8d99d1, 0x4fbf, 0x4181, { 0xa8, 0x2c, 0xaf, 0x66, 0xbf, 0x7b, 0xd2, 0x4e } };
	// static const GUID IID_IDXGIAdapter    = { 0x2411e7e1, 0x12ac, 0x4ccf, { 0xbd, 0x14, 0x97, 0x98, 0xe8, 0x53, 0x4d, 0xc0 } };
	// static const GUID IID_IDXGIAdapter2   = { 0x0aa1ae0a, 0xfa0e, 0x4b84, { 0x86, 0x44, 0xe0, 0x5f, 0xf8, 0xe5, 0xac, 0xb5 } };
	// static const GUID IID_IDXGIAdapter3   = { 0x645967a4, 0x1392, 0x4310, { 0xa7, 0x98, 0x80, 0x53, 0xce, 0x3e, 0x93, 0xfd } };
	// static const GUID IID_IDXGIAdapter4   = { 0x3c8d99d1, 0x4fbf, 0x4181, { 0xa8, 0x2c, 0xaf, 0x66, 0xbf, 0x7b, 0xd2, 0x4e } };

	// IID_IDXGIOutput, _   = windows.GUIDFromString("{ae02eedb-c735-4690-8d52-5a8dc20213aa}")
	IID_IDXGIOutput1, _  = windows.GUIDFromString("{00cddea8-939b-4b83-a340-a685226666cc}")
	IID_IDXGIOutput5, _  = windows.GUIDFromString("{80A07424-AB52-42EB-833C-0C42FD282D98}")
	IID_IDXGIFactory1, _ = windows.GUIDFromString("{770aae78-f26f-4dba-a829-253c83d1b387}")
	IID_IDXGIFactory2, _ = windows.GUIDFromString("{50c83a1c-e072-4c48-87b0-3630fa36a6d0}")
	IID_IDXGIFactory4, _ = windows.GUIDFromString("{7632e1f5-ee65-4dca-87fd-84cd75f8838d}")
	// static const GUID IID_IDXGIFactory5   = { 0x } };

	// IID_IDXGIDevice3, _ = windows.GUIDFromString("{6007896c-3244-4afd-bf18-a6d3beda5023}")

	// GUID_ENTRY(0x645967a4,0x1392,0x4310,0xa7,0x98,0x80,0x53,0xce,0x3e,0x93,0xfd,IID_IDXGIAdapter3)
	// static const GUID IID_IDXGIFactory    = { 0x7b7166ec, 0x21c7, 0x44ae, { 0xb2, 0x1a, 0xc9, 0xae, 0x32, 0x1a, 0xe3, 0x69 } };
	// static const GUID IID_IDXGIFactory2   = { 0x50c83a1c, 0xe072, 0x4c48, { 0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0 } };
	// static const GUID IID_IDXGIFactory3   = { 0x25483823, 0xcd46, 0x4c7d, { 0x86, 0xca, 0x47, 0xaa, 0x95, 0xb8, 0x37, 0xbd } };
	// static const GUID IID_IDXGIFactory4   = { 0x1bc6ea02, 0xef36, 0x464f, { 0xbf, 0x0c, 0x21, 0xca, 0x39, 0xe5, 0x16, 0x8a } };
	// static const GUID IID_IDXGIDevice0    = { 0x54ec77fa, 0x1377, 0x44e6, { 0x8c, 0x32, 0x88, 0xfd, 0x5f, 0x44, 0xc8, 0x4c } };
	// static const GUID IID_IDXGIDevice1    = { 0x77db970f, 0x6276, 0x48ba, { 0xba, 0x28, 0x07, 0x01, 0x43, 0xb4, 0x39, 0x2c } };
	// static const GUID IID_IDXGIDevice2    = { 0x05008617, 0xfbfd, 0x4051, { 0xa7, 0x90, 0x14, 0x48, 0x84, 0xb4, 0xf6, 0xa9 } };
	// static const GUID IID_IDXGIDevice3    = { 0x6007896c, 0x3244, 0x4afd, { 0xbf, 0x18, 0xa6, 0xd3, 0xbe, 0xda, 0x50, 0x23 } };
	// static const GUID IID_IDXGISwapChain3 = { 0x94d99bdb, 0xf1f8, 0x4ab0, { 0xb2, 0x36, 0x7d, 0xa0, 0x17, 0x0e, 0xda, 0xb1 } };
	// static const GUID IID_IDXGISwapChain4 = { 0x3d585d5a, 0xbd4a, 0x489e, { 0xb1, 0xf4, 0x3d, 0xbc, 0xb6, 0x45, 0x2f, 0xfb } };
	// static const GUID IID_IDXGIOutput6    = { 0x068346e8, 0xaaec, 0x4b84, { 0xad, 0xd7, 0x13, 0x7f, 0x51, 0x3f, 0x77, 0xa1 } };

	// IID_IDXGIResource, _ = windows.GUIDFromString("{035f3ab4-482e-4e50-b41f-8a7f8bd8960b}")
	IID_IDXGISurface, _ = windows.GUIDFromString("{cafcb56c-6ac3-4889-bf47-9e23bbd260ec}")
)

const (
	DXGI_MAP_READ    = 1 << 0
	DXGI_MAP_WRITE   = 1 << 1
	DXGI_MAP_DISCARD = 1 << 2
)

type IDXGIFactory4 struct {
	vtbl *IDXGIFactory4Vtbl
}

func (obj *IDXGIFactory4) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory4) EnumAdapters1(adapter uint32, pp **IDXGIAdapter1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumAdapters1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(adapter),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory4) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

type IDXGIFactory2 struct {
	vtbl *IDXGIFactory2Vtbl
}

func (obj *IDXGIFactory2) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory2) EnumAdapters1(adapter uint32, pp **IDXGIAdapter1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumAdapters1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(adapter),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory2) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

type IDXGIFactory1 struct {
	vtbl *IDXGIFactory1Vtbl
}

func (obj *IDXGIFactory1) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory1) EnumAdapters1(adapter uint32, pp **IDXGIAdapter1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumAdapters1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(adapter),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIFactory1) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func CreateDXGIFactory1(ppFactory **IDXGIFactory1) error {
	ret, _, _ := syscall.SyscallN(
		procCreateDXGIFactory1.Addr(),
		uintptr(unsafe.Pointer(&IID_IDXGIFactory1)),
		uintptr(unsafe.Pointer(ppFactory)),
	)
	if ret != 0 {
		return HRESULT(ret)
	}

	return nil

}

func CreateDXGIFactory2(flags uint32, ppFactory **IDXGIFactory2) error {
	ret, _, _ := syscall.SyscallN(
		procCreateDXGIFactory2.Addr(),
		uintptr(flags),
		uintptr(unsafe.Pointer(&IID_IDXGIFactory2)),
		uintptr(unsafe.Pointer(ppFactory)),
	)
	if ret != 0 {
		return HRESULT(ret)
	}

	return nil

}

// TODO factory 4 might be useful with EnumAdapterByLuid to allow follow-up lookups just for the GPUs we already know about

type IDXGIAdapter4 struct {
	vtbl *IDXGIAdapter4Vtbl
}

func (obj *IDXGIAdapter4) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIAdapter4) EnumOutputs(output uint32, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumOutputs,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}
func (obj *IDXGIAdapter4) GetDesc1(desc *DXGI_ADAPTER_DESC1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}
func (obj *IDXGIAdapter4) GetDesc2(desc *DXGI_ADAPTER_DESC2) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}
func (obj *IDXGIAdapter4) GetDesc3(desc *DXGI_ADAPTER_DESC3) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}
func (obj *IDXGIAdapter4) QueryVideoMemoryInfo(NodeIndex uint32, MemorySegmentGroup DXGI_MEMORY_SEGMENT_GROUP, pVideoMemoryInfo *DXGI_QUERY_VIDEO_MEMORY_INFO) int32 {
	// slog.Debug(fmt.Sprintf("XXX adapter vtbl %+v", obj.vtbl))
	// slog.Debug(fmt.Sprintf("XXX IDXGIAdapter3 %+v", obj))
	// slog.Debug(fmt.Sprintf("XXX NodeIndex %+v", NodeIndex))
	// slog.Debug(fmt.Sprintf("XXX MemorySegmentGroup %+v", MemorySegmentGroup))
	// slog.Debug(fmt.Sprintf("XXX pVideoMemoryInfo %+v", pVideoMemoryInfo))
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.QueryVideoMemoryInfo,
		uintptr(unsafe.Pointer(obj)),
		uintptr(NodeIndex),
		uintptr(MemorySegmentGroup),
		uintptr(unsafe.Pointer(pVideoMemoryInfo)),
	)
	return int32(ret)
}

type IDXGIAdapter3 struct {
	vtbl *IDXGIAdapter3Vtbl
}

func (obj *IDXGIAdapter3) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIAdapter3) EnumOutputs(output uint32, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumOutputs,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}
func (obj *IDXGIAdapter3) GetDesc1(desc *DXGI_ADAPTER_DESC1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}
func (obj *IDXGIAdapter3) GetDesc2(desc *DXGI_ADAPTER_DESC2) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}
func (obj *IDXGIAdapter3) QueryVideoMemoryInfo(NodeIndex uint32, MemorySegmentGroup DXGI_MEMORY_SEGMENT_GROUP, pVideoMemoryInfo *DXGI_QUERY_VIDEO_MEMORY_INFO) int32 {
	// slog.Debug(fmt.Sprintf("XXX adapter vtbl %+v", obj.vtbl))
	// slog.Debug(fmt.Sprintf("XXX IDXGIAdapter3 %+v", obj))
	// slog.Debug(fmt.Sprintf("XXX NodeIndex %+v", NodeIndex))
	// slog.Debug(fmt.Sprintf("XXX MemorySegmentGroup %+v", MemorySegmentGroup))
	// slog.Debug(fmt.Sprintf("XXX pVideoMemoryInfo %+v", pVideoMemoryInfo))
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.QueryVideoMemoryInfo,
		uintptr(unsafe.Pointer(obj)),
		uintptr(NodeIndex),
		uintptr(MemorySegmentGroup),
		uintptr(unsafe.Pointer(pVideoMemoryInfo)),
	)
	return int32(ret)
}

// TODO - maybe redundant?
func (obj *IDXGIAdapter3) AddRef() uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.AddRef,
		uintptr(unsafe.Pointer(obj)),
	)
	return uint32(ret)
}

type IDXGIAdapter2 struct {
	vtbl *IDXGIAdapter2Vtbl
}

func (obj *IDXGIAdapter2) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIAdapter2) EnumOutputs(output uint32, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumOutputs,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}

func (obj *IDXGIAdapter2) GetDesc1(desc *DXGI_ADAPTER_DESC1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}

type IDXGIAdapter1 struct {
	vtbl *IDXGIAdapter1Vtbl
}

func (obj *IDXGIAdapter1) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

func (obj *IDXGIAdapter1) EnumOutputs(output uint32, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumOutputs,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}

func (obj *IDXGIAdapter1) GetDesc1(desc *DXGI_ADAPTER_DESC1) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}

func (obj *IDXGIAdapter1) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

type IDXGIAdapter struct {
	vtbl *IDXGIAdapterVtbl
}

func (obj *IDXGIAdapter) EnumOutputs(output uint32, pp **IDXGIOutput) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.EnumOutputs,
		uintptr(unsafe.Pointer(obj)),
		uintptr(output),
		uintptr(unsafe.Pointer(pp)),
	)
	return uint32(ret)
}

func (obj *IDXGIAdapter) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIDevice struct {
	vtbl *IDXGIDeviceVtbl
}

func (obj *IDXGIDevice) GetGPUThreadPriority(priority *int) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetGPUThreadPriority,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(priority)),
	)
	return int32(ret)
}
func (obj *IDXGIDevice) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGIDevice) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetParent,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}
func (obj *IDXGIDevice) GetAdapter(pAdapter **IDXGIAdapter) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetAdapter,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pAdapter)),
	)
	return int32(ret)
}
func (obj *IDXGIDevice) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIDevice1 struct {
	vtbl *IDXGIDevice1Vtbl
}

func (obj *IDXGIDevice1) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func (obj *IDXGIDevice1) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetParent,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)

	return int32(ret)
}
func (obj *IDXGIDevice1) GetAdapter(pAdapter *IDXGIAdapter) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetAdapter,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&pAdapter)),
	)

	return int32(ret)
}
func (obj *IDXGIDevice1) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIOutput struct {
	vtbl *IDXGIOutputVtbl
}

func (obj *IDXGIOutput) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}

func (obj *IDXGIOutput) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetParent,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIOutput1 struct {
	vtbl *IDXGIOutput1Vtbl
}

func (obj *IDXGIOutput1) DuplicateOutput(device1 *IDXGIDevice1, ppOutputDuplication **IDXGIOutputDuplication) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.DuplicateOutput,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(device1)),
		uintptr(unsafe.Pointer(ppOutputDuplication)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput1) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetParent,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput1) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIOutput5 struct {
	vtbl *IDXGIOutput5Vtbl
}

type DXGI_FORMAT uint32

func (obj *IDXGIOutput5) GetDesc(desc *DXGI_OUTPUT_DESC) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput5) DuplicateOutput1(device1 *IDXGIDevice1, flags uint32, pSupportedFormats []DXGI_FORMAT, ppOutputDuplication **IDXGIOutputDuplication) int32 {
	pFormats := &pSupportedFormats[0]
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.DuplicateOutput1,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(device1)),
		uintptr(flags),
		uintptr(len(pSupportedFormats)),
		uintptr(unsafe.Pointer(pFormats)),
		uintptr(unsafe.Pointer(ppOutputDuplication)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput5) GetParent(iid windows.GUID, pp *unsafe.Pointer) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetParent,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(&iid)),
		uintptr(unsafe.Pointer(pp)),
	)
	return int32(ret)
}

func (obj *IDXGIOutput5) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIResource struct {
	vtbl *IDXGIResourceVtbl
}

func (obj *IDXGIResource) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGIResource) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGISurface struct {
	vtbl *IDXGISurfaceVtbl
}

func (obj *IDXGISurface) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return ReflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *IDXGISurface) Map(pLockedRect *DXGI_MAPPED_RECT, mapFlags uint32) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Map,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pLockedRect)),
		uintptr(mapFlags),
	)
	return int32(ret)
}
func (obj *IDXGISurface) Unmap() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Unmap,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}
func (obj *IDXGISurface) Release() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}

type IDXGIOutputDuplication struct {
	vtbl *IDXGIOutputDuplicationVtbl
}

func (obj *IDXGIOutputDuplication) GetFrameMoveRects(buffer []DXGI_OUTDUPL_MOVE_RECT, rectsRequired *uint32) int32 {
	var buf *DXGI_OUTDUPL_MOVE_RECT
	if len(buffer) > 0 {
		buf = &buffer[0]
	}
	size := uint32(len(buffer) * 24)
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetFrameMoveRects,
		uintptr(unsafe.Pointer(obj)),
		uintptr(size),
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(rectsRequired)),
	)
	*rectsRequired = *rectsRequired / 24
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) GetFrameDirtyRects(buffer []RECT, rectsRequired *uint32) int32 {
	var buf *RECT
	if len(buffer) > 0 {
		buf = &buffer[0]
	}
	size := uint32(len(buffer) * 16)
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetFrameDirtyRects,
		uintptr(unsafe.Pointer(obj)),
		uintptr(size),
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(rectsRequired)),
	)
	*rectsRequired = *rectsRequired / 16
	return int32(ret)
}

func (obj *IDXGIOutputDuplication) GetFramePointerShape(pointerShapeBufferSize uint32,
	pPointerShapeBuffer []byte,
	pPointerShapeBufferSizeRequired *uint32,
	pPointerShapeInfo *DXGI_OUTDUPL_POINTER_SHAPE_INFO) int32 {

	var buf *byte
	if len(pPointerShapeBuffer) > 0 {
		buf = &pPointerShapeBuffer[0]
	}

	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetFramePointerShape,
		uintptr(unsafe.Pointer(obj)),
		uintptr(pointerShapeBufferSize),
		uintptr(unsafe.Pointer(buf)),
		uintptr(unsafe.Pointer(pPointerShapeBufferSizeRequired)),
		uintptr(unsafe.Pointer(pPointerShapeInfo)),
	)

	return int32(ret)
}
func (obj *IDXGIOutputDuplication) GetDesc(desc *DXGI_OUTDUPL_DESC) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.GetDesc,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(desc)),
	)
	return int32(ret)
}

func (obj *IDXGIOutputDuplication) MapDesktopSurface(pLockedRect *DXGI_MAPPED_RECT) int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.MapDesktopSurface,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(pLockedRect)),
	)
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) UnMapDesktopSurface() int32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.UnMapDesktopSurface,
		uintptr(unsafe.Pointer(obj)),
	)
	return int32(ret)
}
func (obj *IDXGIOutputDuplication) AddRef() uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.AddRef,
		uintptr(unsafe.Pointer(obj)),
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) Release() uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.Release,
		uintptr(unsafe.Pointer(obj)),
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) AcquireNextFrame(timeoutMs uint32, pFrameInfo *DXGI_OUTDUPL_FRAME_INFO, ppDesktopResource **IDXGIResource) uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.AcquireNextFrame,    // function address
		uintptr(unsafe.Pointer(obj)), // always pass the COM object address first
		uintptr(timeoutMs),           // then all function parameters follow
		uintptr(unsafe.Pointer(pFrameInfo)),
		uintptr(unsafe.Pointer(ppDesktopResource)),
	)
	return uint32(ret)
}

func (obj *IDXGIOutputDuplication) ReleaseFrame() uint32 {
	ret, _, _ := syscall.SyscallN(
		obj.vtbl.ReleaseFrame,
		uintptr(unsafe.Pointer(obj)),
	)
	return uint32(ret)
}
