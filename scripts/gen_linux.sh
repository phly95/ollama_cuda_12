#!/bin/bash
# This script is intended to run inside the go generate
# working directory must be ../llm/llama.cpp

set -ex
set -o pipefail

echo "Starting linux generate script"
source $(dirname $0)/gen_common.sh
init_vars
git_module_setup
apply_patches
CMAKE_DEFS="-DLLAMA_CUBLAS=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off ${CMAKE_DEFS}"
BUILD_DIR="gguf/build/cuda"
build
gcc -fPIC -shared -o cuda_server.so -D__GPU_TYPE__=cuda -DLLAMA_SERVER_LIBRARY=1 -I./gguf -I./gguf/common wrap_server.c -Wl,--whole-archive \
    ${BUILD_DIR}/examples/server/libext_server.a \
    ${BUILD_DIR}/common/libcommon.a \
    ${BUILD_DIR}/libllama.a \
    ${BUILD_DIR}/libggml_static.a \
    -Wl,--no-whole-archive \
    /usr/local/cuda/lib64/libcudart_static.a \
    /usr/local/cuda/lib64/libcublas_static.a \
    /usr/local/cuda/lib64/libcublasLt_static.a \
    /usr/local/cuda/lib64/libcudadevrt.a \
    /usr/local/cuda/lib64/libculibos.a

if [ -n "${ROCM_PATH}" -a -d "${ROCM_PATH}" ] ; then
    echo "Building ROCm"
    init_vars
    CMAKE_DEFS="-DCMAKE_C_COMPILER=${ROCM_PATH}/llvm/bin/clang -DCMAKE_CXX_COMPILER=${ROCM_PATH}/llvm/bin/clang++ -DAMDGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102' -DGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102' -DLLAMA_HIPBLAS=on -DLLAMA_K_QUANTS=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off ${CMAKE_DEFS}"
    BUILD_DIR="gguf/build/rocm"
    build
    gcc -fPIC -shared -o rocm_server.so -D__GPU_TYPE__=rocm -DLLAMA_SERVER_LIBRARY=1 -I./gguf -I./gguf/common wrap_server.c -Wl,--whole-archive \
        ${BUILD_DIR}/examples/server/libext_server.a \
        ${BUILD_DIR}/common/libcommon.a \
        ${BUILD_DIR}/libllama.a \
        ${BUILD_DIR}/libggml_static.a \
        -Wl,--no-whole-archive
    # TODO - more roc libraries likely...
fi