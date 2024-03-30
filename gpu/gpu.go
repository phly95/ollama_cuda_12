//go:build linux || windows

package gpu

/*
#cgo linux LDFLAGS: -lrt -lpthread -ldl -lstdc++ -lm
#cgo windows LDFLAGS: -lpthread

#include "gpu_info.h"

*/
import "C"
import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/ollama/ollama/format"
)

type handles struct {
	num_devices int
	cudart      *C.cudart_handle_t
}

const (
	cudaMinimumMemory = 457 * format.MebiByte
	rocmMinimumMemory = 457 * format.MebiByte
)

var gpuMutex sync.Mutex

// With our current CUDA compile flags, older than 5.0 will not work properly
var CudaComputeMin = [2]C.int{5, 0}

var RocmComputeMin = 9

// TODO find a better way to detect iGPU instead of minimum memory
const IGPUMemLimit = 1024 * 1024 * 1024 // 512G is what they typically report, so anything less than 1G must be iGPU

var CudartLinuxGlobs = []string{
	"/usr/local/cuda/lib64/libcudart.so*",
	"/usr/lib/x86_64-linux-gnu/nvidia/current/libcudart.so*",
	"/usr/lib/x86_64-linux-gnu/libcudart.so*",
	"/usr/lib/wsl/lib/libcudart.so*",
	"/usr/lib/wsl/drivers/*/libcudart.so*",
	"/opt/cuda/lib64/libcudart.so*",
	"/usr/local/cuda*/targets/aarch64-linux/lib/libcudart.so*",
	"/usr/lib/aarch64-linux-gnu/nvidia/current/libcudart.so*",
	"/usr/lib/aarch64-linux-gnu/libcudart.so*",
	"/usr/local/cuda/lib*/libcudart.so*",
	"/usr/lib*/libcudart.so*",
	"/usr/local/lib*/libcudart.so*",
}

var CudartWindowsGlobs = []string{
	"c:\\Program Files\\NVIDIA GPU Computing Toolkit\\CUDA\\v*\\bin\\cudart64_*.dll",
}

// Jetson devices have JETSON_JETPACK="x.y.z" factory set to the Jetpack version installed.
// Included to drive logic for reducing Ollama-allocated overhead on L4T/Jetson devices.
var CudaTegra string = os.Getenv("JETSON_JETPACK")

// Note: gpuMutex must already be held
func initGPUHandles() *handles {

	// TODO - if the ollama build is CPU only, don't do these checks as they're irrelevant and confusing

	gpuHandles := &handles{}
	var cudartMgmtName string
	var cudartMgmtPatterns []string

	tmpDir, _ := PayloadsDir()
	switch runtime.GOOS {
	case "windows":
		cudartMgmtName = "cudart64_*.dll"
		localAppData := os.Getenv("LOCALAPPDATA")
		cudartMgmtPatterns = []string{filepath.Join(localAppData, "Programs", "Ollama", cudartMgmtName)}
		cudartMgmtPatterns = append(cudartMgmtPatterns, CudartWindowsGlobs...)
	case "linux":
		cudartMgmtName = "libcudart.so*"
		if tmpDir != "" {
			// TODO - add "payloads" for subprocess
			cudartMgmtPatterns = []string{filepath.Join(tmpDir, "cuda*", cudartMgmtName)}
		}
		cudartMgmtPatterns = append(cudartMgmtPatterns, CudartLinuxGlobs...)
	default:
		return gpuHandles
	}

	slog.Info("Detecting GPUs")
	cudartLibPaths := FindGPULibs(cudartMgmtName, cudartMgmtPatterns)
	if len(cudartLibPaths) > 0 {
		num_devices, cudart, libPath := LoadCUDARTMgmt(cudartLibPaths)
		if cudart != nil {
			slog.Info(fmt.Sprintf("%s reports %d GPUs present", libPath, num_devices))
			gpuHandles.cudart = cudart
			gpuHandles.num_devices = num_devices
			return gpuHandles
		}
	}
	return gpuHandles
}

