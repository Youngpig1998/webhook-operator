/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	b64 "encoding/base64"
	"github.com/go-logr/logr"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)




const (
	// APP tag name in deployment
	APP_NAME = "audit-webhook"
	// CPU resource application for a single POD
	CPU_REQUEST = "300m"
	// Upper limit of CPU resources of a single POD
	CPU_LIMIT = "500m"
	// Memory resource application for a single POD
	MEM_REQUEST = "100Mi"
	// Upper limit of memory resources of a single POD
	MEM_LIMIT = "200Mi"

)



// WebHookReconciler reconciles a WebHook object
type WebHookReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Co     CmObserver
}

// +kubebuilder:rbac:groups=webhook.example.com,resources=webhooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webhook.example.com,resources=webhooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webhook.example.com,resources=webhooks/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete

func (r *WebHookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("webhook", req.NamespacedName)

	// your logic here

	log.Info("1. start reconcile logic")

	// Instantiate data structure
	instance := &webhookv1.WebHook{}

	// Query through client tools
	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {

		// If there is no instance, an empty result is returned, so that the external party will not call the Reconcile method immediately
		if errors.IsNotFound(err) {
			log.Info("2.1. instance not found, maybe removed")
			return reconcile.Result{}, nil
		}

		log.Error(err, "2.2 error")
		// Return error message to the outside
		return ctrl.Result{}, err
	}


	go observeSecret(ctx,r,instance,req)
	go observeConfigmap(ctx,r,instance,req)
	go observeService(ctx,r,instance,req)
	go observeDeployment(ctx,r,instance,req)
	go observeMutatingWebhookConfiguration(ctx,r,instance,req)


	return ctrl.Result{}, nil
	
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebHookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.WebHook{}).
		Complete(r)
}




//Observe service
func observeService(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)  {
	log := r.Log.WithValues("func", "observeService")

	instance := &webhookv1.WebHook{}
	service := &corev1.Service{}
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



		//err := r.Get(ctx, req.NamespacedName, configmap)
		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "audit-webhook-service"}, service)

		// If there is no error in the query result, it proves that the service is normal, and nothing is done
		if err == nil {
			//log.Info("service exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		//If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			log.Error(err, "query service error")
		}

		// Instantiate a data structure
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webHook.Namespace,
				Name:      "audit-webhook-service",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Port:     443,
					TargetPort: intstr.IntOrString{
						IntVal: 8081,
						StrVal: "8081",
					},
					Protocol: corev1.ProtocolTCP,
				},
				},
				Selector: map[string]string{
					"app": APP_NAME,
				},
			},
		}

		// This step is very critical！
		// After the association is established, the service will be deleted when the webhook resource is deleted
		//log.Info("set reference")
		if err := controllerutil.SetControllerReference(webHook, service, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference error")
		}

		//  Create service
		log.Info("start create service")
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "create service error")
		}

		log.Info("create service success")
		time.Sleep(5000 * time.Millisecond)

	}

}




//Observe configmap
func observeConfigmap(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)  {
	log := r.Log.WithValues("func", "observeConfigmap")

	instance := &webhookv1.WebHook{}
	configmap := &corev1.ConfigMap{}
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



		//err := r.Get(ctx, req.NamespacedName, configmap)
		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "audit-webhook-configmap"}, configmap)

		// If there is no error in the query result, it proves that the configmap is normal, and nothing is done
		if err == nil {
			//log.Info("configmap exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		// If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			log.Error(err, "query configmap error")
		}

		imageName := webHook.Spec.DockerRegistryPrefix + "/fluent:1.10-plugin-script"
		volume_patch := "{\"name\":\"internal-tls\",\"secret\":{\"secretName\":\"internal-tls\",\"defaultMode\":420}}"
		container_patch := "{\"name\":\"sidecar\",\"image\":\"" + imageName  + "\",\"securityContext\":{\"runAsNonRoot\":true},\"resources\":{\"requests\":{\"memory\":\"100Mi\",\"cpu\":\"100m\"},\"limits\":{\"memory\":\"250Mi\",\"cpu\":\"250m\"}},\"imagePullPolicy\":\"IfNotPresent\",\"args\":[\"/bin/bash\",\"-c\",\"fluentd -c /fluentd/etc/fluent.conf\"],\"volumeMounts\":[{\"name\":\"varlog\",\"mountPath\":\"/var/log\"}],\"env\":[{\"name\":\"NS_DOMAIN\",\"value\":\""+ webHook.Namespace   + "\"}]}"


		//Instantiate a data structure
		configmap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webHook.Namespace,
				Name:      "audit-webhook-configmap",
			},
			Data: map[string]string{
				"volume_patch": volume_patch,
				"container_patch": container_patch,
			},
		}

		// This step is very critical!
		// After the association is established, the service will also be deleted when the elasticweb resource is deleted
		//log.Info("set reference")
		if err := controllerutil.SetControllerReference(webHook, configmap, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference error")
		}

		// Create configmap
		log.Info("start create configmap")
		if err := r.Create(ctx, configmap); err != nil {
			log.Error(err, "create configmap error")
		}

		log.Info("create configmap success")
		time.Sleep(5000 * time.Millisecond)


	}

}




