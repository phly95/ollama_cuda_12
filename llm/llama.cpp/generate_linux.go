//go:build !rocm && !cuda

package llm

//go:generate bash -c "export GPU_TYPE=cpu; ./gen_linux.sh"
