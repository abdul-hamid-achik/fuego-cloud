package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type AppConfig struct {
	Name         string
	Namespace    string
	Image        string
	Replicas     int32
	Port         int32
	EnvVars      map[string]string
	Domain       string
	DomainSuffix string
}

func GenerateNamespace(cfg *AppConfig) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       cfg.Name,
				"app.kubernetes.io/managed-by": "nexo-cloud",
			},
		},
	}
}

func GenerateSecret(cfg *AppConfig) *corev1.Secret {
	stringData := make(map[string]string)
	for k, v := range cfg.EnvVars {
		stringData[k] = v
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name + "-env",
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       cfg.Name,
				"app.kubernetes.io/managed-by": "nexo-cloud",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}
}

func GenerateDeployment(cfg *AppConfig) *appsv1.Deployment {
	labels := map[string]string{
		"app.kubernetes.io/name":       cfg.Name,
		"app.kubernetes.io/managed-by": "nexo-cloud",
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cfg.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  cfg.Name,
							Image: cfg.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: cfg.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cfg.Name + "-env",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt32(cfg.Port),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       30,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt32(cfg.Port),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}
}

func GenerateService(cfg *AppConfig) *corev1.Service {
	labels := map[string]string{
		"app.kubernetes.io/name":       cfg.Name,
		"app.kubernetes.io/managed-by": "nexo-cloud",
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt32(cfg.Port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func GenerateIngress(cfg *AppConfig) *networkingv1.Ingress {
	labels := map[string]string{
		"app.kubernetes.io/name":       cfg.Name,
		"app.kubernetes.io/managed-by": "nexo-cloud",
	}

	pathType := networkingv1.PathTypePrefix
	ingressClassName := "traefik"

	host := cfg.Name + "." + cfg.DomainSuffix
	if cfg.Domain != "" {
		host = cfg.Domain
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer":           "letsencrypt-prod",
				"traefik.ingress.kubernetes.io/router.tls": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{host},
					SecretName: cfg.Name + "-tls",
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: cfg.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
