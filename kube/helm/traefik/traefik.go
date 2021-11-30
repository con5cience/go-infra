package traefik

import (
	helm "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	hpa "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/autoscaling/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Deploy(ctx *pulumi.Context) (*helm.Chart, error) {

	hpa, err := func(args *pulumi.ResourceTransformationArgs) *pulumi.ResourceTransformationResult {
		if args.Type == "Deployment" {
			return &pulumi.ResourceTransformationResult{
				hpa.NewHorizontalPodAutoscaler(
					ctx,
					"traefik-hpa",
					&hpa.HorizontalPodAutoscalerArgs{
						Metadata: pulumi.Map{
							"namespace": pulumi.String("infra"),
						},
						Spec: pulumi.Map{
							"minReplicas": pulumi.Int(3),
							"maxReplicas": pulumi.Int(20),
							"scaleTargetRef": pulumi.Map{
								"apiVersion": pulumi.String("apps/v1"),
								"kind": pulumi.String("deployment"),
								"name": pulumi.String(args.Name),
							},
						},
					},
				)
			}
		}
	}

	traefik, err := helm.NewChart(
		ctx,
		"traefik",
		helm.ChartArgs{
			Chart:     pulumi.String("nginx-ingress"),
			Version:   pulumi.String("9.13.0"),
			Namespace: pulumi.String("infra"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://charts.helm.sh/stable"),
			},
			Values: pulumi.Map{
				"providers": pulumi.Map{
					"kubernetesIngress": pulumi.Map{
						"publishedService": pulumi.Map{
							"enabled": pulumi.Bool(true),
						},
					},
				},
				"ports": pulumi.Map{
					"web": pulumi.Map{
						"redirectTo": pulumi.String("websecure"),
					},
				},
				"logs": pulumi.Map{
					"general": pulumi.Map{
						"level":  pulumi.String("ERROR"),
						"format": pulumi.String("json"),
					},
					"access": pulumi.Map{
						"enabled": pulumi.Bool(true),
						"format":  pulumi.String("json"),
						"fields": pulumi.Map{
							"headers": pulumi.Map{
								"defaultmode": pulumi.String("keep"),
								"names": pulumi.Map{
									"Authorization": pulumi.String("redact"),
								},
							},
						},
					},
				},
				"resources": pulumi.Map{
					"limits": pulumi.Map{
						"cpu":    pulumi.String("1000m"),
						"memory": pulumi.String("1.25G"),
					},
					"requests": pulumi.Map{
						"cpu":    pulumi.String("1000m"),
						"memory": pulumi.String("1.25G"),
					},
				},
				"additionalArguments": pulumi.StringArray{
					pulumi.String("--api.dashboard"),
					pulumi.String("--metrics.datadog.address=datadog-statsd:8125"),
					// TODO: align these CIDRs with the VPC we created earlier
					pulumi.String("--entryPoints.web.forwardedHeaders.trustedIPs=10.9.0.0/16,10.10.0.0/16"),
					pulumi.String("--entryPoints.web.proxyProtocol.trustedIPs=10.9.0.0/16,10.10.0.0/16"),
					pulumi.String("--entryPoints.websecure.forwardedHeaders.trustedIPs=10.9.0.0/16,10.10.0.0/16"),
					pulumi.String("--entryPoints.websecure.proxyProtocol.trustedIPs=10.9.0.0/16,10.10.0.0/16"),
					pulumi.String("--entryPoints.web.transport.respondingTimeouts.readTimeout=30s"),
					pulumi.String("--entryPoints.web.transport.respondingTimeouts.writeTimeout=30s"),
					pulumi.String("--entryPoints.web.transport.respondingTimeouts.idleTimeout=30s"),
					pulumi.String("--entryPoints.websecure.transport.respondingTimeouts.readTimeout=30s"),
					pulumi.String("--entryPoints.websecure.transport.respondingTimeouts.writeTimeout=30s"),
					pulumi.String("--entryPoints.websecure.transport.respondingTimeouts.idleTimeout=30s"),
				},
				"globalArguments": pulumi.StringArray{},
				"service": pulumi.Map{
					"loadBalancerSourceRanges": pulumi.StringArray{
						pulumi.String("10.10.0.0/16"),
					},
					"annotations": pulumi.Map{
						// TODO: elb vs nlb
						"service.beta.kubernetes.io/aws-load-balancer-backend-protocol": pulumi.String("http"),
						// TODO: pass in certificate
						// "service.beta.kubernetes.io/aws-load-balancer-ssl-cert': pulumi.String(pulumi.Sprintf("%s", certificate.arn.ElementType().String())),
						"service.beta.kubernetes.io/aws-load-balancer-ssl-ports":               pulumi.String("websecure"),
						"service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout": pulumi.String("30"),
						// This is for layer 4. It will forward ip through to traefik. THIS DOESN'T SUPPORT SECURITY GROUPS
						"service.beta.kubernetes.io/aws-load-balancer-type": pulumi.String("nlb"),
					},
				},
				"podDisruptionBudget": pulumi.Map{
					"enabled":      pulumi.Bool(true),
					"minAvailable": pulumi.Int(3),
				},
				"deployment": pulumi.Map{
					"replicas": pulumi.Int(3),
				},
			},
			Transformations: []pulumi.ResourceTransformations{
				pulumi.ResourceTransformation{hpa},
			},
		},
	)

	return traefik, err
}
