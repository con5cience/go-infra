package iam

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Deploy(
	ctx *pulumi.Context,
	provider *aws.Provider,
	env string,
	clusterName string,
) *iam.Role {

	eksRole, err := iam.NewRole(
		ctx,
		fmt.Sprintf("%s-eks-iam-assumeRole-%s", clusterName, env),
		&iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2008-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "eks.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		},
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	eksPolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
		"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
	}

	for i, eksPolicy := range eksPolicies {
		_, err := iam.NewRolePolicyAttachment(
			ctx,
			fmt.Sprintf("%s-eks-rpa-%s-%d", clusterName, env, i),
			&iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String(eksPolicy),
				Role:      eksRole.Name,
			},
			pulumi.Parent(eksRole),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}
	}

	nodeGroupRole, err := iam.NewRole(
		ctx,
		fmt.Sprintf("%s-nodegroup-iam-role-%s", clusterName, env),
		&iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "ec2.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		},
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	nodeGroupPolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
	}

	for i, nodeGroupPolicy := range nodeGroupPolicies {
		_, err := iam.NewRolePolicyAttachment(
			ctx,
			fmt.Sprintf("%s-node-gpa-%s-%d", clusterName, env, i),
			&iam.RolePolicyAttachmentArgs{
				Role:      nodeGroupRole.Name,
				PolicyArn: pulumi.String(nodeGroupPolicy),
			},
			pulumi.Parent(nodeGroupRole),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)

		if err != nil {
			panic(err)
		}
	}

	return nodeGroupRole
}
