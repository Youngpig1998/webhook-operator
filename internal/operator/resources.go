package operator

import (
	b64 "encoding/base64"
	odlmv1alpha1 "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha1"
	webhookv1 "github.com/youngpig1998/webhook-operator/api/v1"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/certificates"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/configmaps"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/deployments"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/issuers"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/mutatingwebhookconfigurations"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/networkpolicies"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/secrets"
	"github.com/youngpig1998/webhook-operator/iaw-shared-helpers/pkg/resources/services"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkpolicy "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"strings"
)



const (
	APP_NAME = "audit-webhook"
	//Request CPU resource of a single pod
	CPU_REQUEST = "300m"
	//Request CPU resource limit of a single pod
	CPU_LIMIT = "500m"
	//Request Memory resource of a single pod
	MEM_REQUEST = "100Mi"
	//Request Memory resource limit of a single pod
	MEM_LIMIT = "200Mi"
)


var (
	operandRequestName = "ibm-certmanager-operators"
	networkPolicyName = "audit-webhook-networkpolicy"
	issuerName = "selfsigned-issuer"
	certificateName = "serving-cert"
	secretName = "audit-webhook-tls-secret"
	configMapName = "audit-webhook-configmap"
	serviceName = "audit-webhook-service"
	mutatingwebhookConfigurationName = "audit-webhook-config"
	deploymentName = "audit-webhook-server"
	commonservices = []string{"ibm-cert-manager-operator"}
)





func OperandRequest() (string, *odlmv1alpha1.OperandRequest) {
	operands := []odlmv1alpha1.Operand{}
	for _, commonService := range commonservices {
		operands = append(operands, odlmv1alpha1.Operand{Name: commonService})
	}
	operandRequest := &odlmv1alpha1.OperandRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:   operandRequestName,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
		},
		Spec: odlmv1alpha1.OperandRequestSpec{
			Requests: []odlmv1alpha1.Request{
				{
					Registry:          "common-service",
					RegistryNamespace: "ibm-common-services",
					Operands:          operands,
				},
			},
		},
	}
	return operandRequestName, operandRequest
}


func NetworkPolicy() (string, resources.Reconcileable) {

	netProtocol := corev1.Protocol("TCP")

	networkPolicyIngress := []networkpolicy.NetworkPolicyIngressRule{
		{
			Ports: []networkpolicy.NetworkPolicyPort{
				{
					Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 8081},
					Protocol: &netProtocol,
				},
			},
			From: []networkpolicy.NetworkPolicyPeer{},
		},
	}

	networkPolicy := &networkpolicy.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:   networkPolicyName,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
			//Namespace: webHook.Namespace,
		},
		Spec: networkpolicy.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":   "audit-webhook",
				},
			},
			Ingress: networkPolicyIngress,
			PolicyTypes: []networkpolicy.PolicyType{"Ingress"},
		},
	}

	return networkPolicyName,networkpolicies.From(networkPolicy)
}

func Issuer() (string, resources.Reconcileable){

	issuer := &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       issuerName,
			Labels:		map[string]string{
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
		},
		Spec:       certmanagerv1.IssuerSpec{
			IssuerConfig: certmanagerv1.IssuerConfig{
				SelfSigned: &certmanagerv1.SelfSignedIssuer{},
			},
		},
	}

	return  issuerName,issuers.From(issuer)
}


func Certificate(webHook *webhookv1.WebHook) (string, resources.Reconcileable){

	const dnsNameFront = "audit-webhook-service."
	const dnsNameBack = ".svc"
	var dnsName = dnsNameFront + webHook.Namespace + dnsNameBack
	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       certificateName,
			Labels:		map[string]string{
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
		},
		Spec:       certmanagerv1.CertificateSpec{
			DNSNames:     []string{
				dnsName,
			},
			SecretName:   secretName,
			IssuerRef:    certmanagerv1.ObjectReference{
				Name:  issuerName,
				Kind:  "Issuer",
			},
		},
	}

	return certificateName,certificates.From(certificate)
}


