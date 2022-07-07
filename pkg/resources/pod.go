package resources

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

var podLog = ctrl.Log.WithName("pod")

// ListPods returns a list of pods in the namespace
func ListPods(namespace string) ([]v1.Pod, error) {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	podList := &v1.PodList{}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		podList, err = clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

// PodStatus returns the status of a pod
func PodStatus(name string, namespace string) (v1.PodPhase, error) {
	pod, err := GetPod(name, namespace)
	if err != nil {
		return v1.PodUnknown, err
	}
	return pod.Status.Phase, nil
}

// GetPod returns a pods in the namespace
func GetPod(name string, namespace string) (*v1.Pod, error) {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	pod := &v1.Pod{}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		pod, err = clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return pod, nil
}

// // UpdatePod updates a pods in the namespace
// func UpdatePod(pod *v1.Pod) (*v1.Pod, error) {
// 	clientset, err := clientset()
// 	if err != nil {
// 		podLog.Error(err, "error getting client for cluster")
// 		return nil, err
// 	}
// 	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
// 		pod, err = clientset.CoreV1().Pods(pod.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
// 		if err != nil {
// 			podLog.Info(fmt.Sprintf("error updating pods: %s", err))
// 			return err
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return pod, nil
// }

// RemovePodLabels removes the labels from a pod
func RemovePodLabels(name string, namespace string, labels []string) error {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return err
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}

		for _, v := range pod.ObjectMeta.Labels {
			delete(pod.ObjectMeta.Labels, v)
		}

		_, err = clientset.CoreV1().Pods(namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}
		return nil
	})
	return err
}

// UpdatePodLabel updates a pods labels
func UpdatePodLabel(name string, namespace string, key string, value string) error {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return err
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}

		if val, ok := pod.ObjectMeta.Labels[key]; ok {
			if val != value {
				pod.ObjectMeta.Labels[key] = value
			}
		}

		_, err = clientset.CoreV1().Pods(namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
		if err != nil {
			podLog.Info(fmt.Sprintf("error listing pods: %s", err))
			return err
		}
		return nil
	})
	return err
}

// UpdatePod updates a pods
func UpdatePod(pod *v1.Pod) (*v1.Pod, error) {
	clientset, err := clientset()
	if err != nil {
		podLog.Error(err, "error getting client for cluster")
		return nil, err
	}
	ret, err := clientset.CoreV1().Pods(pod.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
	if err != nil {
		podLog.Info(fmt.Sprintf("error listing pods: %s", err))
		return nil, err
	}
	return ret, nil
}
