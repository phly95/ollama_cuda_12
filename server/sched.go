package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/gpu"
	"github.com/ollama/ollama/llm"
	"golang.org/x/exp/slices"
)

type runnerRef struct {
	refMu     sync.Mutex
	refCond   sync.Cond // Signaled on transition from 1 -> 0 refCount
	refCount  uint      // prevent unloading if > 0
	unloading bool      // set to true when we are trying to unload the runner

	llama         *llm.LlamaServer
	gpus          gpu.GpuInfoList // Recorded at time of provisioning
	estimatedVRAM uint64

	sessionDuration time.Duration
	expireTimer     *time.Timer

	model      string
	adapters   []string
	projectors []string
	*api.Options
}

func (runner *runnerRef) Use() error {
	runner.refMu.Lock()
	defer runner.refMu.Unlock()

	// Safeguard in case runner has been unloaded while we waited for the refMu lock
	if runner.llama == nil {
		slog.Info("request rejected after model was unloading")
		return fmt.Errorf("model was unloaded to make room for another model")
	}

	if runner.unloading {
		slog.Info("request rejected while model is unloading")
		return fmt.Errorf("model is being unloaded")
	}

	// Note: LlamaServer implements limits and blocks requests until there's an available slot
	// so we can allow many requests to be in flight concurrently at this layer
	runner.refCount++

	return nil
}

func (runner *runnerRef) Release() {
	runner.refMu.Lock()
	defer runner.refMu.Unlock()
	runner.refCount--
	if runner.refCount == 0 {
		runner.refCond.Signal()
	}
}

var loadedMu sync.Mutex
var loaded = map[string]*runnerRef{}
var loadedMax = 0 // Maximum runners; < 1 maps to as many as will fit in VRAM (unlimited for CPU runners)

type ByDuration []*runnerRef

func (a ByDuration) Len() int      { return len(a) }
func (a ByDuration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDuration) Less(i, j int) bool {
	// uint64 to turn negative time (never unload) to largest
	return uint64(a[i].sessionDuration) < uint64(a[j].sessionDuration)
}

// TODO - future consideration to pick runners based on size
// type BySize []*runnerRef
// func (a BySize) Len() int           { return len(a) }
// func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
// func (a BySize) Less(i, j int) bool { return a[i].estimatedVRAM < a[j].estimatedVRAM }

// Only allow the getRunner func to run one at a time
var getRunnerMu sync.Mutex

