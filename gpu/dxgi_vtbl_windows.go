package gpu

type IDXGIObjectVtbl struct {
	IUnknownVtbl

	SetPrivateData          uintptr
	SetPrivateDataInterface uintptr
	GetPrivateData          uintptr
	GetParent               uintptr
}

type IDXGIAdapterVtbl struct {
	IDXGIObjectVtbl

	EnumOutputs           uintptr
	GetDesc               uintptr
	CheckInterfaceSupport uintptr
}
type IDXGIAdapter1Vtbl struct {
	IDXGIAdapterVtbl

	GetDesc1 uintptr
}

type IDXGIAdapter2Vtbl struct {
	IDXGIAdapter1Vtbl

	GetDesc2 uintptr
}

type IDXGIAdapter3Vtbl struct {
	IDXGIAdapter2Vtbl

	QueryVideoMemoryInfo                                 uintptr
	RegisterHardwareContentProtectionTeardownStatusEvent uintptr
	RegisterVideoMemoryBudgetChangeNotificationEvent     uintptr
	SetVideoMemoryReservation                            uintptr
	UnregisterHardwareContentProtectionTeardownStatus    uintptr
	UnregisterVideoMemoryBudgetChangeNotification        uintptr
}

// TODO - this didn't help, can probably remove
type IDXGIAdapter4Vtbl struct {
	IDXGIAdapter3Vtbl

	GetDesc3 uintptr
}

type IDXGIDeviceVtbl struct {
	IDXGIObjectVtbl

	CreateSurface          uintptr
	GetAdapter             uintptr
	GetGPUThreadPriority   uintptr
	QueryResourceResidency uintptr
	SetGPUThreadPriority   uintptr
}

type IDXGIDevice1Vtbl struct {
	IDXGIDeviceVtbl

	GetMaximumFrameLatency uintptr
	SetMaximumFrameLatency uintptr
}

type IDXGIDeviceSubObjectVtbl struct {
	IDXGIObjectVtbl

	GetDevice uintptr
}

type IDXGISurfaceVtbl struct {
	IDXGIDeviceSubObjectVtbl

	GetDesc uintptr
	Map     uintptr
	Unmap   uintptr
}

type IDXGIResourceVtbl struct {
	IDXGIDeviceSubObjectVtbl

	GetSharedHandle     uintptr
	GetUsage            uintptr
	SetEvictionPriority uintptr
	GetEvictionPriority uintptr
}

type IDXGIOutputVtbl struct {
	IDXGIObjectVtbl

	GetDesc                     uintptr
	GetDisplayModeList          uintptr
	FindClosestMatchingMode     uintptr
	WaitForVBlank               uintptr
	TakeOwnership               uintptr
	ReleaseOwnership            uintptr
	GetGammaControlCapabilities uintptr
	SetGammaControl             uintptr
	GetGammaControl             uintptr
	SetDisplaySurface           uintptr
	GetDisplaySurfaceData       uintptr
	GetFrameStatistics          uintptr
}

type IDXGIOutput1Vtbl struct {
	IDXGIOutputVtbl

	GetDisplayModeList1      uintptr
	FindClosestMatchingMode1 uintptr
	GetDisplaySurfaceData1   uintptr
	DuplicateOutput          uintptr
}

type IDXGIOutput2Vtbl struct {
	IDXGIOutput1Vtbl

	SupportsOverlays uintptr
}

type IDXGIOutput3Vtbl struct {
	IDXGIOutput2Vtbl

	CheckOverlaySupport uintptr
}

type IDXGIOutput4Vtbl struct {
	IDXGIOutput3Vtbl

	CheckOverlayColorSpaceSupport uintptr
}
type IDXGIOutput5Vtbl struct {
	IDXGIOutput4Vtbl

	DuplicateOutput1 uintptr
}

type IDXGIOutputDuplicationVtbl struct {
	IDXGIObjectVtbl

	GetDesc              uintptr
	AcquireNextFrame     uintptr
	GetFrameDirtyRects   uintptr
	GetFrameMoveRects    uintptr
	GetFramePointerShape uintptr
	MapDesktopSurface    uintptr
	UnMapDesktopSurface  uintptr
	ReleaseFrame         uintptr
}
type IDXGIFactoryVtbl struct {
	IDXGIObjectVtbl

	EnumAdapters          uintptr
	MakeWindowAssociation uintptr
	GetWindowAssociation  uintptr
	CreateSwapChain       uintptr
	CreateSoftwareAdapter uintptr
}
type IDXGIFactory1Vtbl struct {
	IDXGIFactoryVtbl

	EnumAdapters1 uintptr
	IsCurrent     uintptr
}
type IDXGIFactory2Vtbl struct {
	IDXGIFactory1Vtbl

	CreateSwapChainForComposition uintptr
	CreateSwapChainForCoreWindow  uintptr
	CreateSwapChainForHwnd        uintptr
	GetSharedResourceAdapterLuid  uintptr
	IsWindowedStereoEnabled       uintptr
	RegisterOcclusionStatusEvent  uintptr
	RegisterOcclusionStatusWindow uintptr
	RegisterStereoStatusEvent     uintptr
	RegisterStereoStatusWindow    uintptr
	UnregisterOcclusionStatus     uintptr
	UnregisterStereoStatus        uintptr
}
type IDXGIFactory3Vtbl struct {
	IDXGIFactory2Vtbl

	GetCreationFlags uintptr
}

type IDXGIFactory4Vtbl struct {
	IDXGIFactory3Vtbl

	EnumAdapterByLuid uintptr
	EnumWarpAdapter   uintptr
}
