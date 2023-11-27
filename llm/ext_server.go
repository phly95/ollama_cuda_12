package llm

/*
#cgo CFLAGS: -I${SRCDIR}/llama.cpp/ -I${SRCDIR}/llama.cpp/gguf -I${SRCDIR}/llama.cpp/gguf/common
#cgo CFLAGS: -DLLAMA_SERVER_LIBRARY=1 -D_XOPEN_SOURCE=600 -DACCELERATE_NEW_LAPACK -DACCELERATE_LAPACK_ILP64
#cgo CFLAGS: -Wmissing-noreturn -Wall -Wextra -Wcast-qual -Wno-unused-function -Wno-array-bounds
#cgo CPPFLAGS: -Ofast -Wall -Wextra -Wno-unused-function -Wno-unused-variable -Wno-deprecated-declarations -Wno-unused-but-set-variable
#cgo darwin CFLAGS: -D_DARWIN_C_SOURCE
#cgo darwin CPPFLAGS:  -DGGML_USE_ACCELERATE
#cgo darwin,arm64 CPPFLAGS: -DGGML_USE_METAL
#cgo darwin LDFLAGS: -lc++ -framework Accelerate
#cgo darwin,arm64 LDFLAGS: -framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/metal/common/libcommon.a
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/metal/examples/server/libext_server.a
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/metal/libllama.a
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/metal/libggml_static.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/cpu/common/libcommon.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/cpu/examples/server/libext_server.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/cpu/libllama.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/llama.cpp/gguf/build/cpu/libggml_static.a
#cgo linux CFLAGS: -D_GNU_SOURCE
#cgo linux windows CFLAGS: -DGGML_CUDA_DMMV_X=32 -DGGML_CUDA_MMV_Y=1 -DGGML_CUDA_PEER_MAX_BATCH_SIZE=128 -DGGML_USE_CUBLAS
#cgo linux LDFLAGS: -L${SRCDIR}/llama.cpp/ -lcuda_server
#cgo linux LDFLAGS: -lrt -lpthread -ldl -lstdc++ -lm
#cgo windows LDFLAGS: -L${SRCDIR}/llama.cpp/gguf/build/wincuda/dist/bin
#cgo windows LDFLAGS: -lext_server_shared -lpthread

#include "wrap_server.h"

*/
import "C"
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/jmorganca/ollama/api"
)

func errWrap(resp C.ext_server_err) error {
	if resp.code == 0 {
		return nil
	}
	err := fmt.Errorf(C.GoString(resp.err))
	C.free(unsafe.Pointer(resp.err))
	return err
}

type llamaExtServer struct {
	api.Options
}

// Note: current implementation does not support concurrent instantiations
var mutex sync.Mutex

