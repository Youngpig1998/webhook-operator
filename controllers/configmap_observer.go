package controllers

import (
	"context"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)




type CmObserver struct {
	Observer //匿名字段
	Cm       *corev1.ConfigMap
}


func (co *CmObserver) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

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

