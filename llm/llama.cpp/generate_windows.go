package llm

//go:generate git submodule init

//go:generate git submodule update --force ggml
//go:generate git -C ggml apply ../patches/0001-add-detokenize-endpoint.patch
//go:generate git -C ggml apply ../patches/0002-34B-model-support.patch
//go:generate cmake -S ggml -B ggml/build/cpu -DLLAMA_K_QUANTS=on
//go:generate cmake --build ggml/build/cpu --target server --config Release
//go:generate cmd /c move ggml\build\cpu\bin\Release\server.exe ggml\build\cpu\bin\Release\ollama-runner.exe

//go:generate git submodule update --force gguf
//go:generate powershell -c "rm -erroraction ignore -path gguf/examples/server/server.h; exit(0)"
//go:generate git -C gguf apply ../patches/0001-Expose-callable-API-for-server.patch
// go:generate cmake -S gguf -B gguf/build/wincuda -DCMAKE_VERBOSE_MAKEFILE=ON -DBUILD_SHARED_LIBS=on -A x64 -DLLAMA_CUBLAS=on -DLLAMA_ACCELERATE=on -DLLAMA_K_QUANTS=on -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off
//go:generate cmake -S gguf -B gguf/build/wincuda -DLLAMA_CUBLAS=ON -DCMAKE_VERBOSE_MAKEFILE=ON -DBUILD_SHARED_LIBS=on -A x64
//go:generate cmake --build gguf/build/wincuda --target server --target llava_shared --target ggml_shared --target llama --target build_info --target common --target ext_server_shared --config Release