// getRunner idempotently loads a runner for the given model and options.
// If the same model is loaded with other options, the runner will be unloaded (once idle)
// and replaced with a new runner and the new options.
// If one or more other models are loaded, this routine will determine if the new runner
// can be loaded and fit fully in VRAM.  If it can not, other runners will be unloaded
// until there is enough room, or this new model becomes the only runner.
func getRunner(c *gin.Context, model *Model, opts api.Options, sessionDuration time.Duration) (*runnerRef, error) {

	// TODO debug logging is a bit verbose, but should help us troubleshoot any glitches
	// Once things are looking solid, quiet down the logs
	slog.Debug("getRunner called", "model", model.ModelPath)
	getRunnerMu.Lock()
	defer getRunnerMu.Unlock()
	slog.Debug("getRunner processing", "model", model.ModelPath)

	maxRunners := os.Getenv("OLLAMA_MAX_RUNNERS")
	if maxRunners != "" {
		m, err := strconv.Atoi(maxRunners)
		if err != nil {
			slog.Error("invalid setting", "OLLAMA_MAX_RUNNERS", maxRunners, "error", err)
		} else {
			loadedMax = m
		}
	}
	loadedMu.Lock()
	runner := loaded[model.ModelPath]
	loadedMu.Unlock()

	var gpus gpu.GpuInfoList
	var ggml *llm.GGML

	if runner != nil {
		slog.Debug("evaluating already loaded", "model", model.ModelPath)
		runner.refMu.Lock()
		defer runner.refMu.Unlock()
		// Ignore the NumGPU settings for comparison
		optsExisting := runner.Options.Runner
		optsExisting.NumGPU = -1
		optsNew := opts.Runner
		optsNew.NumGPU = -1
		ctx, cancel := context.WithTimeout(c, 10*time.Second) // BUG - 2 rapid requests to the same model that is taking a long time to load probably fails this test
		defer cancel()
		if !reflect.DeepEqual(runner.adapters, model.AdapterPaths) || // have the adapters changed?
			!reflect.DeepEqual(runner.projectors, model.ProjectorPaths) || // have the projectors changed?
			!reflect.DeepEqual(optsExisting, optsNew) || // have the runner options changed?
			runner.llama.Ping(ctx) != nil {
			if runner.refCount > 0 {
				slog.Debug("runner reload required, but active requests in flight...")
				runner.unloading = true
				runner.refCond.Wait()
			}
			slog.Info("changing loaded model to update settings", "model", model.ModelPath)
			runner.llama.Close()
			runner.llama = nil
			runner.adapters = nil
			runner.projectors = nil
			runner.Options = nil
			runner = nil
		}
	}
	if runner == nil {
		slog.Debug("needs load", "model", model.ModelPath)
		var err error
		var estimatedVRAM uint64
		ggml, err = llm.LoadModel(model.ModelPath)
		if err != nil {
			return nil, err
		}
		loadedMu.Lock()
	fit:
		for len(loaded) > 0 {
			slog.Debug("attempting to fit new model with existing models running", "runner_count", len(loaded))

			allGpus := gpu.GetGPUInfo()
			if (len(allGpus) == 1 && allGpus[0].Library == "cpu") || opts.NumGPU == 0 {

				// TODO evaluate system memory to see if we should block the load, or force an unload of another CPU runner

				slog.Debug("CPU mode, allowing multiple model loads")
				gpus = allGpus // No-op, will load on CPU anyway
				break
			}

			if len(loaded) >= loadedMax {
				slog.Debug("max runners achieved, unloading one to make room", "runner_count", len(loaded))
				unloadBestFitRunner()
				continue
			}

			// Memory usage of loaded runners ramps up as they load the model,
			// even after NewLlamaServer returns and can fluctuate over time so
			// we can't trust the snapshot from GPUInfo as a true metric for
			// available space. We'll instead rely on our predictions of the size
			// of the other models.
			// Note: we can't even get free VRAM metrics for metal or rocm+windows

			// TODO we should expose some user supported way to reserve a buffer per GPU and add that in here

			type predKey struct {
				Library string
				ID      string
			}
			predMap := map[predKey]uint64{} // Sum up the total predicted usage per GPU for all runners
			for _, r := range loaded {
				r.refMu.Lock()
				gpuIDs := make([]string, 0, len(r.gpus))
				if r.llama != nil {

					// TODO this should be broken down by GPU instead of assuming uniform spread
					estimatedVRAMPerGPU := r.llama.EstimatedVRAM / uint64(len(r.gpus))
					for _, gpu := range r.gpus {
						gpuIDs = append(gpuIDs, gpu.ID)
					}
					for _, gpu := range allGpus {
						if slices.Contains(gpuIDs, gpu.ID) {
							predMap[predKey{gpu.Library, gpu.ID}] += estimatedVRAMPerGPU
						}
					}
				} else {
					slog.Warn("unexpected nil runner reference, memory prediction may be incorrect")
				}
				r.refMu.Unlock()
			}

			// Now that we've summed up all the GPU usage predictions across all the loaded runners, update the gpu list
			for i := range allGpus {
				if p, ok := predMap[predKey{allGpus[i].Library, allGpus[i].ID}]; ok {
					slog.Debug(fmt.Sprintf("[%s] %s reported freeMemory %dM", allGpus[i].ID, allGpus[i].Library, allGpus[i].FreeMemory/1024/1024))
					if p > allGpus[i].TotalMemory {
						// Shouldn't happen
						slog.Warn("predicted usage exceeds VRAM", "ID", allGpus[i].ID, "totalMemory", allGpus[i].TotalMemory, "predicted", p)
						allGpus[i].FreeMemory = 0
					} else if (allGpus[i].TotalMemory - p) < allGpus[i].FreeMemory { // predicted free is smaller than reported free, use it
						// TODO maybe we should just always trust our numbers, since cuda's free memory reporting is laggy
						// and we might unload models we didn't actually need to.  The risk is if some other GPU intensive app is loaded
						// after we start our first runner, then we'll never acount for that, so picking the smallest free value seems prudent.
						allGpus[i].FreeMemory = allGpus[i].TotalMemory - p
					}
					slog.Info(fmt.Sprintf("[%s] updated %s totalMemory %dM", allGpus[i].ID, allGpus[i].Library, allGpus[i].TotalMemory/1024/1024))
					slog.Info(fmt.Sprintf("[%s] updated %s freeMemory  %dM", allGpus[i].ID, allGpus[i].Library, allGpus[i].FreeMemory/1024/1024))
				}
			}

			for _, gl := range allGpus.ByLibrary() {
				var ok bool
				// First attempt to fit the model into a single GPU
				for _, g := range gl {
					if ok, estimatedVRAM, _ = llm.PredictServerFit([]gpu.GpuInfo{g}, ggml, model.AdapterPaths, model.ProjectorPaths, opts); ok {
						slog.Debug("new model will fit in available VRAM in single GPU, loading", "model", model.ModelPath, "gpu", g.ID, "requiredM", estimatedVRAM/1024/1024)
						gpus = []gpu.GpuInfo{g}
						break fit
					}
				}
				// Now try all the GPUs
				if ok, estimatedVRAM, _ = llm.PredictServerFit(gl, ggml, model.AdapterPaths, model.ProjectorPaths, opts); ok {
					slog.Debug("new model will fit in available VRAM, loading", "model", model.ModelPath, "library", gl[0].Library, "requiredM", estimatedVRAM/1024/1024)
					gpus = gl
					break fit
				}
			}
			slog.Debug("new model will not fit in available VRAM without unloading another model", "model", model.ModelPath, "requiredM", estimatedVRAM/1024/1024)
			gpus = nil // force a refresh of the gpu info after we unload a model.

			// If we get to here, then we have GPUs, and can't fit the new model in VRAM
			// Find the best candidate model to unload
			unloadBestFitRunner()

			// For CUDA, free memory reporting is laggy and will lead us astray
			time.Sleep(100 * time.Millisecond)
		}
		slog.Debug("made room for new runner", "model", model.ModelPath)

		runner = &runnerRef{}
		runner.refCond.L = &runner.refMu
		runner.refMu.Lock()
		defer runner.refMu.Unlock()
		loaded[model.ModelPath] = runner
		slog.Info("loaded runners", "count", len(loaded))
		loadedMu.Unlock()

		if len(gpus) == 0 {
			slog.Debug("refreshing GPU info", "model", model.ModelPath)
			// Given we have no other models loaded, just narrow to the first Library
			gpus = gpu.GetGPUInfo().ByLibrary()[0]
		}

		llama, err := llm.NewLlamaServer(gpus, model.ModelPath, ggml, model.AdapterPaths, model.ProjectorPaths, opts)
		if err != nil {
			// some older models are not compatible with newer versions of llama.cpp
			// show a generalized compatibility error until there is a better way to
			// check for model compatibility
			if errors.Is(llm.ErrUnsupportedFormat, err) || strings.Contains(err.Error(), "failed to load model") {
				err = fmt.Errorf("%v: this model may be incompatible with your version of Ollama. If you previously pulled this model, try updating it by running `ollama pull %s`", err, model.ShortName)
			}
			loadedMu.Lock()
			defer loadedMu.Unlock()
			delete(loaded, runner.model)
			slog.Info("NewLlamaServer failed", "model", model.ModelPath, "error", err)
			return nil, err
		}

		runner.model = model.ModelPath
		runner.adapters = model.AdapterPaths
		runner.projectors = model.ProjectorPaths
		runner.llama = llama
		runner.Options = &opts
		runner.refCount = 0
		runner.sessionDuration = sessionDuration
		runner.gpus = gpus
		runner.estimatedVRAM = llama.EstimatedVRAM
		slog.Debug("finished setting up runner", "model", model.ModelPath)
	}

	if runner.expireTimer == nil {
		runner.expireTimer = time.AfterFunc(sessionDuration, func() {
			slog.Debug("timer expired to unload", "model", model.ModelPath)
			runner.refMu.Lock()
			defer runner.refMu.Unlock()
			if runner.refCount > 0 {
				// Typically this shouldn't happen with the timer reset, unless the timer
				// is very short, or the system is very slow
				slog.Debug("runner expireTimer fired while requests still in flight, waiting for completion")
				runner.unloading = true
				runner.refCond.Wait()
			}
			slog.Debug("got lock to unload", "model", model.ModelPath)

			if runner.llama != nil {
				runner.llama.Close()
			}

			runner.llama = nil
			runner.adapters = nil
			runner.projectors = nil
			runner.Options = nil
			runner.gpus = nil
			loadedMu.Lock()
			defer loadedMu.Unlock()
			delete(loaded, runner.model)
			slog.Debug("runner released", "model", runner.model)
		})
	}

	runner.expireTimer.Reset(sessionDuration)
	return runner, nil
}

