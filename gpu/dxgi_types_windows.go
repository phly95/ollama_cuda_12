package gpu

// TODO - insert MIT copyright and link to github.com/kirides/go-d3d

// TODO remove the dependency here....
//  go:generate stringer -type=_DXGI_OUTDUPL_POINTER_SHAPE_TYPE -output=dxgi_types_string_windows.go

type DXGI_RATIONAL struct {
	Numerator   uint32
	Denominator uint32
}

type DXGI_MODE_ROTATION uint32

type DXGI_OUTPUT_DESC struct {
	DeviceName         [32]uint16
	DesktopCoordinates RECT
	AttachedToDesktop  uint32 // BOOL
	Rotation           DXGI_MODE_ROTATION
	Monitor            uintptr
}

type DXGI_ADAPTER_DESC1 struct {
	Description               [128]uint16
	VendorId                  uint32
	DeviceId                  uint32
	SubSysId                  uint32
	Revision                  uint32
	DedicatedVideoMemorySize  uint64
	DedicatedSystemMemorySize uint64
	SharedSystemMemorySize    uint64
	AdapterLuid               [4]byte // LUID
	Flags                     DXGI_ADAPTER_FLAG
}

type DXGI_ADAPTER_DESC2 struct {
	Description                   [128]uint16
	VendorId                      uint32
	DeviceId                      uint32
	SubSysId                      uint32
	Revision                      uint32
	DedicatedVideoMemorySize      uint64
	DedicatedSystemMemorySize     uint64
	SharedSystemMemorySize        uint64
	AdapterLuid                   [4]byte // LUID
	Flags                         DXGI_ADAPTER_FLAG
	GraphicsPreemptionGranularity DXGI_GRAPHICS_PREEMPTION_GRANULARITY
	ComputePreemptionGranularity  DXGI_COMPUTE_PREEMPTION_GRANULARITY
}

type DXGI_ADAPTER_DESC3 struct {
	Description                   [128]uint16
	VendorId                      uint32
	DeviceId                      uint32
	SubSysId                      uint32
	Revision                      uint32
	DedicatedVideoMemorySize      uint64
	DedicatedSystemMemorySize     uint64
	SharedSystemMemorySize        uint64
	AdapterLuid                   [4]byte // LUID
	Flags                         DXGI_ADAPTER_FLAG3
	GraphicsPreemptionGranularity DXGI_GRAPHICS_PREEMPTION_GRANULARITY
	ComputePreemptionGranularity  DXGI_COMPUTE_PREEMPTION_GRANULARITY
}

type DXGI_GRAPHICS_PREEMPTION_GRANULARITY uint32

const (
	DXGI_GRAPHICS_PREEMPTION_DMA_BUFFER_BOUNDARY  = DXGI_GRAPHICS_PREEMPTION_GRANULARITY(0)
	DXGI_GRAPHICS_PREEMPTION_PRIMITIVE_BOUNDARY   = 1
	DXGI_GRAPHICS_PREEMPTION_TRIANGLE_BOUNDARY    = 2
	DXGI_GRAPHICS_PREEMPTION_PIXEL_BOUNDARY       = 3
	DXGI_GRAPHICS_PREEMPTION_INSTRUCTION_BOUNDARY = 4
)

type DXGI_COMPUTE_PREEMPTION_GRANULARITY uint32

const (
	DXGI_COMPUTE_PREEMPTION_DMA_BUFFER_BOUNDARY   = DXGI_COMPUTE_PREEMPTION_GRANULARITY(0)
	DXGI_COMPUTE_PREEMPTION_DISPATCH_BOUNDARY     = 1
	DXGI_COMPUTE_PREEMPTION_THREAD_GROUP_BOUNDARY = 2
	DXGI_COMPUTE_PREEMPTION_THREAD_BOUNDARY       = 3
	DXGI_COMPUTE_PREEMPTION_INSTRUCTION_BOUNDARY  = 4
)

type DXGI_ADAPTER_FLAG uint32

const (
	DXGI_ADAPTER_FLAG_NONE        = DXGI_ADAPTER_FLAG(0)
	DXGI_ADAPTER_FLAG_REMOTE      = 1
	DXGI_ADAPTER_FLAG_SOFTWARE    = 2
	DXGI_ADAPTER_FLAG_FORCE_DWORD = 0xffffffff
)

