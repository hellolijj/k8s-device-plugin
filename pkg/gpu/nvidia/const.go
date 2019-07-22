package nvidia

const (
	ENV_GPU_TOPOLOGY_PRIFIX = "GPU"
	ResourceName = "aliyun.com/gpu"
	
	OptimisticLockErrorMsg = "the object has been modified; please apply your changes to the latest version and try again"
	
	EnvNVGPU              = "NVIDIA_VISIBLE_DEVICES"
	EnvResourceIndex      = "ALIYUN_COM_GPU_GROUP"       // 在 annotation 标记使用哪些gpuid 格式 1,2,4 or 2
	EnvAssignedFlag       = "ALIYUN_COM_GPU_ASSIGNED"
	EnvResourceAssumeTime = "ALIYUN_COM_GPU_ASSUME_TIME"
	EnvAnnotationKey      = "GPU_TOPOLOGY"
)