func newLlamaExtServer(model string, adapters []string, numLayers int64, opts api.Options) (*llamaExtServer, error) {
	if !mutex.TryLock() {
		log.Printf("concurrent llm servers not yet supported, waiting for prior server to complete")
		mutex.Lock()
	}
	server := &llamaExtServer{opts}
	fileInfo, err := os.Stat(model)
	if err != nil {
		return nil, err
	}
	var sparams C.ext_server_params
	sparams.model = C.CString(model)
	defer C.free(unsafe.Pointer(sparams.model))

	numGPU := NumGPU(numLayers, fileInfo.Size(), opts)

	sparams.embedding = true
	sparams.n_ctx = C.uint(opts.NumCtx)
	sparams.n_batch = C.uint(opts.NumBatch)
	sparams.n_gpu_layers = C.int(numGPU)

	if opts.RopeFrequencyBase > 0 {
		sparams.rope_freq_base = C.float(opts.RopeFrequencyBase)
	} else {
		sparams.rope_freq_base = 0.0
	}

	if opts.RopeFrequencyScale > 0 {
		sparams.rope_freq_scale = C.float(opts.RopeFrequencyScale)
	} else {
		sparams.rope_freq_scale = 0.0
	}

	sparams.lora_adapters = nil
	for i := 0; i < len(adapters); i++ {
		la := (*C.ext_server_lora_adapter)(C.malloc(C.sizeof_struct_ext_server_lora_adapter))
		defer C.free(unsafe.Pointer(la))
		la.adapter = C.CString(adapters[i])
		defer C.free(unsafe.Pointer(la.adapter))
		la.scale = C.float(1.0) // TODO expose scale/weights up through ollama UX
		la.next = nil
		if i == 0 {
			sparams.lora_adapters = la
		} else {
			tmp := sparams.lora_adapters
			for ; tmp.next != nil; tmp = tmp.next {
			}
			tmp.next = la
		}
	}

	if opts.NumThread > 0 {
		sparams.n_threads = C.uint(opts.NumThread)
	} else {
		sparams.n_threads = C.uint(runtime.NumCPU())
	}

	sparams.memory_f16 = false
	if opts.F16KV {
		sparams.memory_f16 = true
	}
	sparams.use_mlock = false
	if opts.UseMLock {
		sparams.use_mlock = true
	}
	sparams.use_mmap = true
	if !opts.UseMMap {
		sparams.use_mmap = false
	}
	sparams.numa = false
	if opts.UseNUMA {
		sparams.numa = true
	}

	log.Printf("Initializing internal llama server")
	err = errWrap(C.cuda_llama_server_init(&sparams))
	if err != nil {
		return nil, err
	}

	log.Printf("Starting internal llama main loop")
	C.cuda_llama_server_start()
	return server, nil
}

func (llm *llamaExtServer) Predict(ctx context.Context, prevContext []int, prompt string, format string, fn func(api.GenerateResponse)) error {
	prevConvo, err := llm.Decode(ctx, prevContext)
	if err != nil {
		return err
	}

	nextContext := strings.Builder{}
	// Remove leading spaces from prevConvo if present
	prevConvo = strings.TrimPrefix(prevConvo, " ")
	nextContext.WriteString(prevConvo)
	nextContext.WriteString(prompt)

	request := map[string]any{
		"prompt":            nextContext.String(),
		"stream":            true,
		"n_predict":         llm.NumPredict,
		"n_keep":            llm.NumKeep,
		"temperature":       llm.Temperature,
		"top_k":             llm.TopK,
		"top_p":             llm.TopP,
		"tfs_z":             llm.TFSZ,
		"typical_p":         llm.TypicalP,
		"repeat_last_n":     llm.RepeatLastN,
		"repeat_penalty":    llm.RepeatPenalty,
		"presence_penalty":  llm.PresencePenalty,
		"frequency_penalty": llm.FrequencyPenalty,
		"mirostat":          llm.Mirostat,
		"mirostat_tau":      llm.MirostatTau,
		"mirostat_eta":      llm.MirostatEta,
		"penalize_nl":       llm.PenalizeNewline,
		"seed":              llm.Seed,
		"stop":              llm.Stop,
	}

	if format == "json" {
		request["grammar"] = jsonGrammar
	}

	// Handling JSON marshaling with special characters unescaped.
	buffer := &bytes.Buffer{}
	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(request); err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	req := C.CString(buffer.String())
	defer C.free(unsafe.Pointer(req))

	cmpCtx := C.cuda_llama_server_completion(req)
	if cmpCtx.task_id < 0 {
		defer C.free(unsafe.Pointer(cmpCtx.err))
		return fmt.Errorf(C.GoString(cmpCtx.err))
	}

	for {
		select {
		case <-ctx.Done():
			// This handles the request cancellation
			return errWrap(C.cuda_llama_server_completion_cancel(cmpCtx.task_id))
		default:
			result := C.cuda_llama_server_completion_next_result(cmpCtx.task_id)
			if result.result_json != nil {
				defer C.free(unsafe.Pointer(result.result_json))
			}
			var p prediction
			if err := json.Unmarshal([]byte(C.GoString(result.result_json)), &p); err != nil {
				err2 := errWrap(C.cuda_llama_server_completion_cancel(cmpCtx.task_id))
				return errors.Join(fmt.Errorf("error unmarshaling llm prediction response: %w", err), err2)
			}

			if p.Content != "" {
				fn(api.GenerateResponse{Response: p.Content})
				nextContext.WriteString(p.Content)
			}

			embd, err := llm.Encode(ctx, nextContext.String())
			if err != nil {
				err2 := errWrap(C.cuda_llama_server_completion_cancel(cmpCtx.task_id))
				return errors.Join(fmt.Errorf("encoding context: %w", err), err2)
			}

			fn(api.GenerateResponse{
				Done:               bool(result.stop),
				Context:            embd,
				PromptEvalCount:    p.Timings.PromptN,
				PromptEvalDuration: parseDurationMs(p.Timings.PromptMS),
				EvalCount:          p.Timings.PredictedN,
				EvalDuration:       parseDurationMs(p.Timings.PredictedMS),
			})
			if result.stop {
				return nil
			} else if result.error {
				msg := ""
				if result.result_json != nil {
					msg = C.GoString(result.result_json)
				}
				return fmt.Errorf("error processing completion %s", msg)
			}
		}
	}
}

