package controllers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type CmObserver struct {
	Log    logr.Logger
	Cm *corev1.ConfigMap
}


func (co *CmObserver) Observe(req ctrl.Request) {
	//定义log对象
	log := co.Log.WithValues("webhook", req.NamespacedName)

	for {

		log.Info("its now oberving the service")
		time.Sleep(1000 * time.Millisecond)
	}

}

