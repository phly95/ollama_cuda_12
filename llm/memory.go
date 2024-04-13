package llm

import (
	"fmt"
	"log/slog"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/format"
	"github.com/ollama/ollama/gpu"
)

// This algorithm looks for a complete fit to determine if we need to unload other models
func PredictServerFit(allGpus gpu.GpuInfoList, ggml *GGML, adapters, projectors []string, opts api.Options) (bool, uint64, error) {
	var estimatedVRAM uint64
	if opts.NumCtx > int(ggml.KV().ContextLength()) {
		slog.Warn("requested context length is greater than model max context length", "requested", opts.NumCtx, "model", ggml.KV().ContextLength())
		opts.NumCtx = int(ggml.KV().ContextLength())
	}

	if opts.NumCtx < 4 {
		opts.NumCtx = 4
	}

	// Split up the GPUs by type and try them
	for _, gpus := range allGpus.ByLibrary() {
		var allLayers bool
		_, estimatedVRAM, allLayers = PredictGPULayers(gpus, ggml, projectors, opts)
		if allLayers {
			return true, estimatedVRAM, nil
		}
	}
	return false, estimatedVRAM, nil
}

// Given a model and one or more GPU targets, predict how many layers and bytes we can load, and true
// if this satisfies opts.NumGPU
// The GPUs provided must all be the same Library
func PredictGPULayers(gpus []gpu.GpuInfo, ggml *GGML, projectors []string, opts api.Options) (int, uint64, bool) {
	if gpus[0].Library == "cpu" {
		return 0, 0, false
	}
	memoryAvailable := uint64(0)
	for _, info := range gpus {
		memoryAvailable += info.FreeMemory
	}
	slog.Debug("evaluating", "library", gpus[0].Library, "gpu_count", len(gpus), "availableMB", memoryAvailable/1024/1024)

	// TODO - this is probably wrong, first GPU vs secondaries will have different overheads
	memoryMinimum := gpus[0].MinimumMemory

	for _, projector := range projectors {
		memoryMinimum += projectorMemoryRequirements(projector)

		// multimodal models require at least 2048 context
		opts.NumCtx = max(opts.NumCtx, 2048)
	}

	// fp16 k,v = (1 (k) + 1 (v)) * sizeof(float16) * n_ctx * n_layer * n_embd / n_head * n_head_kv
	var kv uint64 = 2 * 2 * uint64(opts.NumCtx) * ggml.KV().BlockCount() * ggml.KV().EmbeddingLength() / ggml.KV().HeadCount() * ggml.KV().HeadCountKV()

	graphPartialOffload, graphFullOffload := ggml.GraphSize(uint64(opts.NumCtx), uint64(min(opts.NumCtx, opts.NumBatch)))
	if graphPartialOffload == 0 {
		graphPartialOffload = ggml.KV().GQA() * kv / 6
	}

	if graphFullOffload == 0 {
		graphFullOffload = graphPartialOffload
	}

	// memoryRequiredTotal represents the memory required for full GPU offloading (all layers)
	memoryRequiredTotal := memoryMinimum + graphFullOffload

	// memoryRequiredPartial represents the memory required for partial GPU offloading (n > 0, n < layers)
	memoryRequiredPartial := memoryMinimum + graphPartialOffload

	if gpus[0].Library != "metal" {
		if memoryRequiredPartial > memoryAvailable {
			slog.Debug("insufficient VRAM to load any model layers")
			return 0, 0, false
		}
	}

	var layerCount int
	layers := ggml.Tensors().Layers()
	for i := 0; i < int(ggml.KV().BlockCount()); i++ {
		memoryLayer := layers[fmt.Sprintf("%d", i)].size()

		// KV is proportional to the number of layers
		memoryLayer += kv / ggml.KV().BlockCount()

		memoryRequiredTotal += memoryLayer
		if memoryAvailable > memoryRequiredPartial+memoryLayer {
			memoryRequiredPartial += memoryLayer
			layerCount++
		}
	}

	memoryLayerOutput := layers["output"].size()
	memoryRequiredTotal += memoryLayerOutput
	if memoryAvailable > memoryRequiredTotal {
		layerCount = int(ggml.KV().BlockCount()) + 1
		memoryRequiredPartial = memoryRequiredTotal
	}

	var allLayers bool
	if opts.NumGPU < 0 {
		opts.NumGPU = layerCount
		allLayers = layerCount >= int(ggml.KV().BlockCount()+1)
	} else {
		allLayers = layerCount >= opts.NumGPU
	}

	slog.Info(
		"offload to gpu",
		"reallayers", opts.NumGPU,
		"layers", layerCount,
		"required", format.HumanBytes2(memoryRequiredTotal),
		"used", format.HumanBytes2(memoryRequiredPartial),
		"available", format.HumanBytes2(memoryAvailable),
		"kv", format.HumanBytes2(kv),
		"fulloffload", format.HumanBytes2(graphFullOffload),
		"partialoffload", format.HumanBytes2(graphPartialOffload),
	)
	return layerCount, uint64(memoryRequiredTotal), allLayers
}
