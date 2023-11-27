#include "wrap_server.h"

#ifdef __cplusplus
extern "C"
{
#endif

inline ext_server_err cuda_llama_server_init(ext_server_params *sparams) {
    return llama_server_init(sparams);
}

inline void cuda_llama_server_start() {
    llama_server_start();
}

inline void cuda_llama_server_stop() {
    llama_server_stop();
}

inline ext_server_completion_resp cuda_llama_server_completion(const char *json_req) {
    return llama_server_completion(json_req);
}

inline ext_task_result cuda_llama_server_completion_next_result(const int task_id) {
    return llama_server_completion_next_result(task_id);
}

inline ext_server_err cuda_llama_server_completion_cancel(const int task_id) {
    return llama_server_completion_cancel(task_id);
}

inline ext_server_err cuda_llama_server_tokenize(const char *json_req, ext_server_resp *resp) {
    return llama_server_tokenize(json_req, resp);
}

inline ext_server_err cuda_llama_server_detokenize(const char *json_req, ext_server_resp *resp) {
    return llama_server_detokenize(json_req, resp);
}

inline ext_server_err cuda_llama_server_embedding(const char *json_req, ext_server_resp *resp) {
    return llama_server_embedding(json_req, resp);
}

inline int64_t cuda_check_vram() {
    return check_vram();
}

#ifdef __cplusplus
extern "C"
}
#endif