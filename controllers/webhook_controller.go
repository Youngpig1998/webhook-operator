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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)




// WebHookReconciler reconciles a WebHook object
type WebHookReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Observes [6] Observe
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




	Working(r.Observes,ctx,r,instance,req)




	return ctrl.Result{}, nil
	
}




func Working(ob [6]Observe,ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {
	var i int
	for i = 0; i < 6; i++ {
		go ob[i].Update(ctx,r,webHook,req)
	}
}




// SetupWithManager sets up the controller with the Manager.
func (r *WebHookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webhookv1.WebHook{}).
		Complete(r)
}









