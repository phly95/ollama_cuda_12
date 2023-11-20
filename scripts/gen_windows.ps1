#!powershell

# TODO - this was emitted from codellama and definitely has bugs...

$ErrorActionPreference = "Stop"

function init_vars {
    $patches="0001-Expose-callable-API-for-server.patch"
    $cmakeDefs="-DLLAMA_ACCELERATE=on"
    if ($env:OLLAMA_DEBUG) {
        $cmakeDefs = "-DCMAKE_BUILD_TYPE=Debug -DCMAKE_VERBOSE_MAKEFILE=on ${CMAKE_DEFS}"
    }
}

function git_module_setup {
    # TODO add flags to skip the init/patch logic to make it easier to mod llama.cpp code in-repo
    git submodule init
    git submodule update --force gguf
}

function apply_patches {
    for patch in ${patches} ; do
        git -C gguf apply ../patches/${patch}
    done
}

function build {
    cmake -S gguf -B $buildDir $cmakeDefs
    cmake --build $buildDir $cmakeTargets --config Release
}

init_vars
git_module_setup
apply_patches
build