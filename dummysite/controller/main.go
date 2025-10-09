package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	dummySiteGVR = schema.GroupVersionResource{
		Group:    "codegeek.com",
		Version:  "v1",
		Resource: "dummysites",
	}
)

type Controller struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	informer      cache.SharedIndexInformer
}

func NewController(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) *Controller {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return dynamicClient.Resource(dummySiteGVR).Namespace(corev1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return dynamicClient.Resource(dummySiteGVR).Namespace(corev1.NamespaceAll).Watch(context.TODO(), options)
			},
		},
		&unstructured.Unstructured{},
		time.Minute*10,
		cache.Indexers{},
	)

	controller := &Controller{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		informer:      informer,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.handleAdd,
		UpdateFunc: controller.handleUpdate,
		DeleteFunc: controller.handleDelete,
	})

	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer klog.Info("Shutting down controller")

	klog.Info("Starting DummySite controller")
	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		klog.Error("Timed out waiting for cache sync")
		return
	}

	klog.Info("Controller synced and ready")
	<-stopCh
}

func (c *Controller) handleAdd(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	klog.Infof("DummySite added: %s/%s", u.GetNamespace(), u.GetName())
	c.reconcile(u)
}

func (c *Controller) handleUpdate(oldObj, newObj interface{}) {
	u := newObj.(*unstructured.Unstructured)
	klog.Infof("DummySite updated: %s/%s", u.GetNamespace(), u.GetName())
	c.reconcile(u)
}

func (c *Controller) handleDelete(obj interface{}) {
	u := obj.(*unstructured.Unstructured)
	klog.Infof("DummySite deleted: %s/%s", u.GetNamespace(), u.GetName())
	// Kubernetes will handle cascade deletion of owned resources
}

func (c *Controller) reconcile(obj *unstructured.Unstructured) {
	ctx := context.Background()
	name := obj.GetName()
	namespace := obj.GetNamespace()

	// Extract website_url from spec
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		klog.Errorf("Failed to get spec: %v", err)
		return
	}

	websiteURL, found, err := unstructured.NestedString(spec, "website_url")
	if err != nil || !found {
		klog.Errorf("Failed to get website_url: %v", err)
		return
	}

	klog.Infof("Reconciling DummySite %s/%s with URL: %s", namespace, name, websiteURL)

	// Fetch HTML content
	htmlContent, err := c.fetchHTML(websiteURL)
	if err != nil {
		klog.Errorf("Failed to fetch HTML: %v", err)
		c.updateStatus(ctx, namespace, name, "Error", "")
		return
	}

	// Create or update ConfigMap with HTML content
	if err := c.ensureConfigMap(ctx, namespace, name, htmlContent, obj.GetUID()); err != nil {
		klog.Errorf("Failed to ensure ConfigMap: %v", err)
		return
	}

	// Create or update Deployment
	if err := c.ensureDeployment(ctx, namespace, name, obj.GetUID()); err != nil {
		klog.Errorf("Failed to ensure Deployment: %v", err)
		return
	}

	// Create or update Service
	if err := c.ensureService(ctx, namespace, name, obj.GetUID()); err != nil {
		klog.Errorf("Failed to ensure Service: %v", err)
		return
	}

	// Create or update Ingress (optional)
	if err := c.ensureIngress(ctx, namespace, name, obj.GetUID()); err != nil {
		klog.Errorf("Failed to ensure Ingress: %v", err)
		return
	}

	// Update status
	serviceURL := fmt.Sprintf("http://%s.%s.svc.cluster.local", name, namespace)
	c.updateStatus(ctx, namespace, name, "Ready", serviceURL)
}

func (c *Controller) fetchHTML(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *Controller) ensureConfigMap(ctx context.Context, namespace, name, content string, ownerUID types.UID) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-html",
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "codegeek.com/v1",
					Kind:       "DummySite",
					Name:       name,
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
		},
		Data: map[string]string{
			"index.html": content,
		},
	}

	_, err := c.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, configMap.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	_, err = c.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	return err
}

func (c *Controller) ensureDeployment(ctx context.Context, namespace, name string, ownerUID types.UID) error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "codegeek.com/v1",
					Kind:       "DummySite",
					Name:       name,
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:alpine",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "html",
									MountPath: "/usr/share/nginx/html",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "html",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name + "-html",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (c *Controller) ensureService(ctx context.Context, namespace, name string, ownerUID types.UID) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "codegeek.com/v1",
					Kind:       "DummySite",
					Name:       name,
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	_, err := c.clientset.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	_, err = c.clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
	return err
}

func (c *Controller) ensureIngress(ctx context.Context, namespace, name string, ownerUID types.UID) error {
	pathTypePrefix := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "codegeek.com/v1",
					Kind:       "DummySite",
					Name:       name,
					UID:        ownerUID,
					Controller: boolPtr(true),
				},
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.codegeek.com", name),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, ingress.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = c.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	}

	_, err = c.clientset.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	return err
}

func (c *Controller) updateStatus(ctx context.Context, namespace, name, state, url string) {
	statusMap := map[string]interface{}{
		"state": state,
		"url":   url,
	}

	obj, err := c.dynamicClient.Resource(dummySiteGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get DummySite for status update: %v", err)
		return
	}

	if err := unstructured.SetNestedMap(obj.Object, statusMap, "status"); err != nil {
		klog.Errorf("Failed to set status: %v", err)
		return
	}

	_, err = c.dynamicClient.Resource(dummySiteGVR).Namespace(namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update status: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create dynamic client: %v", err)
	}

	controller := NewController(clientset, dynamicClient)

	stopCh := make(chan struct{})
	defer close(stopCh)

	controller.Run(stopCh)
}
