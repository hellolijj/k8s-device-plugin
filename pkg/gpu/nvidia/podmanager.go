package nvidia

import (
	"sort"

	"k8s.io/api/core/v1"
)

type orderedPodByAssumeTime []*v1.Pod

func (this orderedPodByAssumeTime) Len() int {
	return len(this)
}

func (this orderedPodByAssumeTime) Less(i, j int) bool {
	return getAssumeTimeFromPodAnnotation(this[i]) <= getAssumeTimeFromPodAnnotation(this[j])
}

func (this orderedPodByAssumeTime) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

// make the pod ordered by GPU assumed time
func makePodOrderdByAge(pods []*v1.Pod) []*v1.Pod {
	newPodList := make(orderedPodByAssumeTime, 0, len(pods))
	for _, v := range pods {
		newPodList = append(newPodList, v)
	}
	sort.Sort(newPodList)
	return []*v1.Pod(newPodList)
}
