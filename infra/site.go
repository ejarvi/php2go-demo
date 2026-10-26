package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// The table's name, fixed so the console, backups and tests all refer to the same one.
const tableName = "php2go-data"

// Where the deploy writes the site's secrets (cmd/secrets); the web Lambda may read only these.
const ssmPrefix = "/php2go/"

// The header CloudFront adds to every request; must match internal/platform OriginHeader.
const originHeader = "X-Php2go-Origin"

// SiteConfig is what the site stack needs from outside the code.
type SiteConfig struct {
	OriginSecret string // CloudFront adds it to every request; the app refuses requests without it
	LambdaAsset  string // folder holding the built "bootstrap" binary
	// The AWS WAF web ACL that CloudFront's flat-rate Free plan creates and attaches when
	// the distribution subscribes. A distribution on a plan must keep it, so once it exists
	// the stack references it; otherwise a deploy that changes the distribution would try to
	// detach it and fail. Empty before subscribing.
	WebACLArn string
	// Non-secret settings from the GitHub environment. Secrets go through SSM instead.
	Settings map[string]string
}

// NewSiteStack is the site: CloudFront → Lambda function URL → the Go binary → DynamoDB.
// The Lambda isn't in a VPC, and nothing here costs money while idle (D-020).
func NewSiteStack(scope constructs.Construct, id string, env *awscdk.Environment, cfg SiteConfig) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &awscdk.StackProps{
		Env:         env,
		Description: jsii.String("php2go: Go on Lambda behind CloudFront"),
	})

	// One table for everything. Provisioned 5 + 5 stays well inside the always-free 25 + 25,
	// which every table in the account and region shares. Deletion protection and Retain
	// keep the data if the stack or a click goes wrong.
	table := awsdynamodb.NewTable(stack, jsii.String("Table"), &awsdynamodb.TableProps{
		TableName:           jsii.String(tableName),
		PartitionKey:        &awsdynamodb.Attribute{Name: jsii.String("PK"), Type: awsdynamodb.AttributeType_STRING},
		SortKey:             &awsdynamodb.Attribute{Name: jsii.String("SK"), Type: awsdynamodb.AttributeType_STRING},
		BillingMode:         awsdynamodb.BillingMode_PROVISIONED,
		ReadCapacity:        jsii.Number(5),
		WriteCapacity:       jsii.Number(5),
		TimeToLiveAttribute: jsii.String("expires"),
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(true),
		},
		DeletionProtection: jsii.Bool(true),
		RemovalPolicy:      awscdk.RemovalPolicy_RETAIN,
	})

	logs := awslogs.NewLogGroup(stack, jsii.String("WebLogs"), &awslogs.LogGroupProps{
		Retention:     awslogs.RetentionDays_ONE_MONTH,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	webEnv := map[string]*string{
		"SITE_TABLE":    table.TableName(),
		"SSM_PREFIX":    jsii.String(ssmPrefix),
		"ORIGIN_SECRET": jsii.String(cfg.OriginSecret),
	}
	for k, v := range cfg.Settings {
		if v != "" {
			webEnv[k] = jsii.String(v)
		}
	}
	fn := awslambda.NewFunction(stack, jsii.String("Web"), &awslambda.FunctionProps{
		Description:  jsii.String("php2go web site (cmd/site)"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Architecture: awslambda.Architecture_ARM_64(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String(cfg.LambdaAsset), nil),
		MemorySize:   jsii.Number(128),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(10)),
		LogGroup:     logs,
		Environment:  &webEnv,
	})
	table.GrantReadWriteData(fn)
	// Read the /php2go/ secrets at cold start. They're encrypted with the AWS-managed key
	// aws/ssm, whose key policy already allows use through SSM, so no KMS permission is needed.
	fn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("ReadSiteSecrets"),
		Actions:   jsii.Strings("ssm:GetParameters"),
		Resources: jsii.Strings("arn:aws:ssm:" + *env.Region + ":" + *env.Account + ":parameter" + ssmPrefix + "*"),
	}))

	url := fn.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType: awslambda.FunctionUrlAuthType_NONE,
	})
	// CloudFront's signed origin access can't sign plain HTML form posts to a function URL,
	// so CloudFront adds a secret header that the site checks instead.
	origin := awscloudfrontorigins.NewFunctionUrlOrigin(url, &awscloudfrontorigins.FunctionUrlOriginProps{
		CustomHeaders: &map[string]*string{originHeader: jsii.String(cfg.OriginSecret)},
	})
	props := &awscloudfront.DistributionProps{
		Comment: jsii.String("php2go"),
		// The Free plan allows no price class other than All, at no extra cost.
		PriceClass: awscloudfront.PriceClass_PRICE_CLASS_ALL,
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:               origin,
			AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_ALL(),
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			// Pages show sign-up counts and organizer views, so they aren't cached.
			CachePolicy: awscloudfront.CachePolicy_CACHING_DISABLED(),
			// A function URL needs its own Host header, so everything else is forwarded,
			// including Origin and Sec-Fetch-Site for the same-origin check (D-025).
			OriginRequestPolicy: awscloudfront.OriginRequestPolicy_ALL_VIEWER_EXCEPT_HOST_HEADER(),
		},
		// The stylesheet under /assets/<release>/ is cached at the edge for a year; each
		// release has a new address. No viewer headers or cookies are forwarded for it.
		AdditionalBehaviors: &map[string]*awscloudfront.BehaviorOptions{
			"/assets/*": {
				Origin:               origin,
				AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),
				ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
				CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(),
				Compress:             jsii.Bool(true),
			},
		},
	}
	if cfg.WebACLArn != "" {
		props.WebAclId = jsii.String(cfg.WebACLArn)
	}
	dist := awscloudfront.NewDistribution(stack, jsii.String("Cdn"), props)

	awscdk.NewCfnOutput(stack, jsii.String("SiteUrl"), &awscdk.CfnOutputProps{
		Value: jsii.String("https://" + *dist.DistributionDomainName()),
	})
	awscdk.NewCfnOutput(stack, jsii.String("FunctionUrl"), &awscdk.CfnOutputProps{
		Value:       url.Url(),
		Description: jsii.String("Direct address; must answer 403 (only CloudFront may call it)"),
	})
	awscdk.NewCfnOutput(stack, jsii.String("DistributionId"), &awscdk.CfnOutputProps{Value: dist.DistributionId()})
	return stack
}
