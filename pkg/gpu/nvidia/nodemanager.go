package nvidia

import (
	"os"

	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"fmt"
)

var (
	clientset *kubernetes.Clientset
	nodeName  string
)

func init() {
	kubeInit()
}

func kubeInit() {
	kubeconfigFile := os.Getenv("KUBECONFIG")
	var err error
	var config *rest.Config

	if _, err = os.Stat(kubeconfigFile); err != nil {
		log.V(5).Infof("kubeconfig %s failed to find due to %v", kubeconfigFile, err)
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Failed due to %v", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigFile)
		if err != nil {
			log.Fatalf("Failed due to %v", err)
		}
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed due to %v", err)
	}

	nodeName = os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatalln("Please set env NODE_NAME")
	}
}

func patchGPUTopology(topology gpuTopology) error {
	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})

	if err != nil {
		return err
	}

	newNode := node.DeepCopy()
	for gpu1, temp := range topology {
		for gpu2, topo := range temp {
			envGsocGpuTopology := ENV_GOSC_GPU_TOPOLOGY_PRIFIX + fmt.Sprintf("_%d",gpu1 )+ fmt.Sprintf("_%d",gpu2)
			newNode.ObjectMeta.Annotations[envGsocGpuTopology] = fmt.Sprint(uint(topo))
		}
	}

	_, err = clientset.CoreV1().Nodes().Update(newNode)
	if err != nil {
		log.Infof("Failed to fetch node gpu annotation %s.", topology)
	} else {
		log.Infof("Success in update node gpu annotation %s.", topology)
	}

	return err
}
