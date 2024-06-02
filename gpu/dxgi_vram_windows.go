package gpu

import (
	"fmt"
	"log/slog"

	"github.com/ollama/ollama/format"
	"golang.org/x/sys/windows"
)

func GetFreeVRAM() error {
	var factory1 *IDXGIFactory1
	var factory4 *IDXGIFactory4
	slog.Debug("XXX creating DXGI Factory1")

	if err := CreateDXGIFactory1(&factory1); err != nil {
		return fmt.Errorf("CreateDXGIFactory1: %w", err)
	}
	defer factory1.Release()

	if hr := factory1.QueryInterface(IID_IDXGIFactory4, &factory4); HRESULT(hr).Failed() {
		return fmt.Errorf("XXX failed to convert factory2->factory4 %w", HRESULT(hr))
	}

	// TODO - is there another way to do this that doesn't involve iterating until we break? (e.g. get the actual number of adapters?)

	for i := uint32(0); ; i++ {
		var adapter1 *IDXGIAdapter1
		// var adapter3 *IDXGIAdapter3
		var adapter4 *IDXGIAdapter4
		if hr := factory4.EnumAdapters1(i, &adapter1); HRESULT(hr).Failed() {
			if HRESULT(hr) == DXGI_ERROR_NOT_FOUND {
				// expected once we've enumerated all adapters
				break
			}
			// else unexpected error
			return fmt.Errorf("failed to enumerate adapter %d %w", i, HRESULT(hr))
		}
		defer adapter1.Release()

		var desc1 DXGI_ADAPTER_DESC1
		if hr := adapter1.GetDesc1(&desc1); HRESULT(hr).Failed() {
			return fmt.Errorf("failed to get adapter description. %w", HRESULT(hr))
		}
		slog.Debug("XXX Adapter1", "index", i, "description", windows.UTF16ToString(desc1.Description[:]), "vendor", desc1.VendorId, "device", desc1.DeviceId, "revision", desc1.Revision, "subsystem", desc1.SubSysId, "video memory", format.HumanBytes2(desc1.DedicatedVideoMemorySize), "system memory", format.HumanBytes2(desc1.DedicatedSystemMemorySize), "shared system memory", format.HumanBytes2(desc1.SharedSystemMemorySize), "luid", fmt.Sprintf("%02x%02x%02x%02x", desc1.AdapterLuid[0], desc1.AdapterLuid[1], desc1.AdapterLuid[2], desc1.AdapterLuid[3]))

		// slog.Debug("XXX raw", "desc1", desc1)
		if desc1.Flags == 0 {
			// if hr := adapter1.QueryInterface(IID_IDXGIAdapter3, &adapter3); HRESULT(hr).Failed() {
			if hr := adapter1.QueryInterface(IID_IDXGIAdapter4, &adapter4); HRESULT(hr).Failed() {
				slog.Error("XXX failed to convert", "error", HRESULT(hr))
				continue
			}

			if adapter4 == nil {
				slog.Warn("XXX unable to convert adapter1 -> adapter4")
				continue
			}
			// TODO  -this didn't help
			// if hr := adapter3.AddRef(); HRESULT(hr).Failed() {
			// 	slog.Warn("XXX failed to call AddRef on adapter3", "error", HRESULT(hr))
			// }

			defer adapter4.Release()
			// slog.Debug("XXX Adapter3", "index", i, "addapter", adapter3.vtbl)

			var desc3 DXGI_ADAPTER_DESC3
			if hr := adapter4.GetDesc3(&desc3); HRESULT(hr).Failed() {
				return fmt.Errorf("failed to get adapter description v3. %w", HRESULT(hr))
			}
			slog.Debug("XXX Adapter3", "index", i, "description", windows.UTF16ToString(desc3.Description[:]), "vendor", desc3.VendorId, "device", desc3.DeviceId, "revision", desc3.Revision, "subsystem", desc3.SubSysId, "video memory", desc3.DedicatedVideoMemorySize, "system memory", desc3.DedicatedSystemMemorySize, "shared system memory", desc3.SharedSystemMemorySize, "luid", fmt.Sprintf("%02x%02x%02x%02x", desc3.AdapterLuid[0], desc3.AdapterLuid[1], desc3.AdapterLuid[2], desc3.AdapterLuid[3]), "compute_preemption", desc3.ComputePreemptionGranularity, "graphics_preemption", desc3.GraphicsPreemptionGranularity)

			var memInfo DXGI_QUERY_VIDEO_MEMORY_INFO
			if hr := adapter4.QueryVideoMemoryInfo(0, DXGI_MEMORY_SEGMENT_GROUP_LOCAL, &memInfo); HRESULT(hr).Failed() {
				slog.Warn("XXX QueryVideoMemoryInfo failed", "error", HRESULT(hr))
			} else {
				slog.Debug("XXX memInfo", "index", i, "budget", format.HumanBytes2(memInfo.Budget), "current usage", format.HumanBytes2(memInfo.CurrentUsage), "available for reservation", format.HumanBytes2(memInfo.AvailableForReservation), "current reservation", format.HumanBytes2(memInfo.CurrentReservation))
			}
		} else {
			slog.Debug("XXX flags were nonzero")
		}
	}
	return nil
}