func GetGPUInfo() GpuInfoList {
	// TODO - consider exploring lspci (and equivalent on windows) to check for
	// GPUs so we can report warnings if we see Nvidia/AMD but fail to load the libraries
	gpuMutex.Lock()
	defer gpuMutex.Unlock()

	gpuHandles := initGPUHandles()
	defer func() {
		if gpuHandles.cudart != nil {
			C.cudart_release(*gpuHandles.cudart)
		}
	}()

	// All our GPU builds on x86 have AVX enabled, so fallback to CPU if we don't detect at least AVX
	cpuVariant := GetCPUVariant()
	if cpuVariant == "" && runtime.GOARCH == "amd64" {
		slog.Warn("CPU does not have AVX or AVX2, disabling GPU support.")
	}

	var memInfo C.mem_info_t
	resp := []GpuInfo{}

	// NVIDIA first
	for i := 0; i < gpuHandles.num_devices; i++ {
		// TODO once we support CPU compilation variants of GPU libraries refine this...
		if cpuVariant == "" && runtime.GOARCH == "amd64" {
			continue
		}
		gpuInfo := GpuInfo{
			Library: "cuda",
		}
		C.cudart_check_vram(*gpuHandles.cudart, C.int(i), &memInfo)
		if memInfo.err != nil {
			slog.Info(fmt.Sprintf("error looking up nvidia GPU memory: %s", C.GoString(memInfo.err)))
			C.free(unsafe.Pointer(memInfo.err))
			continue
		}
		if memInfo.major < CudaComputeMin[0] || (memInfo.major == CudaComputeMin[0] && memInfo.minor < CudaComputeMin[1]) {
			slog.Info(fmt.Sprintf("[%d] CUDA GPU is too old. Compute Capability detected: %d.%d", i, memInfo.major, memInfo.minor))
			continue
		}
		gpuInfo.TotalMemory = uint64(memInfo.total)
		gpuInfo.FreeMemory = uint64(memInfo.free)
		gpuInfo.ID = C.GoString(&memInfo.gpu_id[0])
		gpuInfo.Major = int(memInfo.major)
		gpuInfo.Minor = int(memInfo.minor)
		gpuInfo.MinimumMemory = cudaMinimumMemory

		// TODO potentially sort on our own algorithm instead of what the underlying GPU library does...
		resp = append(resp, gpuInfo)
	}

	// Then AMD
	resp = append(resp, AMDGetGPUInfo()...)

	if len(resp) == 0 {
		C.cpu_check_ram(&memInfo)
		if memInfo.err != nil {
			slog.Info(fmt.Sprintf("error looking up CPU memory: %s", C.GoString(memInfo.err)))
			C.free(unsafe.Pointer(memInfo.err))
			return resp
		}
		gpuInfo := GpuInfo{
			Library: "cpu",
			Variant: cpuVariant,
		}
		gpuInfo.TotalMemory = uint64(memInfo.total)
		gpuInfo.FreeMemory = uint64(memInfo.free)
		gpuInfo.ID = C.GoString(&memInfo.gpu_id[0])

		resp = append(resp, gpuInfo)
	}

	return resp
}

func getCPUMem() (memInfo, error) {
	var ret memInfo
	var info C.mem_info_t
	C.cpu_check_ram(&info)
	if info.err != nil {
		defer C.free(unsafe.Pointer(info.err))
		return ret, fmt.Errorf(C.GoString(info.err))
	}
	ret.FreeMemory = uint64(info.free)
	ret.TotalMemory = uint64(info.total)
	return ret, nil
}

