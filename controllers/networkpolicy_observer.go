package controllers

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	extensionv1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

var rootNetYaml_front = `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: audit-webhook-networkpolicy
  namespace: `




var rootNetYaml_back = `
spec:
  podSelector:
    matchLabels:
      app: audit-webhook
  policyTypes:
  - Ingress
  ingress:
  - ports:
    - protocol: TCP
      port: 8081
`




type NetworkPolicyObserver struct {
	Observer //匿名字段
	NetworkPolicy   *extensionv1.NetworkPolicy
}


func (co *NetworkPolicyObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

	log := r.Log.WithValues("func", "observeNetworkPolicy")
	instance := &webhookv1.WebHook{}
	networkPolicy := &extensionv1.NetworkPolicy{}

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

		err = r.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: "selfsigned-issuer"}, networkPolicy)

		// If there is no error in the query result, it proves that the service is normal, and nothing is done
		if err == nil {
			//log.Info("service exists")
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		//If the error is not NotFound, return an error
		if !errors.IsNotFound(err) {
			//log.Error(err, "query service error")
		}


		var rootNetYaml = rootNetYaml_front + req.NamespacedName.Namespace + rootNetYaml_back

		if err := r.createFromYaml(webHook, []byte(rootNetYaml)); err != nil {
			r.Log.Error(err, "create networkpolicy fail")
		}

		//log.Info("create networkpolicy success")
		time.Sleep(5000 * time.Millisecond)

	}



}



func (r *WebHookReconciler) createFromYaml(instance *webhookv1.WebHook, yamlContent []byte) error {
	obj := &unstructured.Unstructured{}
	jsonSpec, err := yaml.YAMLToJSON(yamlContent)
	if err != nil {
		return fmt.Errorf("could not convert yaml to json: %v", err)
	}

	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		return fmt.Errorf("could not unmarshal resource: %v", err)
	}

	obj.SetNamespace(instance.Namespace)

	// Set CommonServiceConfig instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, obj, r.Scheme); err != nil {
		return err
	}

	err = r.Client.Create(context.TODO(), obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("could not Create resource: %v", err)
	}

	return nil
}