func Secret(webHook *webhookv1.WebHook) (string, resources.Reconcileable){

	secretType := corev1.SecretTypeTLS
	//Decoding
	sDecForCrt, _ := b64.StdEncoding.DecodeString(webHook.Spec.TlsCert)
	sDecForKey, _ := b64.StdEncoding.DecodeString(webHook.Spec.TlsKey)


	secret := &corev1.Secret{
		Type: secretType,
		ObjectMeta: metav1.ObjectMeta{
			//Namespace: webHook.Namespace,
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

	return secretName,secrets.From(secret)
}


func ConfigMap(webHook *webhookv1.WebHook) (string, resources.Reconcileable) {

	imageName := "cp.stg.icr.io/cp/opencontent-fluentd:ruby-ubi"
	if len(strings.TrimSpace(webHook.Spec.DockerRegistryPrefix)) > 0 {
		imageName = webHook.Spec.DockerRegistryPrefix + "/opencontent-fluentd@sha256:d71c70d59540caead90cfb46c83ebafe55787078f73e48bf12558f73b997b17e"
	}

	volume_patch := "{\"name\":\"internal-tls\",\"secret\":{\"secretName\":\"internal-tls\",\"defaultMode\":420}}"
	container_patch := "{\"name\": \"sidecar\", 	\"image\": \"" + imageName + "\", 	\"securityContext\": { 		\"runAsNonRoot\": true 	}, 	\"resources\": { 		\"requests\": { 			\"memory\": \"100Mi\", 			\"cpu\": \"100m\" 		}, 		\"limits\": { 			\"memory\": \"250Mi\", 			\"cpu\": \"250m\" 		} 	}, 	\"imagePullPolicy\": \"Always\", 	\"volumeMounts\": [{ 		\"name\": \"varlog\", 		\"mountPath\": \"/var/log\" 	}, { 		\"name\": \"internal-tls\", 		\"mountPath\": \"/etc/internal-tls\" 	}], 	\"env\": [{ 		\"name\": \"NS_DOMAIN\", 		\"value\": \"https://zen-audit-svc." + webHook.Namespace + ":9880/records\" 	}] }"

	// Instantialize the data structure
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			//Namespace: webHook.Namespace,
			Name:      configMapName,
			Labels: map[string]string{
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
		},
		Data: map[string]string{
			"volume_patch":    volume_patch,
			"container_patch": container_patch,
		},
	}

	return  configMapName,configmaps.From(configmap)
}


func Service() (string, resources.Reconcileable) {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			//Namespace: webHook.Namespace,
			Name:      serviceName,
			Labels: map[string]string{
				"app":                          APP_NAME,
				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 443,
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

	return serviceName,services.From(service)
}

func MutatingWebhookConfiguration(webHook *webhookv1.WebHook) (string, resources.Reconcileable) {

	path := "/add-sidecar"

	failurePolicy := new(admissionregistrationv1beta1.FailurePolicyType)
	*failurePolicy = admissionregistrationv1beta1.Ignore

	matchPolicy := new(admissionregistrationv1beta1.MatchPolicyType)
	*matchPolicy = admissionregistrationv1beta1.Equivalent

	scope := new(admissionregistrationv1beta1.ScopeType)
	*scope = admissionregistrationv1beta1.NamespacedScope

	sDecForCABundle, _ := b64.StdEncoding.DecodeString(webHook.Spec.CaBundle)


	mc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			//Namespace: webHook.Namespace,
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




	//mc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
	//	ObjectMeta: metav1.ObjectMeta{
	//		//Namespace: webHook.Namespace,
	//		Name:      mutatingwebhookConfigurationName,
	//		Annotations: map[string]string{
	//			"certmanager.k8s.io/inject-ca-from": certpath + "",
	//		},
	//	},
	//	Webhooks: []admissionregistrationv1beta1.MutatingWebhook{{
	//		Name:        "audit.watson.org",
	//		MatchPolicy: matchPolicy,
	//		ObjectSelector: &metav1.LabelSelector{
	//			MatchLabels: map[string]string{
	//				"cp4d-audit": "yes",
	//			},
	//		},
	//		Rules: []admissionregistrationv1beta1.RuleWithOperations{{
	//			Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
	//			Rule: admissionregistrationv1beta1.Rule{
	//				APIGroups:   []string{""},
	//				APIVersions: []string{"v1"},
	//				Resources:   []string{"pods"},
	//				Scope:       scope,
	//			},
	//		},
	//		},
	//		ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
	//			Service: &admissionregistrationv1beta1.ServiceReference{
	//				Name:      serviceName,
	//				Namespace: webHook.Namespace,
	//				Path:      &path,
	//				Port:      pointer.Int32Ptr(443),
	//			},
	//			//CABundle: stringtoslicebyte(webHook.Spec.CaBundle),
	//			//CABundle: sDecForCABundle,
	//		},
	//		FailurePolicy: failurePolicy,
	//	},
	//	},
	//}

	return mutatingwebhookConfigurationName,mutatingwebhookconfigurations.From(mc)
}

func Deployment(webHook *webhookv1.WebHook) (string, resources.Reconcileable) {

	isRunAsRoot := false
	pIsRunAsRoot := &isRunAsRoot //bool pointer


	imageName := webHook.Spec.DockerRegistryPrefix + "/audit-webhook:v0.1.0"


	// Instantialize the data structure
	//deployment := &appsv1.Deployment{
	//	ObjectMeta: metav1.ObjectMeta{
	//		//Namespace: webHook.Namespace,
	//		Name:      deploymentName,
	//		Labels: map[string]string{
	//			"app":                          APP_NAME,
	//			"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
	//			"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
	//			"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
	//		},
	//	},
	//	Spec: appsv1.DeploymentSpec{
	//		// The replica is computed
	//		Replicas: pointer.Int32Ptr(1),
	//		Selector: &metav1.LabelSelector{
	//			MatchLabels: map[string]string{
	//				"app": APP_NAME,
	//			},
	//		},
	//		Template: corev1.PodTemplateSpec{
	//			ObjectMeta: metav1.ObjectMeta{
	//				Labels: map[string]string{
	//					"app":                          APP_NAME,
	//					"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
	//					"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
	//					"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
	//				},
	//				Annotations: map[string]string{
	//					"productName":    "ibm-auditwebhook",
	//					"productID":      "96808888679886798867988679886798",
	//					"productVersion": "1.0.0",
	//					"productMetric":  "VIRTUAL_PROCESSOR_CORE",
	//					"cloudpakId":     "96808888679886798867988679886798",
	//					"cloudpakName":   "Cloud Pak Open",
	//				},
	//			},
	//			Spec: corev1.PodSpec{
	//				Affinity: &corev1.Affinity{
	//					NodeAffinity: &corev1.NodeAffinity{
	//						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
	//							NodeSelectorTerms: []corev1.NodeSelectorTerm{
	//								corev1.NodeSelectorTerm{
	//									MatchExpressions: []corev1.NodeSelectorRequirement{
	//										corev1.NodeSelectorRequirement{
	//											Key:      "kubernetes.io/arch",
	//											Operator: "In",
	//											Values: []string{
	//												"amd64",
	//											},
	//										},
	//									},
	//								},
	//							},
	//						},
	//						PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
	//							corev1.PreferredSchedulingTerm{
	//								Weight: 3,
	//								Preference: corev1.NodeSelectorTerm{
	//									MatchExpressions: []corev1.NodeSelectorRequirement{
	//										corev1.NodeSelectorRequirement{
	//											Key:      "kubernetes.io/arch",
	//											Operator: "In",
	//											Values: []string{
	//												"amd64",
	//											},
	//										},
	//									},
	//								},
	//							},
	//						},
	//					},
	//				},
	//				HostNetwork:      false,
	//				HostIPC:          false,
	//				HostPID:          false,
	//				ImagePullSecrets: webHook.Spec.ImagePullSecrets,
	//				Containers: []corev1.Container{{
	//					Image: imageName,
	//					LivenessProbe: &corev1.Probe{
	//						Handler: corev1.Handler{
	//							HTTPGet: &corev1.HTTPGetAction{
	//								Path: "/add-sidecar",
	//								Port: intstr.IntOrString{
	//									Type:   intstr.Int,
	//									IntVal: 8081,
	//								},
	//								Scheme: "HTTPS",
	//							},
	//						},
	//						InitialDelaySeconds: 15,
	//						PeriodSeconds:       20,
	//					},
	//					ReadinessProbe: &corev1.Probe{
	//						Handler: corev1.Handler{
	//							HTTPGet: &corev1.HTTPGetAction{
	//								Path: "/add-sidecar",
	//								Port: intstr.IntOrString{
	//									Type:   intstr.Int,
	//									IntVal: 8081,
	//									StrVal: "8081",
	//								},
	//								Scheme: "HTTPS",
	//							},
	//						},
	//						InitialDelaySeconds: 5,
	//						PeriodSeconds:       10,
	//					},
	//					ImagePullPolicy: "IfNotPresent",
	//					Name:            APP_NAME,
	//					Command:         []string{"/audit-webhook"},
	//					Ports: []corev1.ContainerPort{{
	//						ContainerPort: 8081,
	//					}},
	//					Resources: corev1.ResourceRequirements{
	//						Requests: corev1.ResourceList{
	//							"cpu":    resource.MustParse(CPU_REQUEST),
	//							"memory": resource.MustParse(MEM_REQUEST),
	//						},
	//						Limits: corev1.ResourceList{
	//							"cpu":    resource.MustParse(CPU_LIMIT),
	//							"memory": resource.MustParse(MEM_LIMIT),
	//						},
	//					},
	//					SecurityContext: &corev1.SecurityContext{
	//						Capabilities: &corev1.Capabilities{
	//							Drop: []corev1.Capability{
	//								"ALL",
	//							},
	//						},
	//						RunAsNonRoot:             pIsRunAsRoot,
	//						AllowPrivilegeEscalation: pIsAllowPrivilegeEscalation,
	//						ReadOnlyRootFilesystem:   pIsReadOnlyRootFilesystem,
	//						Privileged:               pIsPrivileged,
	//					},
	//					Env: []corev1.EnvVar{
	//						{
	//							Name: "VOLUME_PATCH",
	//							ValueFrom: &corev1.EnvVarSource{
	//								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
	//									LocalObjectReference: corev1.LocalObjectReference{
	//										Name: "audit-webhook-configmap",
	//									},
	//									Key: "volume_patch",
	//								},
	//							},
	//						},
	//						{
	//							Name: "CONTAINER_PATCH",
	//							ValueFrom: &corev1.EnvVarSource{
	//								ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
	//									LocalObjectReference: corev1.LocalObjectReference{
	//										Name: "audit-webhook-configmap",
	//									},
	//									Key: "container_patch",
	//								},
	//							},
	//						},
	//					},
	//					VolumeMounts: []corev1.VolumeMount{
	//						{
	//							MountPath: "/certs",
	//							Name:      "certs",
	//							ReadOnly:  false,
	//						},
	//					},
	//				}},
	//				Volumes: []corev1.Volume{
	//					{
	//						Name: "certs",
	//						VolumeSource: corev1.VolumeSource{
	//							Secret: &corev1.SecretVolumeSource{
	//								SecretName: "audit-webhook-tls-secret",
	//							},
	//						},
	//					},
	//				},
	//			},
	//		},
	//	},
	//}

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
						Command:         []string{"/audit-webhook"},
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


	return deploymentName,deployments.From(deployment)
}





















































































//func NetworkPolicy(webHook *auditv1beta1.AuditWebhook) (*networkpolicy.NetworkPolicy) {
//
//
//	netProtocol := corev1.Protocol("TCP")
//
//	networkPolicyIngress := []networkpolicy.NetworkPolicyIngressRule{
//		{
//			Ports: []networkpolicy.NetworkPolicyPort{
//				{
//					Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 8081},
//					Protocol: &netProtocol,
//				},
//			},
//			From: []networkpolicy.NetworkPolicyPeer{},
//		},
//	}
//
//	networkPolicy := &networkpolicy.NetworkPolicy{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:   "audit-webhook-networkpolicy",
//			Labels: map[string]string{
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//			Namespace: webHook.Namespace,
//		},
//		Spec: networkpolicy.NetworkPolicySpec{
//			PodSelector: metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"app":   "audit-webhook",
//				},
//			},
//			Ingress: networkPolicyIngress,
//			PolicyTypes: []networkpolicy.PolicyType{"Ingress"},
//		},
//	}
//
//	return networkPolicy
//}
//
//func Issuer() (*certmanagerv1.Issuer){
//
//	issuer := &certmanagerv1.Issuer{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:       "selfsigned-issuer",
//			Labels:		map[string]string{
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//		},
//		Spec:       certmanagerv1.IssuerSpec{
//			IssuerConfig: certmanagerv1.IssuerConfig{
//				SelfSigned: &certmanagerv1.SelfSignedIssuer{},
//			},
//		},
//	}
//
//
//	return  issuer
//}
//
//
//func Certificate(webHook *auditv1beta1.AuditWebhook) (*certmanagerv1.Certificate){
//
//	const dnsNameFront = "audit-webhook-service."
//	const dnsNameBack = ".svc"
//	var dnsName = dnsNameFront + webHook.Namespace + dnsNameBack
//	certificate := &certmanagerv1.Certificate{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:       "serving-cert",
//			Labels:		map[string]string{
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//		},
//		Spec:       certmanagerv1.CertificateSpec{
//			DNSNames:     []string{
//				dnsName,
//			},
//			SecretName:   "audit-webhook-tls-secret",
//			IssuerRef:    certmanagerv1.ObjectReference{
//				Name:  "selfsigned-issuer",
//				Kind:  "Issuer",
//			},
//		},
//	}
//
//	return  certificate
//}
//
//
//
//func Secret(webHook *auditv1beta1.AuditWebhook,secretData map[string][]byte) (*corev1.Secret){
//
//
//	secretType := corev1.SecretTypeTLS
//
//	secret := &corev1.Secret{
//		Type: secretType,
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: webHook.Namespace,
//			Name:      "audit-webhook-tls-secret",
//			Labels: map[string]string{
//				"app": APP_NAME,
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by":  "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name": "ibm-auditwebhook-operator",
//			},
//		},
//		Data: secretData,
//	}
//
//	return  secret
//}
//
//
//
//func ConfigMap(webHook *auditv1beta1.AuditWebhook) (*corev1.ConfigMap) {
//
//	imageName := "cp.stg.icr.io/cp/opencontent-fluentd:ruby-ubi"
//	if len(strings.TrimSpace(webHook.Spec.DockerRegistryPrefix)) > 0 {
//		imageName = webHook.Spec.DockerRegistryPrefix + "/opencontent-fluentd@sha256:d71c70d59540caead90cfb46c83ebafe55787078f73e48bf12558f73b997b17e"
//	}
//
//	volume_patch := "{\"name\":\"internal-tls\",\"secret\":{\"secretName\":\"internal-tls\",\"defaultMode\":420}}"
//	container_patch := "{\"name\": \"sidecar\", 	\"image\": \"" + imageName + "\", 	\"securityContext\": { 		\"runAsNonRoot\": true 	}, 	\"resources\": { 		\"requests\": { 			\"memory\": \"100Mi\", 			\"cpu\": \"100m\" 		}, 		\"limits\": { 			\"memory\": \"250Mi\", 			\"cpu\": \"250m\" 		} 	}, 	\"imagePullPolicy\": \"Always\", 	\"volumeMounts\": [{ 		\"name\": \"varlog\", 		\"mountPath\": \"/var/log\" 	}, { 		\"name\": \"internal-tls\", 		\"mountPath\": \"/etc/internal-tls\" 	}], 	\"env\": [{ 		\"name\": \"NS_DOMAIN\", 		\"value\": \"https://zen-audit-svc." + webHook.Namespace + ":9880/records\" 	}] }"
//
//	// Instantialize the data structure
//	configmap := &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: webHook.Namespace,
//			Name:      "audit-webhook-configmap",
//			Labels: map[string]string{
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//		},
//		Data: map[string]string{
//			"volume_patch":    volume_patch,
//			"container_patch": container_patch,
//		},
//	}
//
//
//	return  configmap
//}
//
//
//func Service(webHook *auditv1beta1.AuditWebhook) (*corev1.Service) {
//
//	service := &corev1.Service{
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: webHook.Namespace,
//			Name:      "audit-webhook-service",
//			Labels: map[string]string{
//				"app":                          APP_NAME,
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//		},
//		Spec: corev1.ServiceSpec{
//			Ports: []corev1.ServicePort{{
//				Port: 443,
//				TargetPort: intstr.IntOrString{
//					IntVal: 8081,
//					StrVal: "8081",
//				},
//				Protocol: corev1.ProtocolTCP,
//			},
//			},
//			Selector: map[string]string{
//				"app": APP_NAME,
//			},
//		},
//	}
//
//
//	return  service
//}
//
//func MutatingWebhookConfiguration(webHook *auditv1beta1.AuditWebhook) (*admissionregistrationv1beta1.MutatingWebhookConfiguration) {
//
//	path := "/add-sidecar"
//	certpath := webHook.Namespace + "/serving-cert"
//
//	failurePolicy := new(admissionregistrationv1beta1.FailurePolicyType)
//	*failurePolicy = admissionregistrationv1beta1.Ignore
//
//	matchPolicy := new(admissionregistrationv1beta1.MatchPolicyType)
//	*matchPolicy = admissionregistrationv1beta1.Equivalent
//
//	scope := new(admissionregistrationv1beta1.ScopeType)
//	*scope = admissionregistrationv1beta1.NamespacedScope
//
//	mc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: webHook.Namespace,
//			Name:      "audit-webhook-config",
//			Annotations: map[string]string{
//				"certmanager.k8s.io/inject-ca-from": certpath + "",
//			},
//		},
//		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{{
//			Name:        "audit.watson.org",
//			MatchPolicy: matchPolicy,
//			ObjectSelector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"cp4d-audit": "yes",
//				},
//			},
//			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
//				Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
//				Rule: admissionregistrationv1beta1.Rule{
//					APIGroups:   []string{""},
//					APIVersions: []string{"v1"},
//					Resources:   []string{"pods"},
//					Scope:       scope,
//				},
//			},
//			},
//			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
//				Service: &admissionregistrationv1beta1.ServiceReference{
//					Name:      "audit-webhook-service",
//					Namespace: webHook.Namespace,
//					Path:      &path,
//					Port:      pointer.Int32Ptr(443),
//				},
//				//CABundle: stringtoslicebyte(webHook.Spec.CaBundle),
//				//CABundle: sDecForCABundle,
//			},
//			FailurePolicy: failurePolicy,
//		},
//		},
//	}
//
//	return  mc
//}
//
//func Deployment(webHook *auditv1beta1.AuditWebhook) (*appsv1.Deployment) {
//
//
//	isRunAsRoot := false
//	pIsRunAsRoot := &isRunAsRoot //bool pointer
//
//	isPrivileged := false
//	pIsPrivileged := &isPrivileged
//
//	isAllowPrivilegeEscalation := false
//	pIsAllowPrivilegeEscalation := &isAllowPrivilegeEscalation
//
//	isReadOnlyRootFilesystem := false
//	pIsReadOnlyRootFilesystem := &isReadOnlyRootFilesystem
//
//	imageName := "cp.stg.icr.io/cp/opencontent-audit-webhook@sha256:0d8c98939b31aa261d09b9f38f834cf524007cf6af1a6e02198bee115d04f918"
//	if len(strings.TrimSpace(webHook.Spec.DockerRegistryPrefix)) > 0 {
//		imageName = webHook.Spec.DockerRegistryPrefix + "/opencontent-audit-webhook:v0.1.0"
//	}
//
//
//	// Instantialize the data structure
//	deployment := &appsv1.Deployment{
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: webHook.Namespace,
//			Name:      "audit-webhook-server",
//			Labels: map[string]string{
//				"app":                          APP_NAME,
//				"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//				"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//				"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//			},
//		},
//		Spec: appsv1.DeploymentSpec{
//			// The replica is computed
//			Replicas: pointer.Int32Ptr(1),
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					"app": APP_NAME,
//				},
//			},
//			Template: corev1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Labels: map[string]string{
//						"app":                          APP_NAME,
//						"app.kubernetes.io/instance":   "ibm-auditwebhook-operator",
//						"app.kubernetes.io/managed-by": "ibm-auditwebhook-operator",
//						"app.kubernetes.io/name":       "ibm-auditwebhook-operator",
//					},
//					Annotations: map[string]string{
//						"productName":    "ibm-auditwebhook",
//						"productID":      "96808888679886798867988679886798",
//						"productVersion": "1.0.0",
//						"productMetric":  "VIRTUAL_PROCESSOR_CORE",
//						"cloudpakId":     "96808888679886798867988679886798",
//						"cloudpakName":   "Cloud Pak Open",
//					},
//				},
//				Spec: corev1.PodSpec{
//					Affinity: &corev1.Affinity{
//						NodeAffinity: &corev1.NodeAffinity{
//							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
//								NodeSelectorTerms: []corev1.NodeSelectorTerm{
//									corev1.NodeSelectorTerm{
//										MatchExpressions: []corev1.NodeSelectorRequirement{
//											corev1.NodeSelectorRequirement{
//												Key:      "kubernetes.io/arch",
//												Operator: "In",
//												Values: []string{
//													"amd64",
//												},
//											},
//										},
//									},
//								},
//							},
//							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
//								corev1.PreferredSchedulingTerm{
//									Weight: 3,
//									Preference: corev1.NodeSelectorTerm{
//										MatchExpressions: []corev1.NodeSelectorRequirement{
//											corev1.NodeSelectorRequirement{
//												Key:      "kubernetes.io/arch",
//												Operator: "In",
//												Values: []string{
//													"amd64",
//												},
//											},
//										},
//									},
//								},
//							},
//						},
//					},
//					HostNetwork:      false,
//					HostIPC:          false,
//					HostPID:          false,
//					ImagePullSecrets: webHook.Spec.ImagePullSecrets,
//					Containers: []corev1.Container{{
//						Image: imageName,
//						LivenessProbe: &corev1.Probe{
//							Handler: corev1.Handler{
//								HTTPGet: &corev1.HTTPGetAction{
//									Path: "/add-sidecar",
//									Port: intstr.IntOrString{
//										Type:   intstr.Int,
//										IntVal: 8081,
//										StrVal: "8081",
//									},
//									Scheme: "HTTPS",
//								},
//							},
//							InitialDelaySeconds: 15,
//							PeriodSeconds:       20,
//						},
//						ReadinessProbe: &corev1.Probe{
//							Handler: corev1.Handler{
//								HTTPGet: &corev1.HTTPGetAction{
//									Path: "/add-sidecar",
//									Port: intstr.IntOrString{
//										Type:   intstr.Int,
//										IntVal: 8081,
//										StrVal: "8081",
//									},
//									Scheme: "HTTPS",
//								},
//							},
//							InitialDelaySeconds: 5,
//							PeriodSeconds:       10,
//						},
//						ImagePullPolicy: "IfNotPresent",
//						Name:            APP_NAME,
//						Command:         []string{"/audit-webhook"},
//						Ports: []corev1.ContainerPort{{
//							ContainerPort: 8081,
//						}},
//						Resources: corev1.ResourceRequirements{
//							Requests: corev1.ResourceList{
//								"cpu":    resource.MustParse(CPU_REQUEST),
//								"memory": resource.MustParse(MEM_REQUEST),
//							},
//							Limits: corev1.ResourceList{
//								"cpu":    resource.MustParse(CPU_LIMIT),
//								"memory": resource.MustParse(MEM_LIMIT),
//							},
//						},
//						SecurityContext: &corev1.SecurityContext{
//							Capabilities: &corev1.Capabilities{
//								Drop: []corev1.Capability{
//									"ALL",
//								},
//							},
//							RunAsNonRoot:             pIsRunAsRoot,
//							AllowPrivilegeEscalation: pIsAllowPrivilegeEscalation,
//							ReadOnlyRootFilesystem:   pIsReadOnlyRootFilesystem,
//							Privileged:               pIsPrivileged,
//						},
//						Env: []corev1.EnvVar{
//							{
//								Name: "VOLUME_PATCH",
//								ValueFrom: &corev1.EnvVarSource{
//									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
//										LocalObjectReference: corev1.LocalObjectReference{
//											Name: "audit-webhook-configmap",
//										},
//										Key: "volume_patch",
//									},
//								},
//							},
//							{
//								Name: "CONTAINER_PATCH",
//								ValueFrom: &corev1.EnvVarSource{
//									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
//										LocalObjectReference: corev1.LocalObjectReference{
//											Name: "audit-webhook-configmap",
//										},
//										Key: "container_patch",
//									},
//								},
//							},
//						},
//						VolumeMounts: []corev1.VolumeMount{
//							{
//								MountPath: "/certs",
//								Name:      "certs",
//								ReadOnly:  false,
//							},
//						},
//					}},
//					Volumes: []corev1.Volume{
//						{
//							Name: "certs",
//							VolumeSource: corev1.VolumeSource{
//								Secret: &corev1.SecretVolumeSource{
//									SecretName: "audit-webhook-tls-secret",
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//
//
//
//	return  deployment
//}
//
//
//
