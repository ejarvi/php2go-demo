// Command secrets copies the site's secrets from the deploy's environment variables (the
// aws GitHub environment's secrets) to SSM Parameter Store as SecureString parameters.
// The values stay in memory: they're never written to disk, passed as arguments or printed.
//
// Run by .github/workflows/deploy.yml after signing in to AWS.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"github.com/ejarvi/php2go-demo/internal/platform"
)

const prefix = "/php2go/" // must match infra/site.go ssmPrefix

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "::error::"+err.Error())
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	values := map[string]string{
		platform.ParamOrganizerPassword: os.Getenv("ORGANIZER_PASSWORD"),
		platform.ParamSessionSecret:     os.Getenv("SESSION_SECRET"),
	}
	if values[platform.ParamOrganizerPassword] == "" {
		return fmt.Errorf("secret ORGANIZER_PASSWORD is not set in the aws environment")
	}
	if len(values[platform.ParamSessionSecret]) < 32 {
		return fmt.Errorf("secret SESSION_SECRET is missing or shorter than 32 characters")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}
	client := ssm.NewFromConfig(cfg)
	for _, param := range []string{platform.ParamOrganizerPassword, platform.ParamSessionSecret} {
		_, err := client.PutParameter(ctx, &ssm.PutParameterInput{
			Name:      aws.String(prefix + param),
			Value:     aws.String(values[param]),
			Type:      types.ParameterTypeSecureString,
			Tier:      types.ParameterTierStandard, // free; Advanced costs money (D-020)
			Overwrite: aws.Bool(true),
		})
		if err != nil {
			// The SDK's error names the operation and parameter, never the value.
			return fmt.Errorf("writing %s%s: %w", prefix, param, err)
		}
		fmt.Printf("OK: %s%s\n", prefix, param)
	}
	return nil
}