func (llm *llamaExtServer) Encode(ctx context.Context, prompt string) ([]int, error) {
	data, err := json.Marshal(TokenizeRequest{Content: prompt})
	if err != nil {
		return nil, fmt.Errorf("marshaling encode data: %w", err)
	}
	req := C.CString(string(data))
	defer C.free(unsafe.Pointer(req))
	var resp C.ext_server_resp
	err = errWrap(C.cuda_llama_server_tokenize(req, &resp))
	if resp.json_resp != nil {
		defer C.free(unsafe.Pointer(resp.json_resp))
	}

	var encoded TokenizeResponse
	if err2 := json.Unmarshal([]byte(C.GoString(resp.json_resp)), &encoded); err2 != nil {
		return nil, fmt.Errorf("unmarshal encode response: %w", err2)
	}

	return encoded.Tokens, err
}

func (llm *llamaExtServer) Decode(ctx context.Context, tokens []int) (string, error) {
	if len(tokens) == 0 {
		return "", nil
	}
	data, err := json.Marshal(DetokenizeRequest{Tokens: tokens})
	if err != nil {
		return "", fmt.Errorf("marshaling decode data: %w", err)
	}

	req := C.CString(string(data))
	defer C.free(unsafe.Pointer(req))
	var resp C.ext_server_resp
	err = errWrap(C.cuda_llama_server_detokenize(req, &resp))
	if resp.json_resp != nil {
		defer C.free(unsafe.Pointer(resp.json_resp))
	}

	var decoded DetokenizeResponse
	if err2 := json.Unmarshal([]byte(C.GoString(resp.json_resp)), &decoded); err2 != nil {
		return "", fmt.Errorf("unmarshal encode response: %w", err2)
	}

	return decoded.Content, err
}

func (llm *llamaExtServer) Embedding(ctx context.Context, input string) ([]float64, error) {
	data, err := json.Marshal(TokenizeRequest{Content: input})
	if err != nil {
		return nil, fmt.Errorf("error marshaling embed data: %w", err)
	}

	req := C.CString(string(data))
	defer C.free(unsafe.Pointer(req))
	var resp C.ext_server_resp
	err = errWrap(C.cuda_llama_server_embedding(req, &resp))
	if resp.json_resp != nil {
		defer C.free(unsafe.Pointer(resp.json_resp))
	}
	if err != nil {
		return nil, err
	}

	var embedding EmbeddingResponse
	if err := json.Unmarshal([]byte(C.GoString(resp.json_resp)), &embedding); err != nil {
		return nil, fmt.Errorf("unmarshal tokenize response: %w", err)
	}

	return embedding.Embedding, nil
}

func (llm *llamaExtServer) Ping(ctx context.Context) error {
	// TODO - consider some mechanism to check if the main loop and llama.cpp are in a good state
	return nil
}

func (llm *llamaExtServer) Close() {
	C.cuda_llama_server_stop()
	mutex.Unlock()
}

func (llm *llamaExtServer) SetOptions(opts api.Options) {
	// TODO - this wont actually do anything at this point...
	llm.Options = opts
}
