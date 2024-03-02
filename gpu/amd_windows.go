package gpu

/*
#cgo linux LDFLAGS: -lrt -lpthread -ldl -lstdc++ -lm
#cgo windows LDFLAGS: -lpthread

#include "gpu_info.h"

*/
import "C"

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unsafe"

	"github.com/jmorganca/ollama/version"
)

const (
	// TODO - powershell translation
	curlMsg              = "curl -fsSL https://github.com/ollama/ollama/releases/download/v%s/rocm_v6-x86_64-deps.tgz | tar -zxf - -C %s"
	RocmStandardLocation = "C:\\Program Files\\AMD\\ROCm\\5.7\\bin" // TODO glob?
)

var (
	// Used to validate if the given ROCm lib is usable
	ROCmLibGlobs = []string{"hipblas.dll", "rocblas"} // TODO - probably include more coverage of files here...
)

func AMDGetGPUInfo(resp *GpuInfo) {
	hl, err := NewHipLib()
	if err != nil {
		slog.Debug(err.Error())
		return
	}
	defer hl.Release()
	skip := map[int]interface{}{}
	ids := []int{}
	resp.memInfo.DeviceCount = 0
	resp.memInfo.TotalMemory = 0
	resp.memInfo.FreeMemory = 0

	ver, err := hl.AMDDriverVersion()
	if err == nil {
		slog.Info("AMD Driver: " + ver)
	} else {
		// For now this is benign, but we may eventually need to fail compatibility checks
		slog.Debug(fmt.Sprintf("error looking up amd driver version: %s", err))
	}

	count := hl.HipGetDeviceCount()
	if count == 0 {
		return
	}
	libDir, err := AMDValidateLibDir()
	if err != nil {
		slog.Warn(fmt.Sprintf("unable to verify rocm library, will use cpu: %s", err))
		return
	}

	var supported []string
	gfxOverride := os.Getenv("HSA_OVERRIDE_GFX_VERSION")
	if gfxOverride == "" {
		supported, err = GetSupportedGFX(libDir)
		if err != nil {
			slog.Warn(fmt.Sprintf("failed to lookup supported GFX types, falling back to CPU mode: %s", err))
			return
		}
	} else {
		slog.Debug("skipping rocm gfx compatibility check with HSA_OVERRIDE_GFX_VERSION=" + gfxOverride)
	}

	slog.Info(fmt.Sprintf("detected %d hip devices", count))
	for i := 0; i < count; i++ {
		ids = append(ids, i)
		err = hl.HipSetDevice(i)
		if err != nil {
			slog.Warn(fmt.Sprint("[%d] %s", i, err))
			skip[i] = struct{}{}
			continue
		}

		props, err := hl.HipGetDeviceProperties(i)
		if err != nil {
			slog.Warn(fmt.Sprint("[%d] %s", i, err))
			skip[i] = struct{}{}
			continue
		}
		n := bytes.IndexByte(props.Name[:], 0)
		name := string(props.Name[:n])
		slog.Info(fmt.Sprintf("[%d] Name: %s", i, name))
		n = bytes.IndexByte(props.GcnArchName[:], 0)
		gfx := string(props.GcnArchName[:n])
		slog.Info(fmt.Sprintf("[%d] GcnArchName: %s", i, gfx))
		//slog.Info(fmt.Sprintf("[%d] Integrated: %d", i, props.iGPU)) // DOESN'T REPORT CORRECTLY!  Always 0
		// HACK!!!  Why isn't props.iGPU accurate!?
		if strings.EqualFold(name, iGPUName) {
			slog.Info(fmt.Sprintf("iGPU detected [%d] skipping", i))
			skip[i] = struct{}{}
			continue
		}
		if gfxOverride == "" {
			if !slices.Contains[[]string, string](supported, gfx) {
				slog.Warn(fmt.Sprintf("amdgpu [%d] %s is not supported by %s %v", i, gfx, libDir, supported))
				// TODO - consider discrete markdown just for ROCM troubleshooting?
				slog.Warn("See https://github.com/ollama/ollama/blob/main/docs/troubleshooting.md for HSA_OVERRIDE_GFX_VERSION usage")
				skip[i] = struct{}{}
				continue
			} else {
				slog.Debug(fmt.Sprintf("amdgpu [%d] %s is supported", i, gfx))
			}
		}

		totalMemory, freeMemory, err := hl.HipMemGetInfo()
		if err != nil {
			slog.Warn(fmt.Sprintf("[%d] %s", i, err))
			continue
		}

		// TODO according to docs, freeMem may lie on windows!
		slog.Info(fmt.Sprintf("[%d] Total Mem: %d", i, totalMemory))
		slog.Info(fmt.Sprintf("[%d] Free Mem:  %d", i, freeMemory))
		resp.memInfo.DeviceCount++
		resp.memInfo.TotalMemory += totalMemory
		resp.memInfo.FreeMemory += freeMemory
	}
	if resp.memInfo.DeviceCount > 0 {
		resp.Library = "rocm"
	}
	// Abort if all GPUs are skipped
	if len(skip) >= count {
		slog.Info("all detected amdgpus are skipped, falling back to CPU")
		return
	}
	slog.Debug(fmt.Sprintf("XXX ids: %v", ids))
	slog.Debug(fmt.Sprintf("XXX skip: %v", skip))
	if len(skip) > 0 {
		amdSetVisibleDevices(ids, skip)
	}
	UpdatePath(libDir)
}

// TODO this likely needs more work
func AMDValidateLibDir() (string, error) {

	// If we already have a rocm dependency wired, nothing more to do
	libDir, err := LibDir()
	if err != nil {
		return "", fmt.Errorf("unable to lookup lib dir: %w", err)
	}
	rocmTargetDir := filepath.Join(libDir, "rocm")
	if rocmLibUsable(rocmTargetDir) {
		return rocmTargetDir, nil
	}

	// Prefer explicit HIP env var
	hipPath := os.Getenv("HIP_PATH")
	if hipPath != "" {
		hipLibDir := filepath.Join(hipPath, "bin")
		if rocmLibUsable(hipLibDir) {
			slog.Debug("detected ROCM via HIP_PATH=" + hipPath)
			return hipLibDir, nil
		}
	}

	// Well known location(s)
	if rocmLibUsable(RocmStandardLocation) {
		return RocmStandardLocation, nil
	}

	slog.Warn("amdgpu detected, but no compatible rocm library found.  Either install rocm, or run the following")
	slog.Warn(fmt.Sprintf(curlMsg, version.Version, rocmTargetDir))
	return "", fmt.Errorf("no suitable rocm found, falling back to CPU")
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
