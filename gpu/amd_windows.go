package gpu

import (
	"fmt"
	"log/slog"
	"unsafe"
)

func AMDGetGPUInfo(resp *GpuInfo) {
	// TODO implement
}

// TODO - might be useful on windows stil...
func rocmMemLookup(memInfo *C.mem_info_t, skip map[int]interface{}) {
	C.rocm_check_vram(*gpuHandles.rocm, memInfo)
	if memInfo.err != nil {
		slog.Info(fmt.Sprintf("error looking up ROCm GPU memory: %s", C.GoString(memInfo.err)))
		C.free(unsafe.Pointer(memInfo.err))
	} else if memInfo.igpu_index >= 0 && memInfo.count == 1 {
		// Only one GPU detected and it appears to be an integrated GPU - skip it
		slog.Info("ROCm unsupported integrated GPU detected")
	} else if memInfo.count > 0 {
		if memInfo.igpu_index >= 0 {
			// We have multiple GPUs reported, and one of them is an integrated GPU
			// so we have to set the env var to bypass it
			for i := 0; i < int(memInfo.count); i++ {
				if i == int(memInfo.igpu_index) {
					slog.Info("ROCm unsupported integrated GPU detected")
					skip[i] = struct{}{}
				}
			}
		}
	}
}
