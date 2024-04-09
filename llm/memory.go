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
	var layers int
	if opts.NumCtx > int(ggml.KV().ContextLength()) {
		slog.Warn("requested context length is greater than model max context length", "requested", opts.NumCtx, "model", ggml.KV().ContextLength())
		opts.NumCtx = int(ggml.KV().ContextLength())
	}

	if opts.NumCtx < 4 {
		opts.NumCtx = 4
	}

	// Split up the GPUs by type and try them
	for _, gpus := range allGpus.ByLibrary() {
		layers, estimatedVRAM = PredictGPULayers(gpus, ggml, projectors, opts)
		if layers >= int(ggml.KV().BlockCount()) { // TODO handle user specified lower layer count
			return true, estimatedVRAM, nil
		}
	}
	return false, estimatedVRAM, nil
}

// Given a model and one or more GPU targets, predict how many layers and bytes we can load
func PredictGPULayers(gpus []gpu.GpuInfo, ggml *GGML, projectors []string, opts api.Options) (int, uint64) {
	availableMemory := int64(0)
	for _, info := range gpus {
		availableMemory += int64(info.FreeMemory)
	}
	slog.Debug("evaluating", "library", gpus[0].Library, "gpu_count", len(gpus), "availableMB", availableMemory/1024/1024)

	// TODO - this is probably wrong, first GPU vs secondaries will have different overheads
	usedMemory := gpus[0].MinimumMemory

	for _, projector := range projectors {
		usedMemory += projectorMemoryRequirements(projector)

		// multimodal models require at least 2048 context
		opts.NumCtx = max(opts.NumCtx, 2048)
	}

	// fp16 k,v = (1 (k) + 1 (v)) * sizeof(float16) * n_ctx * n_layer * n_embd / n_head * n_head_kv
	kv := 2 * 2 * int64(opts.NumCtx) * int64(ggml.KV().BlockCount()) * int64(ggml.KV().EmbeddingLength()) / int64(ggml.KV().HeadCount()) * int64(ggml.KV().HeadCountKV())

	graph, ok := ggml.GraphSize(opts.NumCtx, min(opts.NumCtx, opts.NumBatch))
	if !ok {
		graph = int64(ggml.KV().GQA()) * kv / 6
	}

	usedMemory += graph

	if ggml.IsCPUOnlyModel() {
		slog.Debug("CPU only model")
		return 0, 0
	}
	if usedMemory > availableMemory {
		slog.Debug("insufficient VRAM to load any model layers")
		return 0, 0
	}

	requiredMemory := usedMemory

	var layers int
	for i := 0; i < int(ggml.KV().BlockCount()); i++ {
		layerMemory := ggml.LayerSize(fmt.Sprintf("blk.%d.", i)) + kv/int64(ggml.KV().BlockCount())
		requiredMemory += layerMemory

		if availableMemory > usedMemory+layerMemory && (opts.NumGPU < 0 || layers < opts.NumGPU) {
			usedMemory += layerMemory
			layers++
		}
	}

	memOutputLayer := ggml.LayerSize("output.")
	requiredMemory += memOutputLayer

	// only offload output layer if all repeating layers are offloaded
	if layers >= int(ggml.KV().BlockCount()) && availableMemory > usedMemory+memOutputLayer {
		usedMemory += memOutputLayer
		layers++
	}

	slog.Info(
		"offload to gpu",
		"layers", layers,
		"required", format.HumanBytes2(requiredMemory),
		"used", format.HumanBytes2(usedMemory),
		"available", format.HumanBytes2(availableMemory),
		"kv", format.HumanBytes2(kv),
		"graph", format.HumanBytes2(graph),
	)
	return layers, uint64(usedMemory)
}
