package networking

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type SubnetConfig struct {
	Name          string
	IPv4CidrBlock string
	Tags          pulumi.StringMap
}

type AzSubnets struct {
	Public  SubnetConfig
	Private SubnetConfig
}

type Components struct {
	Vpc                  string
	PrivateSubnets       []string
	PublicSubnets        []string
	ClusterSecurityGroup string
}

func Deploy(
	ctx *pulumi.Context,
	provider *aws.Provider,
	env string,
	clusterName string,
) Components {

	privatePrefix := "primary-private"
	publicPrefix := "primary-public"

	vpc, err := ec2.NewVpc(
		ctx,
		fmt.Sprintf("%s-vpc-%s", clusterName, env),
		&ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(clusterName),
			},
		},
		pulumi.Parent(provider),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	publicRouteTable, err := ec2.NewRouteTable(
		ctx,
		fmt.Sprintf("%s-public-rt-%s", clusterName, env),
		&ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(clusterName + "-public"),
			},
		},
		pulumi.Parent(vpc),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	igw, err := ec2.NewInternetGateway(
		ctx,
		fmt.Sprintf("%s-igw-%s", clusterName, env),
		&ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(clusterName + "-primary"),
			},
		},
		pulumi.Parent(publicRouteTable),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	_, err = ec2.NewRoute(
		ctx,
		fmt.Sprintf("%s-route-public-%s", clusterName, env),
		&ec2.RouteArgs{
			RouteTableId:         publicRouteTable.ID(),
			DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
			GatewayId:            igw.ID(),
		},
		pulumi.Parent(publicRouteTable),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	azSubnets := map[string]*AzSubnets{
		"eu-central-1a": {
			Public: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-public-1a", clusterName, env),
				IPv4CidrBlock: "10.0.0.0/20",
				Tags: pulumi.StringMap{
					"Name":                   pulumi.String(publicPrefix + "-1a"),
					"kubernetes.io/role/elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
			Private: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-private-1a", clusterName, env),
				IPv4CidrBlock: "10.0.48.0/20",
				Tags: pulumi.StringMap{
					"Name":                            pulumi.String(privatePrefix + "-1a"),
					"kubernetes.io/role/internal-elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
		},
		"eu-central-1b": {
			Public: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-public-1b", clusterName, env),
				IPv4CidrBlock: "10.0.16.0/20",
				Tags: pulumi.StringMap{
					"Name":                   pulumi.String(publicPrefix + "-1b"),
					"kubernetes.io/role/elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
			Private: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-private-1b", clusterName, env),
				IPv4CidrBlock: "10.0.64.0/20",
				Tags: pulumi.StringMap{
					"Name":                            pulumi.String(privatePrefix + "-1b"),
					"kubernetes.io/role/internal-elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
		},
		"eu-central-1c": {
			Public: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-public-1c", clusterName, env),
				IPv4CidrBlock: "10.0.32.0/20",
				Tags: pulumi.StringMap{
					"Name":                   pulumi.String(publicPrefix + "-1c"),
					"kubernetes.io/role/elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
			Private: SubnetConfig{
				Name:          fmt.Sprintf("%s-%s-private-1c", clusterName, env),
				IPv4CidrBlock: "10.0.80.0/20",
				Tags: pulumi.StringMap{
					"Name":                            pulumi.String(privatePrefix + "-1c"),
					"kubernetes.io/role/internal-elb": pulumi.String("1"),
					fmt.Sprintf("kubernetes.io/cluster/%s", clusterName): pulumi.String("shared"),
				},
			},
		},
	}

	var privateSubnetIds []string
	var publicSubnetIds []string

	for az, subnet := range azSubnets {

		publicSubnet, err := ec2.NewSubnet(
			ctx,
			subnet.Public.Name,
			&ec2.SubnetArgs{
				VpcId:            vpc.ID(),
				AvailabilityZone: pulumi.String(az),
				CidrBlock:        pulumi.String(subnet.Public.IPv4CidrBlock),
				Tags:             subnet.Public.Tags,
			},
			pulumi.Parent(igw),
			pulumi.Provider(provider),
			pulumi.IgnoreChanges([]string{"Tags"}),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		_, err = ec2.NewRouteTableAssociation(
			ctx,
			subnet.Public.Name,
			&ec2.RouteTableAssociationArgs{
				SubnetId:     publicSubnet.ID(),
				RouteTableId: publicRouteTable.ID(),
			},
			pulumi.Parent(publicSubnet),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		privateRouteTable, err := ec2.NewRouteTable(
			ctx,
			subnet.Private.Name,
			&ec2.RouteTableArgs{
				VpcId: vpc.ID(),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(privatePrefix),
				},
			},
			pulumi.Parent(vpc),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		privateSubnet, err := ec2.NewSubnet(
			ctx,
			subnet.Private.Name,
			&ec2.SubnetArgs{
				VpcId:            vpc.ID(),
				AvailabilityZone: pulumi.String(az),
				CidrBlock:        pulumi.String(subnet.Private.IPv4CidrBlock),
				Tags:             subnet.Private.Tags,
			},
			pulumi.Parent(privateRouteTable),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		eip, err := ec2.NewEip(
			ctx,
			subnet.Private.Name,
			&ec2.EipArgs{
				PublicIpv4Pool: pulumi.String("amazon"),
			},
			pulumi.Parent(privateSubnet),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		nat, err := ec2.NewNatGateway(
			ctx,
			subnet.Private.Name,
			&ec2.NatGatewayArgs{
				AllocationId: eip.AllocationId,
				SubnetId:     privateSubnet.ID(),
				Tags: pulumi.StringMap{
					"Name": pulumi.String(privatePrefix),
				},
			},
			pulumi.Parent(privateSubnet),
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{igw}),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		_, err = ec2.NewRoute(
			ctx,
			subnet.Private.Name,
			&ec2.RouteArgs{
				RouteTableId:         privateRouteTable.ID(),
				DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
				NatGatewayId:         nat.ID(),
			},
			pulumi.Parent(privateRouteTable),
			pulumi.Provider(provider),
			pulumi.Protect(true),
		)
		if err != nil {
			panic(err)
		}

		publicSubnetIds = append(publicSubnetIds, publicSubnet.ID().ToIDOutput().ElementType().String())
		privateSubnetIds = append(privateSubnetIds, privateSubnet.ID().ToIDOutput().ElementType().String())
	}

	clusterSecurityGroup, err := ec2.NewSecurityGroup(
		ctx,
		fmt.Sprintf("%s-%s-sg", clusterName, env),
		&ec2.SecurityGroupArgs{
			VpcId: pulumi.String(vpc.ID().ToIDOutput().ElementType().String()),
			Egress: ec2.SecurityGroupEgressArray{
				ec2.SecurityGroupEgressArgs{
					Protocol:   pulumi.String("-1"),
					FromPort:   pulumi.Int(0),
					ToPort:     pulumi.Int(0),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				ec2.SecurityGroupIngressArgs{
					Protocol:   pulumi.String("tcp"),
					FromPort:   pulumi.Int(80),
					ToPort:     pulumi.Int(80),
					CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				},
			},
		},
		pulumi.Parent(vpc),
		pulumi.Provider(provider),
		pulumi.Protect(true),
	)
	if err != nil {
		panic(err)
	}

	networking := Components{
		Vpc:                  vpc.ID().ToIDOutput().ElementType().String(),
		PublicSubnets:        publicSubnetIds,
		PrivateSubnets:       privateSubnetIds,
		ClusterSecurityGroup: clusterSecurityGroup.ID().ElementType().String(),
	}

	return networking
}
