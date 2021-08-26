package controllers

import (
	"context"
	b64 "encoding/base64"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)




type MutatingWebhookConfigObserver struct {
	Observer //匿名字段
	Mc       *admissionregistrationv1beta1.MutatingWebhookConfiguration
}


func (co *MutatingWebhookConfigObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

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
			//log.Info("MutatingWebhookConfiguration exists")
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

