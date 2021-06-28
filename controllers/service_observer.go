package controllers

import (
	"context"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)



type ServiceObserver struct {
	Observer //匿名字段
	Service  *corev1.Service
}


func (co *ServiceObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

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

