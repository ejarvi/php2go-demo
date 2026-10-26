package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// GitHubConfig is what the GitHub stack needs from outside the code.
type GitHubConfig struct {
	// The token subject the role trusts, such as
	// repo:<owner>@<owner id>/<repo>@<repo id>:environment:aws. Repos using GitHub's
	// immutable subject format include the numeric IDs, so a deleted and re-created repo,
	// or a reused account name, is refused (platform/README.md).
	Subject string
	// An existing GitHub OIDC provider in the account. An account can have only one, so a
	// second project in the same account reuses it. Empty means create one.
	ExistingProviderArn string
}

// NewGitHubStack lets GitHub Actions sign in to AWS without stored keys (OIDC), as a role
// that can do nothing but hand off to CDK's own bootstrap roles and write the site's secrets.
func NewGitHubStack(scope constructs.Construct, id string, env *awscdk.Environment, cfg GitHubConfig) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &awscdk.StackProps{
		Env:         env,
		Description: jsii.String("php2go: GitHub Actions OIDC sign-in and deploy role"),
	})

	providerArn := cfg.ExistingProviderArn
	if providerArn == "" {
		provider := awsiam.NewCfnOIDCProvider(stack, jsii.String("GitHubOidc"), &awsiam.CfnOIDCProviderProps{
			Url:          jsii.String("https://token.actions.githubusercontent.com"),
			ClientIdList: jsii.Strings("sts.amazonaws.com"),
		})
		providerArn = *provider.AttrArn()
	}

	role := awsiam.NewRole(stack, jsii.String("DeployRole"), &awsiam.RoleProps{
		RoleName:    jsii.String("php2go-github-deploy"),
		Description: jsii.String("Assumed by GitHub Actions in the aws environment; may only assume CDK's bootstrap roles"),
		AssumedBy: awsiam.NewWebIdentityPrincipal(jsii.String(providerArn), &map[string]interface{}{
			"StringEquals": map[string]interface{}{
				"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
				"token.actions.githubusercontent.com:sub": cfg.Subject,
			},
		}),
		MaxSessionDuration: awscdk.Duration_Hours(jsii.Number(1)),
	})
	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("AssumeCdkBootstrapRoles"),
		Actions:   jsii.Strings("sts:AssumeRole"),
		Resources: jsii.Strings("arn:aws:iam::" + *env.Account + ":role/cdk-hnb659fds-*"),
	}))
	// The deploy copies GitHub secrets into SSM Parameter Store under /php2go/: write only.
	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("WriteSiteSecrets"),
		Actions:   jsii.Strings("ssm:PutParameter"),
		Resources: jsii.Strings("arn:aws:ssm:" + *env.Region + ":" + *env.Account + ":parameter" + ssmPrefix + "*"),
	}))

	awscdk.NewCfnOutput(stack, jsii.String("DeployRoleArn"), &awscdk.CfnOutputProps{
		Value:       role.RoleArn(),
		Description: jsii.String("GitHub environment secret AWS_ROLE_ARN"),
	})
	return stack
}
