package gpu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TODO - windows vs. non-windows vs darwin

// Discovery logic for AMD/ROCm GPUs

/*
Straw-man algo
- Detect if amdgpu driver is present, if not don't bother going further for AMD support
- If driver present, then get gfx version of card(s)
- Try to live off the land if we see rocmv6 present
	- ROCM_HOME (?)
	- /opt/rocm/lib
	- check for compatibility and abort with good error message if not supported by installed rocm lib
- If no v6 detected on host
	- Check for presence in libDir
	- if not found, download from well known location on our release page
	- if that fails, abort AMD support with good error and fallback to CPU mode


TODO - test old drivers with this approach and make sure rocmv6 doesn't blow up on a supported GPU with old driver
*/

const (
	DriverVersionFile     = "/sys/module/amdgpu/version"
	GPUPropertiesFileGlob = "/sys/class/kfd/kfd/topology/nodes/*/properties"
	// TODO probably break these down per GPU to make the logic simpler
	GPUTotalMemoryFileGlob = "/sys/class/kfd/kfd/topology/nodes/*/mem_banks/*/properties" // size_in_bytes line
	GPUUsedMemoryFileGlob  = "/sys/class/kfd/kfd/topology/nodes/*/mem_banks/*/used_memory"
)

func AMDDetected() bool {
	// Some driver versions (older?) don't have a version file, so just lookup the parent dir
	sysfsDir := filepath.Dir(DriverVersionFile)
	_, err := os.Stat(sysfsDir)
	if errors.Is(err, os.ErrNotExist) {
		slog.Debug("amd driver not detected " + sysfsDir)
		return false
	} else if err != nil {
		slog.Debug(fmt.Sprintf("error looking up amd driver %s %s", sysfsDir, err))
		return false
	}
	return true
}

func AMDGetLibDir() string {
	// Prefer to use host installed ROCm, as long as it meets our minimum requirements

	// TODO

	// HIP_PATH + lib/
	// LD_LIBRARY_PATH
	// /opt/rocm/lib

	// If the host version isn't compatible or not found, then load into our libdir
	libDir, err := LibDir()
	if err != nil {
		slog.Warn("unable to lookup lib dir: %s", err)
		return ""
	}

	// TODO

	// Stat the rocm payloads in the lib dir, if not found, download it
	// Try our current version, then try latest release
	// if we can't download, report a good error to guide the user what to do.

	return libDir

}

func AMDDriverVersion() (string, error) {
	_, err := os.Stat(DriverVersionFile)
	if err != nil {
		return "", fmt.Errorf("amdgpu file stat error: %s %w", DriverVersionFile, err)
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

func AMDGFXVersions() []Version {
	res := []Version{}
	matches, _ := filepath.Glob(GPUPropertiesFileGlob)
	for _, match := range matches {
		fp, err := os.Open(match)
		if err != nil {
			slog.Debug(fmt.Sprintf("failed to open sysfs node file %s: %s", match, err))
			continue
		}
		defer fp.Close()

		scanner := bufio.NewScanner(fp)
		// optionally, resize scanner's capacity for lines over 64K, see next example
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "gfx_target_version") {
				ver := strings.Fields(line)
				if len(ver) != 2 || len(ver[1]) < 5 {
					slog.Debug("malformed " + line)
					continue
				}
				l := len(ver[1])
				patch, err1 := strconv.ParseUint(ver[1][l-2:l], 10, 32)
				minor, err2 := strconv.ParseUint(ver[1][l-4:l-2], 10, 32)
				major, err3 := strconv.ParseUint(ver[1][:l-4], 10, 32)
				if err1 != nil || err2 != nil || err3 != nil {
					slog.Debug("malformed int " + line)
					continue
				}

				res = append(res, Version{
					Major: uint(major),
					Minor: uint(minor),
					Patch: uint(patch),
				})
			}
		}
	}
	return res
}

func (v Version) ToGFXString() string {
	return fmt.Sprintf("gfx%d%d%d", v.Major, v.Minor, v.Patch)
}

func GetSupportedGFX(libDir string) ([]string, error) {
	var ret []string
	files, err := filepath.Glob(filepath.Join(libDir, "rocblas", "library", "TensileLibrary_lazy_gfx*.dat"))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		ret = append(ret, strings.TrimSuffix(strings.TrimPrefix(filepath.Base(file), "TensileLibrary_lazy_"), ".dat"))
	}
	return ret, nil
}
