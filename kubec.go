package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func createClientset(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	return clientset, nil
}

func getCertFromKubernetes(clientset *kubernetes.Clientset, secretName, namespace string) (string, string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", fmt.Errorf("error getting secret %v in namespace %v %v", secretName, namespace, err)
	}
	myKey := removeTrailingLineBreak(string(secret.Data["tls.key"]))
	myCert := removeTrailingLineBreak(string(secret.Data["tls.crt"]))
	return myKey, myCert, nil
}
