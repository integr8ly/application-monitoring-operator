package common

import (
	"bytes"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	errorUtil "github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("pods")


type PodCommander interface {
	ExecIntoPod(dpl *appsv1.Deployment, cmd string) error
}

type OpenShiftPodCommander struct {
	ClientSet *kubernetes.Clientset
}

func (pc *OpenShiftPodCommander) ExecIntoPod(ss *appsv1.StatefulSet, cmd string, container string) error {
	toRun := []string{"/bin/bash", "-c", cmd}
	podName, err := getStatefulSetPod(pc.ClientSet, ss)
	if err != nil {
		return err
	}
	if _, stderr, err := runExec(pc.ClientSet, toRun, podName, ss.Namespace, container); err != nil {
		return errorUtil.Wrapf(err, "failed to exec, %s", stderr)
	}
	return nil
}

// run exec command on pod
func runExec(cs *kubernetes.Clientset, command []string, pod, ns string, container string) (string, string, error) {
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(ns).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command: command,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}, scheme.ParameterCodec)

	cfg, _ := config.GetConfig()
	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return "", "", errorUtil.Wrap(err, "error while creating executor")
	}

	var stdout, stderr bytes.Buffer
	var stdin io.Reader

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}

func getStatefulSetPod(cl *kubernetes.Clientset, ss *appsv1.StatefulSet) (podName string, err error) {
	ns := ss.Namespace
	api := cl.CoreV1()
	listOptions := metav1.ListOptions{
		LabelSelector: "app=prometheus",
	}
	podList, _ := api.Pods(ns).List(listOptions)
	podListItems := podList.Items
	if len(podListItems) == 0 {
		return "", err
	}
	podName = podListItems[0].Name
	return podName, nil
}

func GetK8Client() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}
