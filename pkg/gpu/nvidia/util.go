package nvidia

import (
	"fmt"
	"strconv"
	"time"

	log "github.com/golang/glog"
	"k8s.io/api/core/v1"
)

// Get GPU Memory of the Pod
func getGPUCountFromPodResource(pod *v1.Pod) uint {
	var total uint
	containers := pod.Spec.Containers
	for _, container := range containers {
		if val, ok := container.Resources.Limits[resourceName]; ok {
			total += uint(val.Value())
		}
	}
	return total
}

// get assumed timestamp
func getAssumeTimeFromPodAnnotation(pod *v1.Pod) (assumeTime uint64) {
	if assumeTimeStr, ok := pod.ObjectMeta.Annotations[EnvResourceAssumeTime]; ok {
		u64, err := strconv.ParseUint(assumeTimeStr, 10, 64)
		if err != nil {
			log.Warningf("Failed to parse assume Timestamp %s due to %v", assumeTimeStr, err)
		} else {
			assumeTime = u64
		}
	}

	return assumeTime
}

func getGPUIDsFromPodAnnotation(pod *v1.Pod) (ids string) {
	if len(pod.ObjectMeta.Annotations) > 0 {
		value, found := pod.ObjectMeta.Annotations[EnvResourceIndex]
		if found && len(value) != 0 {
			ids = value
		} else {
			log.Warningf("Failed to get dev id %s for pod %s in ns %s",
				pod.Name,
				pod.Namespace)
		}
	}
	return ids
}

//  update pod env with assigned status
func updatePodAnnotations(oldPod *v1.Pod) (newPod *v1.Pod) {
	newPod = oldPod.DeepCopy()
	if len(newPod.ObjectMeta.Annotations) == 0 {
		newPod.ObjectMeta.Annotations = map[string]string{}
	}

	now := time.Now()
	newPod.ObjectMeta.Annotations[EnvAssignedFlag] = "true"
	newPod.ObjectMeta.Annotations[EnvResourceAssumeTime] = fmt.Sprintf("%d", now.UnixNano())

	return newPod
}
