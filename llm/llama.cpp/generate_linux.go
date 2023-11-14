package llm

//go:generate git submodule init

//go:generate git submodule update --force ggml
//go:generate git -C ggml apply ../patches/0001-add-detokenize-endpoint.patch
//go:generate git -C ggml apply ../patches/0002-34B-model-support.patch
//go:generate git -C ggml apply ../patches/0005-ggml-support-CUDA-s-half-type-for-aarch64-1455-2670.patch
//go:generate git -C ggml apply ../patches/0001-copy-cuda-runtime-libraries.patch
//go:generate cmake -S ggml -B ggml/build/cpu -DLLAMA_K_QUANTS=on
//go:generate cmake --build ggml/build/cpu --target server --config Release
//go:generate mv ggml/build/cpu/bin/server ggml/build/cpu/bin/ollama-runner

//go:generate git submodule update --force gguf
//go:generate rm -f gguf/examples/server/server.h
//go:generate git -C gguf apply ../patches/0001-Expose-callable-API-for-server.patch

//go:generate cmake -S ggml -B ggml/build/cuda -DLLAMA_CUBLAS=on -DLLAMA_ACCELERATE=on -DLLAMA_K_QUANTS=on
//go:generate cmake --build ggml/build/cuda --target server --config Release
//go:generate mv ggml/build/cuda/bin/server ggml/build/cuda/bin/ollama-runner
//go:generate cmake -S gguf -B gguf/build/cuda -DCMAKE_VERBOSE_MAKEFILE=on -DCMAKE_BUILD_TYPE=Debug -DLLAMA_CUBLAS=on -DLLAMA_ACCELERATE=on -DLLAMA_K_QUANTS=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off
//go:generate cmake --build gguf/build/cuda --target server --target ggml --target ggml_static --target llama --target build_info --target common --target ext_server --config Release
