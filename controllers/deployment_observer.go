package controllers

import (
	"context"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)




type DeploymentObserver struct {
	Observer   //匿名字段
	Deployment *appsv1.Deployment
}


func (co *DeploymentObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

	log := r.Log.WithValues("func", "observeDeployment")

	deployment := &appsv1.Deployment{}
	instance := &webhookv1.WebHook{}

	isRunAsRoot := false
	pIsRunAsRoot := &isRunAsRoot //bool type pointer


	imageName := webHook.Spec.DockerRegistryPrefix + "/audit-webhook:v0.1.0"

	//循环监听
	for {

		// Query through client tools
		err := r.Get(ctx, req.NamespacedName, instance)
		if err != nil {

			// If there is no instance, an empty result is returned, so that the external party will not call the Reconcile method immediately
			if errors.IsNotFound(err) {
				//log.Info("2.1. instance not found, maybe removed")
				break
			}

			log.Error(err, "2.2 error")
			// Return error message to the outside
			break
		}

		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "audit-webhook-server"}, deployment)

		// If there is no error in the query result, it proves that the secret is normal, and nothing is done
		if err == nil {
			//log.Info("deployment exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		// If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			log.Error(err, "query deployment error")
		}

		// Instantiate a data structure
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webHook.Namespace,
				Name:      "audit-webhook-server",
				Labels: map[string]string{
					"app": APP_NAME,
				},
			},
			Spec: appsv1.DeploymentSpec{
				// The number of copies is calculated
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": APP_NAME,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": APP_NAME,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image:           imageName,
							ImagePullPolicy: "IfNotPresent",
							Name:            APP_NAME,
							Command:         []string{"/audit-webhook"},
							Ports: []corev1.ContainerPort{{
								ContainerPort: 8081,
							}},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_REQUEST),
									"memory": resource.MustParse(MEM_REQUEST),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse(CPU_LIMIT),
									"memory": resource.MustParse(MEM_LIMIT),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot: pIsRunAsRoot,
							},
							Env: []corev1.EnvVar{
								{
									Name: "VOLUME_PATCH",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "audit-webhook-configmap",
											},
											Key: "volume_patch",
										},
									},
								},
								{
									Name: "CONTAINER_PATCH",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "audit-webhook-configmap",
											},
											Key: "container_patch",
										},
									},

								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: "/certs",
									Name: "certs",
									ReadOnly: false,
								},
							},
						}},
						ImagePullSecrets: webHook.Spec.ImagePullSecrets,
						Volumes: []corev1.Volume{
							{
								Name: "certs",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "audit-webhook-tls-secret",
									},
								},
							},
						},
					},
				},
			},
		}

		// This step is very critical!
		// After the association is established, the service will also be deleted when the elasticweb resource is deleted
		//log.Info("set reference")
		if err := controllerutil.SetControllerReference(webHook, deployment, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference error")
		}

		// 创建deployment
		//log.Info("start create deployment")
		if err := r.Create(ctx, deployment); err != nil {
			log.Error(err, "create deployment error")
		}

		log.Info("create deployment success")


		// Update the status if the creation is successful
		if err = updateStatus(ctx, r, instance,req); err != nil {
			log.Error(err, "error")
		}

		log.Info("create secret success")
		time.Sleep(5000 * time.Millisecond)

	}


}





// After processing the pod, update the latest status
func  updateStatus(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) error {
	log := r.Log.WithValues("webhook", req.NamespacedName)
	// Update the WebHook status with the pod names
	// List the pods for this WebHook's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(webHook.Namespace),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "WebHook.Namespace", webHook.Namespace)
		return  err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, webHook.Status.Nodes) {
		webHook.Status.Nodes = podNames
		err := r.Status().Update(ctx, webHook)
		if err != nil {
			log.Error(err, "Failed to update WebHook's status")
			return  err
		}
	}

	return nil
}



// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}