// TODO clean up call sites to this routine so we don't do back-to-back calls any more
func CheckVRAM() (int64, error) {
	userLimit := os.Getenv("OLLAMA_MAX_VRAM")
	if userLimit != "" {
		avail, err := strconv.ParseInt(userLimit, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Invalid OLLAMA_MAX_VRAM setting %s: %s", userLimit, err)
		}
		slog.Info(fmt.Sprintf("user override OLLAMA_MAX_VRAM=%d", avail))
		return avail, nil
	}
	// TODO - this wrong, quick hack to get things compiling...
	gpuInfo := GetGPUInfo()
	if gpuInfo[0].FreeMemory > 0 && (gpuInfo[0].Library == "cuda" || gpuInfo[0].Library == "rocm") {
		return int64(gpuInfo[0].FreeMemory), nil
	}

	return 0, fmt.Errorf("no GPU detected") // TODO - better handling of CPU based memory determiniation
}

func FindGPULibs(baseLibName string, patterns []string) []string {
	// Multiple GPU libraries may exist, and some may not work, so keep trying until we exhaust them
	var ldPaths []string
	gpuLibPaths := []string{}
	slog.Debug(fmt.Sprintf("Searching for GPU management library %s", baseLibName))

	switch runtime.GOOS {
	case "windows":
		ldPaths = strings.Split(os.Getenv("PATH"), ";")
	case "linux":
		ldPaths = strings.Split(os.Getenv("LD_LIBRARY_PATH"), ":")
	default:
		return gpuLibPaths
	}
	// Start with whatever we find in the PATH/LD_LIBRARY_PATH
	for _, ldPath := range ldPaths {
		d, err := filepath.Abs(ldPath)
		if err != nil {
			continue
		}
		patterns = append(patterns, filepath.Join(d, baseLibName+"*"))
	}
	slog.Debug(fmt.Sprintf("gpu library search paths: %v", patterns))
	for _, pattern := range patterns {
		// Ignore glob discovery errors
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			// Resolve any links so we don't try the same lib multiple times
			// and weed out any dups across globs
			libPath := match
			tmp := match
			var err error
			for ; err == nil; tmp, err = os.Readlink(libPath) {
				if !filepath.IsAbs(tmp) {
					tmp = filepath.Join(filepath.Dir(libPath), tmp)
				}
				libPath = tmp
			}
			new := true
			for _, cmp := range gpuLibPaths {
				if cmp == libPath {
					new = false
					break
				}
			}
			if new {
				gpuLibPaths = append(gpuLibPaths, libPath)
			}
		}
	}
	slog.Debug(fmt.Sprintf("discovered GPU libraries: %v", gpuLibPaths))
	return gpuLibPaths
}

func LoadCUDARTMgmt(cudartLibPaths []string) (int, *C.cudart_handle_t, string) {
	var resp C.cudart_init_resp_t
	resp.ch.verbose = getVerboseState()
	for _, libPath := range cudartLibPaths {
		lib := C.CString(libPath)
		defer C.free(unsafe.Pointer(lib))
		C.cudart_init(lib, &resp)
		if resp.err != nil {
			slog.Debug("Unable to load cudart", "library", libPath, "error", C.GoString(resp.err))
			C.free(unsafe.Pointer(resp.err))
		} else {
			return int(resp.num_devices), &resp.ch, libPath
		}
	}
	return 0, nil, ""
}

func getVerboseState() C.uint16_t {
	if debug := os.Getenv("OLLAMA_DEBUG"); debug != "" {
		return C.uint16_t(1)
	}
	return C.uint16_t(0)
}

// Given the list of GPUs this instantiation is targeted for,
// figure out the visible devices environment variable
//
// If different libraries are detected, the first one is what we use
// TODO consider mixed a bug here?
func (l GpuInfoList) GetVisibleDevicesEnv() (string, string) {
	if len(l) == 0 {
		return "", ""
	}
	switch l[0].Library {
	case "cuda":
		return cudaGetVisibleDevicesEnv(l)
	case "rocm":
		return rocmGetVisibleDevicesEnv(l)
	default:
		slog.Debug("no filter required for library " + l[0].Library)
		return "", ""
	}
}
