package kube

import (
	"go-infra/kube/helm/traefik"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/eks"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	providers "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/providers"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Deploy(
	ctx *pulumi.Context,
	env string,
	cluster *eks.Cluster,
) {

	kubeconfig := generateKubeconfig(
		cluster.Endpoint,
		cluster.CertificateAuthority.Data().Elem(),
		cluster.Name,
	)

	ctx.Export("kubeconfig", kubeconfig)

	kubeProvider, err := providers.NewProvider(
		ctx,
		"k8s",
		&providers.ProviderArgs{
			Kubeconfig: kubeconfig,
		})
	if err != nil {
		panic(err)
	}

	corev1.NewNamespace(
		ctx,
		"infra",
		&corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("infra"),
			},
		},
		pulumi.Provider(kubeProvider),
	)

	_, err = traefik.Deploy(ctx)
	if err != nil {
		panic(err)
	}

	// TODO: External DNS
	// TODO: CoreDNS
	// TODO: Test Kubernetes Namespace w/Service, Ingress
	// TODO: Metrics server
	// TODO: Cluster Autoscaler

}

// Create the KubeConfig Structure as per https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
func generateKubeconfig(
	clusterEndpoint pulumi.StringOutput,
	certData pulumi.StringOutput,
	clusterName pulumi.StringOutput,
) pulumi.StringOutput {
	return pulumi.Sprintf(`{
        "apiVersion": "v1",
        "clusters": [{
            "cluster": {
                "server": "%s",
                "certificate-authority-data": "%s"
            },
            "name": "kubernetes",
        }],
        "contexts": [{
            "context": {
                "cluster": "kubernetes",
                "user": "aws",
            },
            "name": "aws",
        }],
        "current-context": "aws",
        "kind": "Config",
        "users": [{
            "name": "aws",
            "user": {
                "exec": {
                    "apiVersion": "client.authentication.k8s.io/v1alpha1",
                    "command": "aws-iam-authenticator",
                    "args": [
                        "token",
                        "-i",
                        "%s",
                    ],
                },
            },
        }],
    }`,
		clusterEndpoint,
		certData,
		clusterName)
}
