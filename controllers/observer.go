package controllers

import (
	"context"
	"github.com/go-logr/logr"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
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

type Observe interface {
	Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request)
}



type Observer struct {
	Log    logr.Logger
}



//Observer实现Observe接口
func (observer *Observer) Update(ctx context.Context, r *WebHookReconciler, webHook *webhookv1.WebHook, req ctrl.Request) {

}


