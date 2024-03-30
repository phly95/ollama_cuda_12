package gpu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Discovery logic for AMD/ROCm GPUs

const (
	DriverVersionFile     = "/sys/module/amdgpu/version"
	AMDNodesSysfsDir      = "/sys/class/kfd/kfd/topology/nodes/"
	GPUPropertiesFileGlob = AMDNodesSysfsDir + "*/properties"

	// Prefix with the node dir
	GPUTotalMemoryFileGlob = "mem_banks/*/properties" // size_in_bytes line
	GPUUsedMemoryFileGlob  = "mem_banks/*/used_memory"
	RocmStandardLocation   = "/opt/rocm/lib"
)

var (
	// Used to validate if the given ROCm lib is usable
	ROCmLibGlobs = []string{"libhipblas.so.2*", "rocblas"} // TODO - probably include more coverage of files here...
)

// Gather GPU information from the amdgpu driver if any supported GPUs are detected
func AMDGetGPUInfo() []GpuInfo {
	resp := []GpuInfo{}
	if !AMDDetected() {
		return resp
	}

	// Opportunistic logging of driver version to aid in troubleshooting
	ver, err := AMDDriverVersion()
	if err == nil {
		slog.Info("AMD Driver: " + ver)
	} else {
		// TODO - if we see users crash and burn with the upstreamed kernel this can be adjusted to hard-fail rocm support and fallback to CPU
		slog.Warn(fmt.Sprintf("ollama recommends running the https://www.amd.com/en/support/linux-drivers: %s", err))
	}

	// Determine if the user has already pre-selected which GPUs to look at, then ignore the others
	var visibleDevices []string
	hipVD := os.Getenv("HIP_VISIBLE_DEVICES")   // zero based index only
	rocrVD := os.Getenv("ROCR_VISIBLE_DEVICES") // zero based index or UUID, but consumer cards seem to not support UUID
	gpuDO := os.Getenv("GPU_DEVICE_ORDINAL")    // zero based index
	switch {
	// TODO is this priorty order right?
	case hipVD != "":
		visibleDevices = strings.Split(hipVD, ",")
	case rocrVD != "":
		visibleDevices = strings.Split(rocrVD, ",")
		// TODO - since we don't yet support UUIDs, consider detecting and reporting here
		// all our test systems show GPU-XX indicating UUID is not supported
	case gpuDO != "":
		visibleDevices = strings.Split(gpuDO, ",")
	}

	gfxOverride := os.Getenv("HSA_OVERRIDE_GFX_VERSION")
	var supported []string
	libDir := ""

	// The amdgpu driver always exposes the host CPU(s) first, but we have to skip them and subtract
	// from the other IDs to get alignment with the HIP libraries expectations (zero is the first GPU, not the CPU)
	matches, _ := filepath.Glob(GPUPropertiesFileGlob)
	cpuCount := 0
	for _, match := range matches {
		slog.Debug("evaluating amdgpu node " + match)
		fp, err := os.Open(match)
		if err != nil {
			slog.Debug(fmt.Sprintf("failed to open sysfs node file %s: %s", match, err))
			continue
		}
		defer fp.Close()
		nodeID, err := strconv.Atoi(filepath.Base(filepath.Dir(match)))
		if err != nil {
			slog.Debug(fmt.Sprintf("failed to parse node ID %s", err))
			continue
		}

		scanner := bufio.NewScanner(fp)
		isCPU := false
		var major, minor, patch uint64
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Note: we could also use "cpu_cores_count X" where X is greater than zero to detect CPUs
			if strings.HasPrefix(line, "gfx_target_version") {
				ver := strings.Fields(line)

				// Detect CPUs
				if len(ver) == 2 && ver[1] == "0" {
					slog.Debug("detected CPU " + match)
					isCPU = true
					break
				}

				if len(ver) != 2 || len(ver[1]) < 5 {
					slog.Warn("malformed "+match, "gfx_target_version", line)
					// If this winds up being a CPU, our offsets may be wrong
					continue
				}
				l := len(ver[1])
				var err1, err2, err3 error
				patch, err1 = strconv.ParseUint(ver[1][l-2:l], 10, 32)
				minor, err2 = strconv.ParseUint(ver[1][l-4:l-2], 10, 32)
				major, err3 = strconv.ParseUint(ver[1][:l-4], 10, 32)
				if err1 != nil || err2 != nil || err3 != nil {
					slog.Debug("malformed int " + line)
					continue
				}
			}

			// TODO - any other properties we want to extract and record?
			// vendor_id + device_id -> pci lookup for "Name"
			// Other metrics that may help us understand relative performance between multiple GPUs
		}

		if isCPU {
			cpuCount++
			continue
		}

		// CPUs are always first in the list
		gpuID := nodeID - cpuCount

		// Shouldn't happen, but just in case...
		if gpuID < 0 {
			slog.Error("unexpected amdgpu sysfs data resulted in negative GPU ID, please set OLLAMA_DEBUG=1 and report an issue")
			return []GpuInfo{}
		}

		if int(major) < RocmComputeMin {
			slog.Warn(fmt.Sprintf("amdgpu [%d] too old gfx%d%d%d", gpuID, major, minor, patch))
			continue
		}

		// Look up the memory for the current node
		totalMemory := uint64(0)
		usedMemory := uint64(0)
		propGlob := filepath.Join(AMDNodesSysfsDir, strconv.Itoa(nodeID), GPUTotalMemoryFileGlob)
		propFiles, err := filepath.Glob(propGlob)
		if err != nil {
			slog.Warn(fmt.Sprintf("error looking up total GPU memory: %s %s", propGlob, err))
		}
		// 1 or more memory banks - sum the values of all of them
		for _, propFile := range propFiles {
			fp, err := os.Open(propFile)
			if err != nil {
				slog.Warn(fmt.Sprintf("failed to open sysfs node file %s: %s", propFile, err))
				continue
			}
			defer fp.Close()
			scanner := bufio.NewScanner(fp)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "size_in_bytes") {
					ver := strings.Fields(line)
					if len(ver) != 2 {
						slog.Warn("malformed " + line)
						continue
					}
					bankSizeInBytes, err := strconv.ParseUint(ver[1], 10, 64)
					if err != nil {
						slog.Warn("malformed int " + line)
						continue
					}
					totalMemory += bankSizeInBytes
				}
			}
		}
		if totalMemory == 0 {
			slog.Warn(fmt.Sprintf("amdgpu [%d] reports zero total memory", gpuID))
			continue
		}
		usedGlob := filepath.Join(AMDNodesSysfsDir, strconv.Itoa(nodeID), GPUUsedMemoryFileGlob)
		usedFiles, err := filepath.Glob(usedGlob)
		if err != nil {
			slog.Warn(fmt.Sprintf("error looking up used GPU memory: %s %s", usedGlob, err))
			continue
		}
		for _, usedFile := range usedFiles {
			fp, err := os.Open(usedFile)
			if err != nil {
				slog.Warn(fmt.Sprintf("failed to open sysfs node file %s: %s", usedFile, err))
				continue
			}
			defer fp.Close()
			data, err := io.ReadAll(fp)
			if err != nil {
				slog.Warn(fmt.Sprintf("failed to read sysfs node file %s: %s", usedFile, err))
				continue
			}
			used, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
			if err != nil {
				slog.Warn(fmt.Sprintf("malformed used memory %s: %s", string(data), err))
				continue
			}
			usedMemory += used
		}

		// iGPU detection, remove this check once we can support an iGPU variant of the rocm library
		if totalMemory < IGPUMemLimit {
			slog.Info(fmt.Sprintf("amdgpu [%d] appears to be an iGPU with %dM reported total memory, skipping", gpuID, totalMemory/1024/1024))
			continue
		}

		slog.Info(fmt.Sprintf("[%d] amdgpu totalMemory %dM", gpuID, totalMemory/1024/1024))
		slog.Info(fmt.Sprintf("[%d] amdgpu freeMemory  %dM", gpuID, (totalMemory-usedMemory)/1024/1024))
		gpuInfo := GpuInfo{
			Library: "rocm",
			memInfo: memInfo{
				TotalMemory: totalMemory,
				FreeMemory:  (totalMemory - usedMemory),
			},
			ID: fmt.Sprintf("%d", gpuID),
			// Name: not exposed in sysfs directly, would require pci device id lookup
			Major:         int(major),
			Minor:         int(minor),
			Patch:         int(patch),
			MinimumMemory: rocmMinimumMemory,
		}

		// If the user wants to filter to a subset of devices, filter out if we aren't a match
		if len(visibleDevices) > 0 {
			include := false
			for _, visible := range visibleDevices {
				if visible == gpuInfo.ID {
					include = true
					break
				}
			}
			if !include {
				slog.Info("filtering out device per user request", "id", gpuInfo.ID, "visible_devices", visibleDevices)
				continue
			}
		}

		// Final validation is gfx compatibility - load the library if we haven't already loaded it
		// even if the user overrides, we still need to validate the library
		if libDir == "" {
			libDir, err = AMDValidateLibDir()
			if err != nil {
				slog.Warn(fmt.Sprintf("unable to verify rocm library, will use cpu: %s", err))
				return []GpuInfo{}
			}
		}
		gpuInfo.DependencyPath = libDir

		if gfxOverride == "" {
			// Only load supported list once
			if len(supported) == 0 {
				supported, err = GetSupportedGFX(libDir)
				if err != nil {
					slog.Warn(fmt.Sprintf("failed to lookup supported GFX types, falling back to CPU mode: %s", err))
					return []GpuInfo{}
				}
				slog.Debug(fmt.Sprintf("rocm supported GPU types %v", supported))
			}
			gfx := fmt.Sprintf("gfx%d%d%d", gpuInfo.Major, gpuInfo.Minor, gpuInfo.Patch)
			if !slices.Contains[[]string, string](supported, gfx) {
				slog.Warn(fmt.Sprintf("[%s] amdgpu %s is not supported by %s %v", gpuInfo.ID, gfx, libDir, supported))
				// TODO - consider discrete markdown just for ROCM troubleshooting?
				slog.Warn("See https://github.com/ollama/ollama/blob/main/docs/gpu.md#overrides for HSA_OVERRIDE_GFX_VERSION usage")
				continue
			} else {
				slog.Info(fmt.Sprintf("amdgpu [%s] %s is supported", gpuInfo.ID, gfx))
			}
		} else {
			slog.Debug("skipping rocm gfx compatibility check with HSA_OVERRIDE_GFX_VERSION=" + gfxOverride)
		}

		// The GPU has passed all the verification steps and is supported
		resp = append(resp, gpuInfo)
	}
	if len(resp) == 0 {
		slog.Info("no compatible amdgpu devices detected")
	}
	return resp
}

