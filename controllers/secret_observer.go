package controllers

import (
	"context"
	b64 "encoding/base64"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)





type SecretObserver struct {
	Observer //匿名字段
	Secret   *corev1.Secret
}


func (co *SecretObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

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

