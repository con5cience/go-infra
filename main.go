package main

import (
	"go-infra/eks"
	"go-infra/iam"
	"go-infra/kube"
	"go-infra/networking"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		env := ctx.Stack()
		clusterName := "megocluster"

		provider, err := aws.NewProvider(
			ctx,
			"aws",
			&aws.ProviderArgs{
				// AccessKey: pulumi.StringPtr(os.Getenv("AWS_ACCESS_KEY")),
				// SecretKey: pulumi.StringPtr(os.Getenv("AWS_SECRET_ACCESS_KEY")),
				Region:  pulumi.String("eu-central-1"),
				Profile: pulumi.String(env),
			},
		)
		if err != nil {
			panic(err)
		}

		// TODO: DNS Zones
		// TODO: Certificates

		network := networking.Deploy(ctx, provider, env, clusterName)
		nodeGroup := iam.Deploy(ctx, provider, env, clusterName)
		cluster := eks.Deploy(ctx, provider, env, network, nodeGroup, clusterName)

		kube.Deploy(ctx, env, cluster)

		return nil
	})
}