// func updateLibPath(libDir string) {
// 	ldPaths := []string{}
// 	if val, ok := os.LookupEnv("LD_LIBRARY_PATH"); ok {
// 		ldPaths = strings.Split(val, ":")
// 	}
// 	for _, d := range ldPaths {
// 		if d == libDir {
// 			return
// 		}
// 	}
// 	val := strings.Join(append(ldPaths, libDir), ":")
// 	slog.Debug("updated lib path", "LD_LIBRARY_PATH", val)
// 	os.Setenv("LD_LIBRARY_PATH", val)
// }

// Quick check for AMD driver so we can skip amdgpu discovery if not present
func AMDDetected() bool {
	// Some driver versions (older?) don't have a version file, so just lookup the parent dir
	sysfsDir := filepath.Dir(DriverVersionFile)
	_, err := os.Stat(sysfsDir)
	if errors.Is(err, os.ErrNotExist) {
		slog.Debug("amdgpu driver not detected " + sysfsDir)
		return false
	} else if err != nil {
		slog.Debug(fmt.Sprintf("error looking up amd driver %s %s", sysfsDir, err))
		return false
	}
	return true
}

// Prefer to use host installed ROCm, as long as it meets our minimum requirements
// failing that, tell the user how to download it on their own
func AMDValidateLibDir() (string, error) {
	libDir, err := commonAMDValidateLibDir()
	if err == nil {
		return libDir, nil
	}

	// Well known ollama installer path
	installedRocmDir := "/usr/share/ollama/lib/rocm"
	if rocmLibUsable(installedRocmDir) {
		return installedRocmDir, nil
	}

	// If we still haven't found a usable rocm, the user will have to install it on their own
	slog.Warn("amdgpu detected, but no compatible rocm library found.  Either install rocm v6, or follow manual install instructions at https://github.com/ollama/ollama/blob/main/docs/linux.md#manual-install")
	return "", fmt.Errorf("no suitable rocm found, falling back to CPU")
}

func AMDDriverVersion() (string, error) {
	_, err := os.Stat(DriverVersionFile)
	if err != nil {
		return "", fmt.Errorf("amdgpu version file missing: %s %w", DriverVersionFile, err)
	}
	fp, err := os.Open(DriverVersionFile)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	verString, err := io.ReadAll(fp)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(verString)), nil
}
