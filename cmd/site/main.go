// Command site is the club's events site. The same binary runs as a local web server
// (tests, CI) or on AWS Lambda behind CloudFront.
//
//	SITE_TABLE         DynamoDB table name; unset = in-memory store (local only)
//	DYNAMODB_ENDPOINT  a DynamoDB Local address, for tests; the table is created if missing
//	SSM_PREFIX         where the deploy wrote the secrets in SSM Parameter Store; unset =
//	                   ORGANIZER_PASSWORD and SESSION_SECRET from the environment
//	ORIGIN_SECRET      the header value CloudFront adds; unset = no check (local)
//	PORT               the local port (default 8080)
//	CLUB_NAME, CLUB_TIMEZONE, SITE_URL, FORM_MIN_SECONDS   see platform/README.md
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/ejarvi/php2go-demo/internal/platform"
	"github.com/ejarvi/php2go-demo/internal/site"
	"github.com/ejarvi/php2go-demo/internal/store"
)

func main() {
	ctx := context.Background()
	log := slog.Default()
	fail := func(msg string, err error) {
		log.Error(msg, "err", err)
		os.Exit(1)
	}

	var st store.Store = store.NewMemory()
	var ssmClient *ssm.Client
	if table := os.Getenv("SITE_TABLE"); table != "" {
		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			fail("loading AWS config", err)
		}
		endpoint := os.Getenv("DYNAMODB_ENDPOINT")
		d := store.NewDynamo(dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
			if endpoint != "" {
				o.BaseEndpoint = aws.String(endpoint)
			}
		}), table)
		if endpoint != "" {
			if err := d.EnsureTable(ctx); err != nil {
				fail("creating the local table", err)
			}
		}
		st = d
		if os.Getenv("SSM_PREFIX") != "" {
			ssmClient = ssm.NewFromConfig(awsCfg)
		}
	}

	cfg, err := platform.LoadConfig(ctx, ssmClient)
	if err != nil {
		fail("loading configuration", err)
	}
	handler, err := site.New(site.Options{Store: st, Config: cfg, Log: log})
	if err != nil {
		fail("starting the site", err)
	}
	handler = platform.OriginGuard(os.Getenv("ORIGIN_SECRET"), handler)

	// A Lambda function URL sends payload format 2.0.
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(httpadapter.NewV2(handler).ProxyWithContext)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Info("listening", "url", "http://127.0.0.1:"+port, "release", platform.Release())
	if err := http.ListenAndServe("127.0.0.1:"+port, handler); err != nil {
		fail("server stopped", err)
	}
}
