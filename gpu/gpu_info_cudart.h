#ifndef __APPLE__
#ifndef __GPU_INFO_CUDART_H__
#define __GPU_INFO_CUDART_H__
#include "gpu_info.h"

// Just enough typedef's to dlopen/dlsym for memory information
typedef enum cudartReturn_enum {
  CUDART_SUCCESS = 0,
  CUDART_UNSUPPORTED = 1,
  CUDA_ERROR_INSUFFICIENT_DRIVER = 35,
  // Other values omitted for now...
} cudartReturn_t;

typedef enum cudartDeviceAttr_enum {
  cudartDevAttrComputeCapabilityMajor = 75,
  cudartDevAttrComputeCapabilityMinor = 76,

  // TODO - not yet wired up but may be useful for Jetson or other
  // integrated GPU scenarios with shared memory
  cudaDevAttrIntegrated = 18

} cudartDeviceAttr_t;

typedef void *cudartDevice_t;  // Opaque is sufficient
typedef struct cudartMemory_st {
  size_t total;
  size_t free;
  size_t used;
} cudartMemory_t;

typedef struct cudartDriverVersion {
  int major;
  int minor;
} cudartDriverVersion_t;

typedef struct cudaUUID {
    unsigned char bytes[16];
} cudaUUID_t;
typedef struct cudaDeviceProp {
  char name[256];
  cudaUUID_t uuid;
  size_t totalGlobalMem;
  size_t sharedMemPerBlock;
  int regsPerBlock;
  int warpSize;
  size_t memPitch;
  int maxThreadsPerBlock;
  int maxThreadsDim[3];
  int maxGridSize[3];
  int clockRate;
  size_t totalConstMem;
  int major;
  int minor;
  size_t textureAlignment;
  size_t texturePitchAlignment;
  int deviceOverlap;
  int multiProcessorCount;
  int kernelExecTimeoutEnabled;
  int integrated;
  int canMapHostMemory;
  int computeMode;
  int maxTexture1D;
  int maxTexture1DMipmap;
  int maxTexture1DLinear;
  int maxTexture2D[2];
  int maxTexture2DMipmap[2];
  int maxTexture2DLinear[3];
  int maxTexture2DGather[2];
  int maxTexture3D[3];
  int maxTexture3DAlt[3];
  int maxTextureCubemap;
  int maxTexture1DLayered[2];
  int maxTexture2DLayered[3];
  int maxTextureCubemapLayered[2];
  int maxSurface1D;
  int maxSurface2D[2];
  int maxSurface3D[3];
  int maxSurface1DLayered[2];
  int maxSurface2DLayered[3];
  int maxSurfaceCubemap;
  int maxSurfaceCubemapLayered[2];
  size_t surfaceAlignment;
  int concurrentKernels;
  int ECCEnabled;
  int pciBusID;
  int pciDeviceID;
  int pciDomainID;
  int tccDriver;
  int asyncEngineCount;
  int unifiedAddressing;
  int memoryClockRate;
  int memoryBusWidth;
  int l2CacheSize;
  int persistingL2CacheMaxSize;
  int maxThreadsPerMultiProcessor;
  int streamPrioritiesSupported;
  int globalL1CacheSupported;
  int localL1CacheSupported;
  size_t sharedMemPerMultiprocessor;
  int regsPerMultiprocessor;
  int managedMemory;
  int isMultiGpuBoard;
  int multiGpuBoardGroupID;
  int singleToDoublePrecisionPerfRatio;
  int pageableMemoryAccess;
  int concurrentManagedAccess;
  int computePreemptionSupported;
  int canUseHostPointerForRegisteredMem;
  int cooperativeLaunch;
  int cooperativeMultiDeviceLaunch;
  int pageableMemoryAccessUsesHostPageTables;
  int directManagedMemAccessFromHost;
  int accessPolicyMaxWindowSize;
} cudaDeviceProp_t;

typedef struct cudart_handle {
  void *handle;
  uint16_t verbose;
  cudartReturn_t (*cudaSetDevice)(int device);
  cudartReturn_t (*cudaDeviceSynchronize)(void);
  cudartReturn_t (*cudaDeviceReset)(void);
  cudartReturn_t (*cudaMemGetInfo)(size_t *, size_t *);
  cudartReturn_t (*cudaGetDeviceCount)(int *);
  cudartReturn_t (*cudaDeviceGetAttribute)(int* value, cudartDeviceAttr_t attr, int device);
  cudartReturn_t (*cudaDriverGetVersion) (int *driverVersion);
  cudartReturn_t (*cudaGetDeviceProperties) (cudaDeviceProp_t* prop, int device);
} cudart_handle_t;

typedef struct cudart_init_resp {
  char *err;  // If err is non-null handle is invalid
  cudart_handle_t ch;
  int num_devices;
} cudart_init_resp_t;

void cudart_init(char *cudart_lib_path, cudart_init_resp_t *resp);
void cudart_check_vram(cudart_handle_t ch, int device_id, mem_info_t *resp);
void cudart_release(cudart_handle_t ch);

#endif  // __GPU_INFO_CUDART_H__
#endif  // __APPLE__
