#!/bin/bash
# This script is intended to run inside the go generate
# working directory must be ../llm/llama.cpp

set -ex
set -o pipefail

echo "Starting linux generate script"
if [ -z "${CUDACXX}" -a -x /usr/local/cuda/bin/nvcc ] ; then
    export CUDACXX=/usr/local/cuda/bin/nvcc
fi
source $(dirname $0)/gen_common.sh
init_vars
git_module_setup
apply_patches
CMAKE_DEFS="-DLLAMA_CUBLAS=on -DCMAKE_POSITION_INDEPENDENT_CODE=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off ${CMAKE_DEFS}"
BUILD_DIR="gguf/build/cuda"
build
gcc -fPIC -g -shared -o libcuda_server.so -DLLAMA_SERVER_LIBRARY=1 -I./gguf  wrap_server.c \
    -fvisibility=hidden \
    -Wl,--whole-archive \
    ${BUILD_DIR}/examples/server/CMakeFiles/ext_server.dir/server.cpp.o \
    ${BUILD_DIR}/common/libcommon.a \
    ${BUILD_DIR}/libllama.a \
    ${BUILD_DIR}/examples/llava/libllava_static.a \
    -Wl,--no-whole-archive \
    /usr/local/cuda/lib64/libcudart_static.a \
    /usr/local/cuda/lib64/libcublas_static.a \
    /usr/local/cuda/lib64/libcublasLt_static.a \
    /usr/local/cuda/lib64/libcudadevrt.a \
    /usr/local/cuda/lib64/libculibos.a

if [ -n "${ROCM_PATH}" -a -d "${ROCM_PATH}" ] ; then
    echo "Building ROCm"
    init_vars
    CMAKE_DEFS="-DBUILD_SHARED_LIBS=on -DCMAKE_POSITION_INDEPENDENT_CODE=on -DCMAKE_C_COMPILER=${ROCM_PATH}/llvm/bin/clang -DCMAKE_CXX_COMPILER=${ROCM_PATH}/llvm/bin/clang++ -DAMDGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102' -DGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102' -DLLAMA_HIPBLAS=on -DLLAMA_K_QUANTS=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off ${CMAKE_DEFS}"
    BUILD_DIR="gguf/build/rocm"
    build
    gcc -g -fPIC -shared -o librocm_server.so -D__GPU_TYPE__=rocm -DLLAMA_SERVER_LIBRARY=1 -I./gguf -I./gguf/common wrap_server.c \
        ${BUILD_DIR}/examples/server/libext_server.a \
        ${BUILD_DIR}/common/libcommon.a \
        ${BUILD_DIR}/libllama.a \
        ${BUILD_DIR}/libggml_static.a
    # TODO - more roc libraries likely...
fi