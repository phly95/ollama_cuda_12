package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
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
	refMu    sync.Mutex
	refCond  sync.Cond
	refCount uint

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
	// Note: LlamaServer implements limits and blocks requests until there's an available slot
	runner.refCount++

	// Safeguard in case runner has been unloaded while we waited for the refMu lock
	if runner.llama == nil {
		return fmt.Errorf("model was unloaded to make room for another model")
	}
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

type ByDuration []*runnerRef

func (a ByDuration) Len() int      { return len(a) }
func (a ByDuration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDuration) Less(i, j int) bool {
	// uint64 to turn negative time (never unload) to largest
	return uint64(a[i].sessionDuration) < uint64(a[j].sessionDuration)
}

type BySize []*runnerRef

func (a BySize) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].estimatedVRAM < a[j].estimatedVRAM }

// Only allow the getRunner func to run one at a time
var loadMu sync.Mutex

// load a model into memory if it is not already loaded
func getRunner(c *gin.Context, model *Model, opts api.Options, sessionDuration time.Duration) (*runnerRef, error) {
	slog.Debug("getRunner called", "model", model.ModelPath)
	loadMu.Lock()
	defer loadMu.Unlock()
	slog.Debug("getRunner processing", "model", model.ModelPath)

	loadedMu.Lock()
	runner := loaded[model.ModelPath]
	loadedMu.Unlock()

	var gpus gpu.GpuInfoList
	var ggml *llm.GGML

	if runner != nil {
		slog.Debug("evaluating already loaded", "model", model.ModelPath)
		runner.refMu.Lock()
		defer runner.refMu.Unlock()
		// Ignore the NumGPU settings
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
		for len(loaded) > 0 {
			// TODO - optional user visible max runner setting

			slog.Debug("attempting to fit new model with existing models running", "runner_count", len(loaded))

			gpus = gpu.GetGPUInfo()
			if ggml.IsCPUOnlyModel() || (len(gpus) == 1 && gpus[0].Library == "cpu") {

				// TODO evaluate system memory to see if we should block the load, or force an unload of another CPU runner

				slog.Debug("CPU mode, allowing multiple model loads")
				break
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
					for _, gpu := range gpus {
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
			for i := range gpus {
				if p, ok := predMap[predKey{gpus[i].Library, gpus[i].ID}]; ok {
					slog.Debug(fmt.Sprintf("[%s] %s reported freeMemory %dM", gpus[i].ID, gpus[i].Library, gpus[i].FreeMemory/1024/1024))
					if p > gpus[i].TotalMemory {
						// Shouldn't happen
						slog.Warn("predicted usage exceeds VRAM", "ID", gpus[i].ID, "totalMemory", gpus[i].TotalMemory, "predicted", p)
						gpus[i].FreeMemory = 0
					} else if (gpus[i].TotalMemory - p) < gpus[i].FreeMemory { // predicted free is smaller than reported free, use it
						// TODO maybe we should just always trust our numbers, since cuda's free memory reporting is laggy
						// and we might unload models we didn't actually need to.  The risk is if some other GPU intensive app is loaded
						// after we start our first runner, then we'll never acount for that, so picking the smallest free value seems prudent.
						gpus[i].FreeMemory = gpus[i].TotalMemory - p
					}
					slog.Info(fmt.Sprintf("[%s] updated %s totalMemory %dM", gpus[i].ID, gpus[i].Library, gpus[i].TotalMemory/1024/1024))
					slog.Info(fmt.Sprintf("[%s] updated %s freeMemory  %dM", gpus[i].ID, gpus[i].Library, gpus[i].FreeMemory/1024/1024))
				}
			}

			fits := false
			if fits, estimatedVRAM, _ = llm.PredictServerFit(gpus, ggml, model.AdapterPaths, model.ProjectorPaths, opts); fits {
				slog.Debug("new model will fit in available VRAM, loading", "model", model.ModelPath)
				break
			}
			slog.Debug("new model will not fit in available VRAM without unloading another model", "model", model.ModelPath, "requiredM", estimatedVRAM/1024/1024)
			gpus = nil // force a refresh of the gpu info after we unload a model.

			// If we get to here, then we have GPUs, and can't fit the new model in VRAM
			// Find the best candidate model to unload
			unloadBestFitRunner()

			// For CUDA, free memory reporting is laggy and will lead us astray
			// TODO is this sufficient, or too long? (initial testing looks like this is sufficient)
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
			gpus = gpu.GetGPUInfo()
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

// Caller must already have loadedMu lock
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
			// TODO - prevent new requests from coming in
			r.refCond.Wait()
		}
	}

	slog.Info("unloading old model before its timeout to make room for new model", "old_model", r.model)

	if r.expireTimer != nil {
		r.expireTimer.Stop()
	}

	r.llama.Close()
	r.llama = nil
	r.adapters = nil
	r.projectors = nil
	r.Options = nil
	r.gpus = nil
	delete(loaded, r.model)
}
