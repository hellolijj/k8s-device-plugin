package nvidia

import (
	"fmt"
	"time"

	log "github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

var (
	retries = 5
)

func buildErrResponse(reqs *pluginapi.AllocateRequest) *pluginapi.AllocateResponse {
	responses := pluginapi.AllocateResponse{}
	for range reqs.ContainerRequests {
		response := pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				EnvResourceIndex: fmt.Sprintf("-1"),
			},
		}
		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}
	return &responses
}

// Allocate which return list of devices.
func (m *NvidiaDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	devs := m.devs
	responses := pluginapi.AllocateResponse{}

	log.Infoln("----Allocating GPU for gpu mem is started----")
	var (
		podReqGPU uint
		found     bool
		assumePod *v1.Pod
	)
	
	for _, req := range reqs.ContainerRequests {
		podReqGPU += uint(len(req.DevicesIDs))
	}
	log.Infof("RequestPodGPUs: %d", podReqGPU)

	m.Lock()
	defer m.Unlock()
	log.Infoln("checking...")
	pods, err := getCandidatePods()
	if err != nil {
		log.Infof("invalid allocation requst: Failed to find candidate pods due to %v", err)
		return buildErrResponse(reqs), nil
	}

	if log.V(4) {
		for _, pod := range pods {
			log.Infof("Pod %s in ns %s request GPU %d with timestamp %v",
				pod.Name,
				pod.Namespace,
				getGPUCountFromPodResource(pod),
				getAssumeTimeFromPodAnnotation(pod))
		}
	}

	for _, pod := range pods {
		if getGPUCountFromPodResource(pod) == podReqGPU {
			log.Infof("Found Assumed GPU shared Pod %s in ns %s with GPU Memory %d",
				pod.Name,
				pod.Namespace,
				podReqGPU)
			assumePod = pod
			found = true
			break
		}
	}

	if found {
		ids := getGPUIDsFromPodAnnotation(assumePod)
		if len(ids) == 0 {
			log.Warningf("Failed to get the dev ", assumePod)
		}

		// 1. Create container requests
		for _, req := range reqs.ContainerRequests {
			response := pluginapi.ContainerAllocateResponse{
				Envs: map[string]string{
					EnvNVGPU: ids,
				},
			}
			for _, id := range req.DevicesIDs {
				if !deviceExists(devs, id) {
					return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
				}
			}
			responses.ContainerResponses = append(responses.ContainerResponses, &response)
		}

		// 2. Update Pod spec
		newPod := updatePodAnnotations(assumePod)
		_, err = clientset.CoreV1().Pods(newPod.Namespace).Update(newPod)
		if err != nil {
			// the object has been modified; please apply your changes to the latest version and try again
			if err.Error() == OptimisticLockErrorMsg {
				// retry
				pod, err := clientset.CoreV1().Pods(assumePod.Namespace).Get(assumePod.Name, metav1.GetOptions{})
				if err != nil {
					log.Warningf("Failed due to %v", err)
					return buildErrResponse(reqs), nil
				}
				newPod = updatePodAnnotations(pod)
				_, err = clientset.CoreV1().Pods(newPod.Namespace).Update(newPod)
				if err != nil {
					log.Warningf("Failed due to %v", err)
					return buildErrResponse(reqs), nil
				}
			} else {
				log.Warningf("Failed due to %v", err)
				return buildErrResponse(reqs), nil
			}
		}

	} else {
		log.Warningf("invalid allocation requst: request GPU %d can't be satisfied.",
			podReqGPU)
		return buildErrResponse(reqs), nil
	}

	return &responses, nil
}

// pick up the gpushare pod with assigned status is false, and
func getCandidatePods() ([]*v1.Pod, error) {
	candidatePods := []*v1.Pod{}
	allPods, err := getPendingPodsInNode()
	if err != nil {
		return candidatePods, err
	}
	for _, pod := range allPods {
		current := pod
		if isGPUAssumedPod(&current) {
			candidatePods = append(candidatePods, &current)
		}
	}

	if log.V(4) {
		for _, pod := range candidatePods {
			log.Infof("candidate pod %s in ns %s with timestamp %d is found.",
				pod.Name,
				pod.Namespace,
				getAssumeTimeFromPodAnnotation(pod))
		}
	}

	return makePodOrderdByAge(candidatePods), nil
}

func getPendingPodsInNode() ([]v1.Pod, error) {
	pods := []v1.Pod{}

	podIDMap := map[types.UID]bool{}

	selector := fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName, "status.phase": "Pending"})
	podList, err := clientset.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
		FieldSelector: selector.String(),
		LabelSelector: labels.Everything().String(),
	})

	// 延迟 5s 统计 pending pod
	for i := 0; i < retries && err != nil; i++ {
		podList, err = clientset.CoreV1().Pods(v1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: selector.String(),
			LabelSelector: labels.Everything().String(),
		})
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get Pods assigned to node %v", nodeName)
	}

	log.V(5).Infof("all pod list %v", podList.Items)

	for _, pod := range podList.Items {
		if pod.Spec.NodeName != nodeName {
			log.Warningf("Pod name %s in ns %s is not assigned to node %s as expected, it's placed on node %s ",
				pod.Name,
				pod.Namespace,
				nodeName,
				pod.Spec.NodeName)
		} else {
			log.Infof("list pod %s in ns %s in node %s and status is %s",
				pod.Name,
				pod.Namespace,
				nodeName,
				pod.Status.Phase,
			)
			if _, ok := podIDMap[pod.UID]; !ok {
				pods = append(pods, pod)
				podIDMap[pod.UID] = true
			}
		}

	}

	return pods, nil
}

// determine if the pod is GPU share pod, and is already assumed but not assigned
func isGPUAssumedPod(pod *v1.Pod) (assumed bool) {
	log.V(6).Infof("Determine if the pod %v is GPUSharedAssumed pod", pod)
	var ok bool

	// 1. Check if it's for GPU share
	if getGPUCountFromPodResource(pod) <= 0 {
		log.V(6).Infof("Pod %s in namespace %s has not GPU Memory Request, so it's not GPUSharedAssumed assumed pod.",
			pod.Name,
			pod.Namespace)
		return assumed
	}

	// 2. Check if it already has assume time
	if _, ok = pod.ObjectMeta.Annotations[EnvResourceAssumeTime]; !ok {
		log.V(4).Infof("No assume timestamp for pod %s in namespace %s, so it's not GPUAssumed assumed pod.",
			pod.Name,
			pod.Namespace)
		return assumed
	}

	// 3. Check if it has been assigned already
	if assigned, ok := pod.ObjectMeta.Annotations[EnvAssignedFlag]; ok {

		if assigned == "false" {
			log.V(4).Infof("Found GPUSharedAssumed assumed pod %s in namespace %s.",
				pod.Name,
				pod.Namespace)
			assumed = true
		} else {
			log.Infof("GPU assigned Flag for pod %s exists in namespace %s and its assigned status is %s, so it's not GPUSharedAssumed assumed pod.",
				pod.Name,
				pod.Namespace,
				assigned)
		}
	} else {
		log.Warningf("No GPU assigned Flag for pod %s in namespace %s, so it's not GPUSharedAssumed assumed pod.",
			pod.Name,
			pod.Namespace)
	}

	return assumed
}
