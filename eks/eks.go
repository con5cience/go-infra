package eks

import (
	"fmt"
	"go-infra/networking"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Deploy(
	ctx *pulumi.Context,
	provider *aws.Provider,
	env string,
	network networking.Components,
	eksIamRole *iam.Role,
	clusterName string,
) *eks.Cluster {

	cluster, err := eks.NewCluster(
		ctx,
		clusterName,
		&eks.ClusterArgs{
			Name:    pulumi.StringPtr(clusterName),
			Version: pulumi.StringPtr("1.19"),
			RoleArn: pulumi.StringInput(eksIamRole.Arn),
			VpcConfig: &eks.ClusterVpcConfigArgs{
				EndpointPrivateAccess: pulumi.BoolPtr(true),
				EndpointPublicAccess:  pulumi.BoolPtr(true),
				PublicAccessCidrs: pulumi.StringArray{
					pulumi.String("0.0.0.0/0"),
				},
				SubnetIds: pulumi.StringArray{
					pulumi.String(network.PrivateSubnets[0]),
					pulumi.String(network.PrivateSubnets[1]),
					pulumi.String(network.PrivateSubnets[2]),
					pulumi.String(network.PublicSubnets[0]),
					pulumi.String(network.PublicSubnets[1]),
					pulumi.String(network.PublicSubnets[2]),
				},
				SecurityGroupIds: pulumi.StringArray{
					pulumi.String(network.ClusterSecurityGroup),
				},
			},
			EnabledClusterLogTypes: pulumi.StringArray{
				pulumi.String("api"),
				pulumi.String("audit"),
				pulumi.String("authenticator"),
				pulumi.String("controllerManager"),
				pulumi.String("scheduler"),
			},
			Tags: pulumi.StringMap{
				"k8s.io/cluster-autoscaler/enabled": pulumi.String("true"),
			},
		},
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	launchTemplate, err := ec2.NewLaunchTemplate(
		ctx,
		fmt.Sprintf("%s-%s-launchTemplate", clusterName, env),
		&ec2.LaunchTemplateArgs{
			Description: pulumi.String("EKS launch template managed by Pulumi"),
			VpcSecurityGroupIds: pulumi.StringArray{
				pulumi.String(network.ClusterSecurityGroup),
				// add other SGs here for whitelisting additional ingresses
			},
			BlockDeviceMappings: ec2.LaunchTemplateBlockDeviceMappingArray{
				&ec2.LaunchTemplateBlockDeviceMappingArgs{
					DeviceName: pulumi.String("/dev/xvda"),
					Ebs: &ec2.LaunchTemplateBlockDeviceMappingEbsArgs{
						VolumeSize:          pulumi.Int(20),
						DeleteOnTermination: pulumi.String("true"),
					},
				},
			},
			EbsOptimized: pulumi.String("true"),
			TagSpecifications: &ec2.LaunchTemplateTagSpecificationArray{
				&ec2.LaunchTemplateTagSpecificationArgs{
					ResourceType: pulumi.String("instance"),
					Tags: pulumi.StringMap{
						"Name": pulumi.String(fmt.Sprintf("%s-eksCluster-worker", clusterName)),
					},
				},
			},
		},
		pulumi.Parent(cluster),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)

	azs := []string{"a", "b", "c"}

	for i, az := range azs {
		_, err := eks.NewNodeGroup(
			ctx,
			fmt.Sprintf("%s-node-group-%s", clusterName, az),
			&eks.NodeGroupArgs{
				ClusterName:   cluster.Name,
				NodeGroupName: pulumi.String(fmt.Sprintf("%s-spot-node-group-%s", clusterName, az)),
				NodeRoleArn:   pulumi.StringInput(eksIamRole.Arn),
				SubnetIds: pulumi.StringArray{
					pulumi.String(network.PrivateSubnets[i]),
				},
				CapacityType:       pulumi.String("SPOT"),
				DiskSize:           pulumi.IntPtr(20),
				ForceUpdateVersion: pulumi.BoolPtr(false),
				AmiType:            pulumi.StringPtr("AL2_x86_64"),
				InstanceTypes: pulumi.StringArray{
					pulumi.String("t3.micro"),
				},
				ScalingConfig: &eks.NodeGroupScalingConfigArgs{
					DesiredSize: pulumi.Int(2),
					MaxSize:     pulumi.Int(3),
					MinSize:     pulumi.Int(1),
				},
				LaunchTemplate: &eks.NodeGroupLaunchTemplateArgs{
					Version: pulumi.String(pulumi.Sprintf("%s", launchTemplate.LatestVersion).ElementType().String()),
					Name:    pulumi.StringPtr(pulumi.Sprintf("%s", launchTemplate.Name).ElementType().String()),
				},
				Tags: pulumi.StringMap{
					"Name": pulumi.String(fmt.Sprintf("%s-eksCluster-worker", clusterName)),
				},
			},
			pulumi.Parent(cluster),
			pulumi.Provider(provider),
			pulumi.Protect(true),
			pulumi.IgnoreChanges([]string{"Tags", "ScalingConfig"}),
		)
		if err != nil {
			panic(err)
		}
	}

	return cluster
}