// unloadBestFitRunner finds a runner to unload to make room for a new model
// Caller must already have loadedMu lock
// This will block until all pending requests to the given runner have drained
func unloadBestFitRunner() {
	runnerList := make([]*runnerRef, 0, len(loaded))
	for _, r := range loaded {
		runnerList = append(runnerList, r)
	}

	// In the future we can enhance the algorithm to be smarter about picking the optimal runner to unload
	sort.Sort(ByDuration(runnerList))

	var r *runnerRef
	// First try to find a runner that's already idle
	for _, runner := range runnerList {
		runner.refMu.Lock()
		if runner.refCount == 0 {
			defer runner.refMu.Unlock()
			r = runner
			break
		}
		runner.refMu.Unlock()
	}
	if r == nil {
		// None appear idle, just wait for the one with the shortest duration
		r = runnerList[0]
		r.refMu.Lock()
		defer r.refMu.Unlock()
		if r.refCount > 0 {
			slog.Info("waiting for pending requests to drain before unloading", "old_model", r.model)
			r.unloading = true
			r.refCond.Wait()
		}
	}

	slog.Info("unloading old model before its timeout to make room for new model", "old_model", r.model)

	if r.expireTimer != nil {
		r.expireTimer.Stop()
	}

	if r.llama != nil {
		r.llama.Close()
	}
	r.llama = nil
	r.adapters = nil
	r.projectors = nil
	r.Options = nil
	r.gpus = nil
	delete(loaded, r.model)
}

func unloadAllRunners() {
	loadedMu.Lock()
	defer loadedMu.Unlock()
	for model, runner := range loaded {
		if runner.llama != nil {
			slog.Debug("shutting down runner", "model", model)
			runner.llama.Close()
		}
	}
}
