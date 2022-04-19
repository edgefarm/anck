package resources

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodeLabels returns the labels of the node
func GetNodeLabels(name string) (map[string]string, error) {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node.Labels, nil
}
