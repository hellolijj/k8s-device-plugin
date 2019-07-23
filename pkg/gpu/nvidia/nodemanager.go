package nvidia

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"encoding/json"

	log "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
	envGPUTopologyMap := map[string]string{}
	for gpu1, temp := range topology {
		for gpu2, topo := range temp {
			if gpu1 < gpu2 {
				envGPUTopologyMap[ENV_GPU_TOPOLOGY_PRIFIX+fmt.Sprintf("_%s", topo.Abbreviation())+fmt.Sprintf("_%d", gpu1)+fmt.Sprintf("_%d", gpu2)] = topo.String()
			}
		}
	}

	// no gpu topo
	if len(envGPUTopologyMap) == 0 {
		return nil
	}

	envGPUTopologyJson, err := json.Marshal(envGPUTopologyMap)
	
	if err != nil {
		log.Infof("invalid gpu topology map %v", envGPUTopologyMap)
		return err
	}

	log.Infof("gpu topology json %v", string(envGPUTopologyJson))
	newNode.ObjectMeta.Annotations[EnvAnnotationKey] = string(envGPUTopologyJson)

	_, err = clientset.CoreV1().Nodes().Update(newNode)
	if err != nil {
		log.Infof("Failed to fetch node gpu annotation %s.", topology)
	} else {
		log.Infof("Success in update node gpu annotation %s.", topology)
	}

	return err
}

func patchNodeType() error {
	
	// get note tpye
	url := "http://100.100.100.200/latest/meta-data/instance/instance-type"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	nodeType := string(body)
	
	log.Infof("fetch node type %v", nodeType)
	
	// update node type
	node, err := clientset.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	
	newNode := node.DeepCopy()
	newNode.ObjectMeta.Annotations[EnvNodeType] = nodeType
	_, err = clientset.CoreV1().Nodes().Update(newNode)
	if err != nil {
		log.Infof("Failed to fetch node type %s.", nodeType)
	} else {
		log.Infof("Success in update node type %s.", nodeType)
	}
	
	return err
}