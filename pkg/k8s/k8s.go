package k8s

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

type K8SClient struct {
	ClientSet kubernetes.Interface
}

func NewK8SClient() (*K8SClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("No cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return &K8SClient{
		ClientSet: clientset,
	}, nil
}

func NewFakeClient() (*K8SClient, error) {
	return &K8SClient{
		ClientSet: fake.NewSimpleClientset(),
	}, nil
}

func GetK8SToken() ([]byte, error) {
	const tokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// is checking for the file needed or ReadFile would return an appropriate error?
	_, err := os.Stat(tokenFile)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (c *K8SClient) GetPodStatus(namespace, selector string) (*v1.PodStatus, error) {
	pods, err := c.ClientSet.CoreV1().Pods(namespace).List(context.TODO(),
		metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	if len((*pods).Items) == 0 {
		return nil, fmt.Errorf("no pods for selector")
	}
	if len((*pods).Items) != 1 {
		return nil, fmt.Errorf("multiple (%d) pods chosen by selector", len((*pods).Items))
	}
	return &(*pods).Items[0].Status, nil
}
