// Command infra is the site's AWS infrastructure as code (AWS CDK in Go).
//
//	Php2goGitHub  GitHub OIDC sign-in and the deploy role. Deployed once by hand from a PC
//	              (platform/README.md), because CI can't create the role it signs in with.
//	Php2go        DynamoDB + Lambda + function URL + CloudFront. Deployed by GitHub Actions.
//
// Nothing account-specific is committed (D-024). It all comes from environment variables:
//
//	AWS_ACCOUNT_ID, AWS_REGION   the account and region to deploy to (both stacks)
//	GITHUB_OIDC_SUBJECT          defines Php2goGitHub: the GitHub subject the role trusts
//	EXISTING_OIDC_PROVIDER_ARN   optional: reuse the account's GitHub OIDC provider
//	ORIGIN_SECRET                defines Php2go: the header CloudFront adds (CI sets it)
//	CLUB_NAME, CLUB_TIMEZONE, SITE_URL   the site's settings, passed to the Lambda
//	FREE_PLAN_WEB_ACL_ARN        optional: the web ACL CloudFront's Free plan attached
//	LAMBDA_ASSET                 optional: the folder with the built "bootstrap" binary
package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

func main() {
	defer jsii.Close()
	account, region := os.Getenv("AWS_ACCOUNT_ID"), os.Getenv("AWS_REGION")
	if account == "" || region == "" {
		fmt.Fprintln(os.Stderr, "infra: set AWS_ACCOUNT_ID and AWS_REGION (platform/README.md)")
		os.Exit(1)
	}
	app := awscdk.NewApp(nil)
	env := &awscdk.Environment{Account: jsii.String(account), Region: jsii.String(region)}
	awscdk.Tags_Of(app).Add(jsii.String("project"), jsii.String("php2go"), nil)

	if subject := os.Getenv("GITHUB_OIDC_SUBJECT"); subject != "" {
		NewGitHubStack(app, "Php2goGitHub", env, GitHubConfig{
			Subject:             subject,
			ExistingProviderArn: os.Getenv("EXISTING_OIDC_PROVIDER_ARN"),
		})
	}

	if secret := os.Getenv("ORIGIN_SECRET"); secret != "" {
		NewSiteStack(app, "Php2go", env, SiteConfig{
			OriginSecret: secret,
			LambdaAsset:  envOr("LAMBDA_ASSET", "../dist/lambda"),
			WebACLArn:    os.Getenv("FREE_PLAN_WEB_ACL_ARN"),
			Settings: map[string]string{
				"CLUB_NAME":     os.Getenv("CLUB_NAME"),
				"CLUB_TIMEZONE": os.Getenv("CLUB_TIMEZONE"),
				"SITE_URL":      os.Getenv("SITE_URL"),
			},
		})
	}

	app.Synth(nil)
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