//Observe secret
func observeSecret(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)  {
	log := r.Log.WithValues("func", "observeSecret")

	secret := &corev1.Secret{}
	instance := &webhookv1.WebHook{}
	secretType := corev1.SecretTypeTLS
	//Decoding
	sDecForCrt, _ := b64.StdEncoding.DecodeString(webHook.Spec.TlsCert)
	sDecForKey, _ := b64.StdEncoding.DecodeString(webHook.Spec.TlsKey)


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

		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "audit-webhook-tls-secret"}, secret)


		// If there is no error in the query result, it proves that the secret is normal, and nothing is done
		if err == nil {
			//log.Info("secret exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		// If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			log.Error(err, "query secret error")
		}

		// Instantiate a data structure
		secret = &corev1.Secret{
			Type: secretType,
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webHook.Namespace,
				Name:      "audit-webhook-tls-secret",
				Labels: map[string]string{
					"app": APP_NAME,
				},
			},
			Data: map[string][]byte{
				"tls.crt": sDecForCrt,
				"tls.key": sDecForKey,
			},
		}

		// This step is very critical!
		// After the association is established, the service will also be deleted when the elasticweb resource is deleted
		//log.Info("set reference")
		if err := controllerutil.SetControllerReference(webHook, secret, r.Scheme); err != nil {
			log.Error(err, "SetControllerReference error")
		}

		// Create secret
		//log.Info("start create secret")
		if err := r.Create(ctx, secret); err != nil {
			log.Error(err, "create secret error")
		}

		log.Info("create secret success")
		time.Sleep(5000 * time.Millisecond)

	}

}



//Observe deployment
func observeDeployment(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)  {
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
							Command: []string{"/audit-webhook"},
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
func updateStatus(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) error {
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



//Observe MutatingWebhookConfiguration
func observeMutatingWebhookConfiguration(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)  {
	log := r.Log.WithValues("func", "observeMutatingWebhookConfiguration")

	instance := &webhookv1.WebHook{}
	mc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
	path := "/add-sidecar"


	failurePolicy := new(admissionregistrationv1beta1.FailurePolicyType)
	*failurePolicy  = admissionregistrationv1beta1.Ignore


	matchPolicy  := new(admissionregistrationv1beta1.MatchPolicyType)
	*matchPolicy =  admissionregistrationv1beta1.Equivalent

	scope  := new(admissionregistrationv1beta1.ScopeType)
	*scope = admissionregistrationv1beta1.NamespacedScope

	sDecForCABundle, _ := b64.StdEncoding.DecodeString(webHook.Spec.CaBundle)
	for {
		// Query through client tools
		err := r.Get(ctx, req.NamespacedName, instance)
		if err != nil {

			// If there is no instance, an empty result is returned, so that the external party will not call the Reconcile method immediately
			if errors.IsNotFound(err) {
				break
			}

			log.Error(err, "2.2 error")
			// Return error message to the outside
			break
		}

		//err := r.Get(ctx, req.NamespacedName, configmap)
		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "audit-webhook-config"}, mc)

		// If there is no error in the query result, it proves that the configmap is normal, and nothing is done
		if err == nil {
			log.Info("MutatingWebhookConfiguration exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}
		// If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			log.Error(err, "query MutatingWebhookConfiguration error")
		}

		// Instantiate a data structure
		mc = &admissionregistrationv1beta1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: webHook.Namespace,
				Name:      "audit-webhook-config",
			},
			Webhooks: []admissionregistrationv1beta1.MutatingWebhook{{
				Name:      "audit.watson.org",
				MatchPolicy: matchPolicy,
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cp4d-audit": "yes",
					},
				},
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups: []string{""},
						APIVersions: []string{"v1"},
						Resources: []string{"pods"},
						Scope: scope,
					},
				},
				},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Name: "audit-webhook-service",
						Namespace: webHook.Namespace,
						Path: &path,
						Port: pointer.Int32Ptr(443),
					},
					CABundle: sDecForCABundle,
				},
				FailurePolicy: failurePolicy,

			},
			},
		}

		// This step is very critical!
		// After the association is established, the service will also be deleted when the elasticweb resource is deleted
		//log.Info("set reference")
		if err := controllerutil.SetControllerReference(webHook, mc, r.Scheme); err != nil {
			//log.Error(err, "SetControllerReference error")
		}

		// 创建MutatingWebhookConfiguration
		//log.Info("start create MutatingWebhookConfiguration")
		if err := r.Create(ctx, mc); err != nil {
			//log.Error(err, "create MutatingWebhookConfiguration error")
		}

		//log.Info("create MutatingWebhookConfiguration success")
		time.Sleep(5000 * time.Millisecond)


	}

}




// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

