#include <stdlib.h>
#include "examples/server/server.h"

// TODO - rename this the cuda server and create a second rocm server wrapper

#ifdef __cplusplus
extern "C"
{
#endif

#define _API_ __attribute__ ((visibility ("default")))

_API_ ext_server_err cuda_llama_server_init(ext_server_params *sparams);
_API_ void cuda_llama_server_start();
_API_ void cuda_llama_server_stop();
_API_ ext_server_completion_resp cuda_llama_server_completion(const char *json_req);
_API_ ext_task_result cuda_llama_server_completion_next_result(const int task_id);
_API_ ext_server_err cuda_llama_server_completion_cancel(const int task_id);
_API_ ext_server_err cuda_llama_server_tokenize(const char *json_req, ext_server_resp *resp);
_API_ ext_server_err cuda_llama_server_detokenize(const char *json_req, ext_server_resp *resp);
_API_ ext_server_err cuda_llama_server_embedding(const char *json_req, ext_server_resp *resp) ;
_API_ int64_t cuda_check_vram();

#ifdef __cplusplus
extern "C"
}
#endif