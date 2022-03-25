package resources

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListPods returns a list of pods in the namespace
func ListPods(namespace string) ([]v1.Pod, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		resourcesLog.Info(fmt.Sprintf("error listing pods: %s", err))
		return nil, err
	}

	return podList.Items, nil
}

// PodStatus returns the status of a pod
func PodStatus(pod string) (v1.PodPhase, error) {
	clientset, err := clientset()
	if err != nil {
		resourcesLog.Error(err, "error getting client for cluster")
		return "", err
	}
	podStatus, err := clientset.CoreV1().Pods(pod).Get(context.Background(), pod, metav1.GetOptions{})
	if err != nil {
		resourcesLog.Info(fmt.Sprintf("error getting pod: %s", err))
		return "", err
	}

	return podStatus.Status.Phase, nil
}
