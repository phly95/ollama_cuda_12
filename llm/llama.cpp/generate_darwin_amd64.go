package llm

//go:generate git submodule init

//go:generate git submodule update --force ggml
//go:generate git -C ggml apply ../patches/0001-add-detokenize-endpoint.patch
//go:generate git -C ggml apply ../patches/0002-34B-model-support.patch
//go:generate git -C ggml apply ../patches/0003-metal-fix-synchronization-in-new-matrix-multiplicati.patch
//go:generate git -C ggml apply ../patches/0004-metal-add-missing-barriers-for-mul-mat-2699.patch
//go:generate cmake -S ggml -B ggml/build/cpu -DLLAMA_ACCELERATE=on -DLLAMA_K_QUANTS=on -DCMAKE_SYSTEM_PROCESSOR=x86_64 -DCMAKE_OSX_ARCHITECTURES=x86_64 -DCMAKE_OSX_DEPLOYMENT_TARGET=11.0
//go:generate cmake --build ggml/build/cpu --target server --config Release
//go:generate mv ggml/build/cpu/bin/server ggml/build/cpu/bin/ollama-runner

//go:generate git submodule update --force gguf
//go:generate rm -f gguf/examples/server/server.h
//go:generate git -C gguf apply ../patches/0001-Expose-callable-API-for-server.patch
//go:generate cmake -S gguf -B gguf/build/cpu -DCMAKE_BUILD_TYPE=Debug -DLLAMA_METAL=off -DLLAMA_ACCELERATE=on -DLLAMA_K_QUANTS=on -DCMAKE_SYSTEM_PROCESSOR=x86_64 -DCMAKE_OSX_ARCHITECTURES=x86_64 -DCMAKE_OSX_DEPLOYMENT_TARGET=11.0
//go:generate cmake --build gguf/build/cpu --target ggml --target ggml_static --target llama --target build_info --target common --target ext_server --config Release
