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
	"github.com/go-logr/logr"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/bootstrap"
	"github.com/youngpig1998/webhook-operator/internal/operator"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkpolicy "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)




// WebHookReconciler reconciles a WebHook object
type WebHookReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	//Observes [6] Observe
	Config *rest.Config
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
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *WebHookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("auditwebhook", req.NamespacedName)


	log.Info("1. start reconcile logic")
	// Instantialize the data structure
	instance := &webhookv1.WebHook{}

	//First,query the webhook instance
	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {
		// If there is no instance, an empty result is returned, so that the Reconcile method will not be called immediately
		if errors.IsNotFound(err) {
			log.Info("Instance not found, maybe removed")
			return reconcile.Result{}, nil
		}
		log.Error(err, "query action happens error")
		// Return error message
		return ctrl.Result{}, err
	}



	//Set the bootstrapClient's owner value as the webhook,so the resources we create then will be set reference to the webhook
	//when the webhook cr is deleted,the resources(such as deployment.configmap,issuer...) we create will be deleted too
	bootstrapClient, err := bootstrap.NewClient(r.Config,r.Scheme,instance)
	if err != nil {
		log.Error(err, "failed to initialise bootstrap client")
		return ctrl.Result{}, err
	}



	//We create networkpolicy first
	networkPolicyName, networkPolicy := operator.NetworkPolicy()
	err = bootstrapClient.CreateResource(networkPolicyName, networkPolicy)
	if err != nil {
		log.Error(err, "failed to create operator NetworkPolicy", "Name", networkPolicyName)
		return ctrl.Result{}, err
	}



	secretName, secret := operator.Secret(instance)
	err = bootstrapClient.CreateResource(secretName, secret)
	if err != nil {
		log.Error(err, "failed to create operator secret", "Name", secretName)
		return ctrl.Result{}, err
	}


	configMapName, configMap := operator.ConfigMap(instance)
	err = bootstrapClient.CreateResource(configMapName, configMap)
	if err != nil {
		log.Error(err, "failed to create operator configMap", "Name", configMapName)
		return ctrl.Result{}, err
	}


	serviceName, service := operator.Service()
	err = bootstrapClient.CreateResource(serviceName, service)
	if err != nil {
		log.Error(err, "failed to create operator service", "Name", serviceName)
		return ctrl.Result{}, err
	}

	deploymentName, deployment := operator.Deployment(instance)
	err = bootstrapClient.CreateResource(deploymentName, deployment)
	if err != nil {
		log.Error(err, "failed to create operator Deployment", "Name", deploymentName)
		return ctrl.Result{}, err
	}

	mcName, mc := operator.MutatingWebhookConfiguration(instance)
	err = bootstrapClient.CreateResource(mcName, mc)
	if err != nil {
		log.Error(err, "failed to create operator Mc", "Name", mcName)
		return ctrl.Result{}, err
	}


	return ctrl.Result{}, nil
	
}




// SetupWithManager sets up the controller with the Manager.
func (r *WebHookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.WebHook{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&admissionregistrationv1beta1.MutatingWebhookConfiguration{}).
		Owns(&networkpolicy.NetworkPolicy{}).
		Complete(r)
}