type DXGI_ADAPTER_FLAG3 uint32

const (
	DXGI_ADAPTER_FLAG3_NONE                         = DXGI_ADAPTER_FLAG3(0)
	DXGI_ADAPTER_FLAG3_REMOTE                       = 1
	DXGI_ADAPTER_FLAG3_SOFTWARE                     = 2
	DXGI_ADAPTER_FLAG3_ACG_COMPATIBLE               = 4
	DXGI_ADAPTER_FLAG3_SUPPORT_MONITORED_FENCES     = 8
	DXGI_ADAPTER_FLAG3_SUPPORT_NON_MONITORED_FENCES = 0x10
	DXGI_ADAPTER_FLAG3_KEYED_MUTEX_CONFORMANCE      = 0x20
	DXGI_ADAPTER_FLAG3_FORCE_DWORD                  = 0xffffffff
)

type DXGI_MEMORY_SEGMENT_GROUP uint32

const (
	DXGI_MEMORY_SEGMENT_GROUP_LOCAL     = DXGI_MEMORY_SEGMENT_GROUP(0)
	DXGI_MEMORY_SEGMENT_GROUP_NON_LOCAL = 1
)

type DXGI_QUERY_VIDEO_MEMORY_INFO struct {
	Budget                  uint64
	CurrentUsage            uint64
	AvailableForReservation uint64
	CurrentReservation      uint64
}

type DXGI_MODE_DESC struct {
	Width            uint32
	Height           uint32
	Rational         DXGI_RATIONAL
	Format           uint32 // DXGI_FORMAT
	ScanlineOrdering uint32 // DXGI_MODE_SCANLINE_ORDER
	Scaling          uint32 // DXGI_MODE_SCALING
}

type DXGI_OUTDUPL_DESC struct {
	ModeDesc                   DXGI_MODE_DESC
	Rotation                   uint32 // DXGI_MODE_ROTATION
	DesktopImageInSystemMemory uint32 // BOOL
}

type DXGI_SAMPLE_DESC struct {
	Count   uint32
	Quality uint32
}

type POINT struct {
	X int32
	Y int32
}
type RECT struct {
	Left, Top, Right, Bottom int32
}

type DXGI_OUTDUPL_MOVE_RECT struct {
	Src  POINT
	Dest RECT
}
type DXGI_OUTDUPL_POINTER_POSITION struct {
	Position POINT
	Visible  uint32
}
type DXGI_OUTDUPL_FRAME_INFO struct {
	LastPresentTime           int64
	LastMouseUpdateTime       int64
	AccumulatedFrames         uint32
	RectsCoalesced            uint32
	ProtectedContentMaskedOut uint32
	PointerPosition           DXGI_OUTDUPL_POINTER_POSITION
	TotalMetadataBufferSize   uint32
	PointerShapeBufferSize    uint32
}
type DXGI_MAPPED_RECT struct {
	Pitch int32
	PBits uintptr
}

const (
	DXGI_FORMAT_R8G8B8A8_UNORM DXGI_FORMAT = 28
	DXGI_FORMAT_B8G8R8A8_UNORM DXGI_FORMAT = 87
)

type DXGI_OUTDUPL_POINTER_SHAPE_TYPE uint32

const (
	DXGI_OUTDUPL_POINTER_SHAPE_TYPE_MONOCHROME   DXGI_OUTDUPL_POINTER_SHAPE_TYPE = 1
	DXGI_OUTDUPL_POINTER_SHAPE_TYPE_COLOR        DXGI_OUTDUPL_POINTER_SHAPE_TYPE = 2
	DXGI_OUTDUPL_POINTER_SHAPE_TYPE_MASKED_COLOR DXGI_OUTDUPL_POINTER_SHAPE_TYPE = 4
)

type DXGI_OUTDUPL_POINTER_SHAPE_INFO struct {
	Type    DXGI_OUTDUPL_POINTER_SHAPE_TYPE
	Width   uint32
	Height  uint32
	Pitch   uint32
	HotSpot POINT
}
