#!/bin/bash
# This script is intended to run inside the go generate
# working directory must be ../llm/llama.cpp

set -ex
set -o pipefail

echo "Starting linux generate script"
source $(dirname $0)/gen_common.sh
init_vars

case ${GPU_TYPE} in
    cuda)
        CMAKE_DEFS="-DLLAMA_CUBLAS=on"
        ;;
    rocm)
        CMAKE_DEFS="-DCMAKE_VERBOSE_MAKEFILE=on -DLLAMA_HIPBLAS=on -DLLAMA_K_QUANTS=on -DCMAKE_C_COMPILER=$ROCM_PATH/llvm/bin/clang -DCMAKE_CXX_COMPILER=$ROCM_PATH/llvm/bin/clang++ -DAMDGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102' -DGPU_TARGETS='gfx803;gfx900;gfx906:xnack-;gfx908:xnack-;gfx90a:xnack+;gfx90a:xnack-;gfx1010;gfx1012;gfx1030;gfx1100;gfx1101;gfx1102'"
        ;;
    *)
        echo "Warning, building without GPU acceleration"
        CMAKE_DEFS=""
        ;;
esac
CMAKE_DEFS="-DLLAMA_ACCELERATE=on -DLLAMA_NATIVE=off -DLLAMA_AVX=on -DLLAMA_AVX2=off -DLLAMA_AVX512=off -DLLAMA_FMA=off -DLLAMA_F16C=off ${CMAKE_DEFS}"
BUILD_DIR="gguf/build/${GPU_TYPE}/"
git_module_setup
apply_patches
build