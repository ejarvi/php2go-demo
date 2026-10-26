package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// Resource types that cost money every month even while the site is idle, or that it
// doesn't need and would lead there (D-020). A prefix ending in "::" bans the whole service.
var costTraps = []string{
	"AWS::EC2::",                    // VPCs, NAT gateways (~$32/month), public IPv4 addresses, instances
	"AWS::ElasticLoadBalancingV2::", // load balancers ($15–30/month)
	"AWS::ElasticLoadBalancing::",   // classic load balancers
	"AWS::ECS::",                    // Fargate
	"AWS::AppRunner::",              // App Runner
	"AWS::RDS::",                    // databases with hourly charges
	"AWS::Route53::HostedZone",      // $0.50/month; the site uses CloudFront's address
	"AWS::SecretsManager::Secret",   // $0.40/month each; SSM standard parameters are free
	"AWS::KMS::Key",                 // $1/month each; AWS-managed keys are free
	"AWS::WAFv2::WebACL",            // pay-as-you-go WAF; CloudFront's Free plan includes one
}

const (
	testAccount = "111111111111"
	testRegion  = "us-east-2"
	testSubject = "repo:example@1/php2go-demo@2:environment:aws"
	testWebACL  = "arn:aws:wafv2:us-east-1:111111111111:global/webacl/CreatedByCloudFront-test/abc"
)

// The jsii runtime (a Node.js process behind CDK's Go API) is started once and closed
// after all tests, and the templates are built once.
func TestMain(m *testing.M) {
	code := m.Run()
	jsii.Close()
	os.Exit(code)
}

var (
	synthOnce sync.Once
	templates map[string]map[string]interface{}
	synthErr  error
)

func synthTemplates(t *testing.T) map[string]map[string]interface{} {
	t.Helper()
	synthOnce.Do(func() { templates, synthErr = synth() })
	if synthErr != nil {
		t.Fatal(synthErr)
	}
	return templates
}

func synth() (map[string]map[string]interface{}, error) {
	asset, err := os.MkdirTemp("", "php2go-asset")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(asset)
	if err := os.WriteFile(filepath.Join(asset, "bootstrap"), []byte("placeholder"), 0o755); err != nil {
		return nil, err
	}
	app := awscdk.NewApp(nil)
	env := &awscdk.Environment{Account: jsii.String(testAccount), Region: jsii.String(testRegion)}
	stacks := map[string]awscdk.Stack{
		"Php2goGitHub": NewGitHubStack(app, "Php2goGitHub", env, GitHubConfig{Subject: testSubject}),
		"Php2go": NewSiteStack(app, "Php2go", env, SiteConfig{
			OriginSecret: "test-secret", LambdaAsset: asset, WebACLArn: testWebACL,
			Settings: map[string]string{"CLUB_NAME": "Test Club", "CLUB_TIMEZONE": "America/Chicago"},
		}),
	}
	out := map[string]map[string]interface{}{}
	for name, s := range stacks {
		tmpl := assertions.Template_FromStack(s, nil).ToJSON()
		res, _ := (*tmpl)["Resources"].(map[string]interface{})
		out[name] = res
	}
	return out, nil
}

func TestNoCostTraps(t *testing.T) {
	for stack, resources := range synthTemplates(t) {
		for id, r := range resources {
			res := r.(map[string]interface{})
			typ, _ := res["Type"].(string)
			for _, trap := range costTraps {
				if strings.HasPrefix(typ, trap) {
					t.Errorf("%s/%s is %s, which costs money while idle (D-020)", stack, id, typ)
				}
			}
			props, _ := res["Properties"].(map[string]interface{})
			if typ == "AWS::Lambda::Function" && props["VpcConfig"] != nil {
				t.Errorf("%s/%s: a Lambda in a VPC needs a NAT gateway to reach the internet (D-020)", stack, id)
			}
			if typ == "AWS::DynamoDB::Table" && props["BillingMode"] == "PAY_PER_REQUEST" {
				t.Errorf("%s/%s: on-demand DynamoDB isn't covered by the always-free allowance; use provisioned", stack, id)
			}
		}
	}
}

// The Free plan requires its web ACL to stay attached, and allows only PriceClass_All.
func TestDistributionFitsFreePlan(t *testing.T) {
	for _, r := range synthTemplates(t)["Php2go"] {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::CloudFront::Distribution" {
			continue
		}
		cfg := jsonString(t, res)
		if !strings.Contains(cfg, testWebACL) {
			t.Error("the CloudFront distribution must reference the Free plan's web ACL when one is configured")
		}
		if !strings.Contains(cfg, `"PriceClass":"PriceClass_All"`) {
			t.Error("the Free plan allows only PriceClass_All")
		}
		return
	}
	t.Fatal("no CloudFront distribution in Php2go")
}

func TestDeployRoleIsConfined(t *testing.T) {
	resources := synthTemplates(t)["Php2goGitHub"]
	found := false
	for _, r := range resources {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::IAM::Role" {
			continue
		}
		found = true
		trust := jsonString(t, res)
		if !strings.Contains(trust, testSubject) || !strings.Contains(trust, "sts.amazonaws.com") {
			t.Errorf("deploy role trust must require subject %q and audience sts.amazonaws.com", testSubject)
		}
	}
	if !found {
		t.Fatal("no IAM role in Php2goGitHub")
	}
	// Exactly two permissions: hand off to CDK's bootstrap roles, and write /php2go/ secrets.
	allowed := map[string]string{
		"sts:AssumeRole":   ":role/cdk-hnb659fds-*",
		"ssm:PutParameter": ":parameter/php2go/*",
	}
	for _, r := range resources {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::IAM::Policy" {
			continue
		}
		doc := res["Properties"].(map[string]interface{})["PolicyDocument"].(map[string]interface{})
		stmts := doc["Statement"].([]interface{})
		if len(stmts) != len(allowed) {
			t.Errorf("deploy role has %d statements, want %d", len(stmts), len(allowed))
		}
		for _, s := range stmts {
			st := s.(map[string]interface{})
			action, _ := st["Action"].(string)
			resource, _ := st["Resource"].(string)
			suffix, ok := allowed[action]
			if !ok || st["Effect"] != "Allow" || !strings.HasSuffix(resource, suffix) || strings.Contains(resource, "*:*") {
				t.Errorf("deploy role statement not allowed: %s", jsonString(t, st))
			}
		}
	}
}

// The table uses free-tier provisioned capacity, no indexes, TTL, point-in-time recovery,
// deletion protection, and is kept if the stack is deleted.
func TestTableSettings(t *testing.T) {
	found := 0
	for id, r := range synthTemplates(t)["Php2go"] {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::DynamoDB::Table" {
			continue
		}
		found++
		p := res["Properties"].(map[string]interface{})
		if p["BillingMode"] != nil && p["BillingMode"] != "PROVISIONED" {
			t.Errorf("%s: BillingMode %v, want PROVISIONED", id, p["BillingMode"])
		}
		pt, _ := p["ProvisionedThroughput"].(map[string]interface{})
		reads, _ := pt["ReadCapacityUnits"].(float64)
		writes, _ := pt["WriteCapacityUnits"].(float64)
		if reads == 0 || writes == 0 || reads > 10 || writes > 10 {
			t.Errorf("%s: capacity %v + %v units; keep it small, since the free 25 + 25 is shared by every table", id, reads, writes)
		}
		if p["GlobalSecondaryIndexes"] != nil || p["LocalSecondaryIndexes"] != nil {
			t.Errorf("%s: secondary indexes use capacity too", id)
		}
		if ttl, _ := p["TimeToLiveSpecification"].(map[string]interface{}); ttl["AttributeName"] != "expires" || ttl["Enabled"] != true {
			t.Errorf("%s: TTL must be on attribute expires", id)
		}
		if pitr, _ := p["PointInTimeRecoverySpecification"].(map[string]interface{}); pitr["PointInTimeRecoveryEnabled"] != true {
			t.Errorf("%s: point-in-time recovery must be on", id)
		}
		if p["DeletionProtectionEnabled"] != true {
			t.Errorf("%s: deletion protection must be on", id)
		}
		if res["DeletionPolicy"] != "Retain" || res["UpdateReplacePolicy"] != "Retain" {
			t.Errorf("%s: the table must be kept if the stack is deleted or it's replaced", id)
		}
	}
	if found != 1 {
		t.Fatalf("want exactly one DynamoDB table, found %d", found)
	}
}

func TestTableNameIsValid(t *testing.T) {
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`).MatchString(tableName) {
		t.Errorf("table name %q isn't valid for DynamoDB", tableName)
	}
}

// Secrets never reach the CloudFormation template; the web Lambda reads them from SSM.
func TestNoSecretsInTemplate(t *testing.T) {
	for _, r := range synthTemplates(t)["Php2go"] {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::Lambda::Function" {
			continue
		}
		env := jsonString(t, res["Properties"].(map[string]interface{})["Environment"])
		for _, bad := range []string{"ORGANIZER_PASSWORD", "SESSION_SECRET"} {
			if strings.Contains(env, bad) {
				t.Errorf("the Lambda's environment mentions %s; secrets belong in SSM", bad)
			}
		}
	}
}

// Only the web Lambda reads secrets, and only under /php2go/.
func TestWebReadsOnlySiteSecrets(t *testing.T) {
	found := false
	for id, r := range synthTemplates(t)["Php2go"] {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::IAM::Policy" {
			continue
		}
		doc := jsonString(t, res)
		if !strings.Contains(doc, "ssm:") {
			continue
		}
		if !strings.HasPrefix(id, "WebServiceRole") {
			t.Errorf("policy %s grants SSM access; only the web Lambda may", id)
		}
		if !strings.Contains(doc, ":parameter/php2go/*") || strings.Contains(doc, "ssm:*") || strings.Contains(doc, "PutParameter") {
			t.Errorf("web SSM access must be ssm:GetParameters on parameter/php2go/* only: %s", doc)
		}
		found = true
	}
	if !found {
		t.Error("the web Lambda can't read its secrets")
	}
}

// The stylesheet is cached at the edge; pages aren't. The Free plan allows 5 behaviours.
func TestCacheBehaviours(t *testing.T) {
	for _, r := range synthTemplates(t)["Php2go"] {
		res := r.(map[string]interface{})
		if res["Type"] != "AWS::CloudFront::Distribution" {
			continue
		}
		cfg := res["Properties"].(map[string]interface{})["DistributionConfig"].(map[string]interface{})
		behaviours, _ := cfg["CacheBehaviors"].([]interface{})
		if len(behaviours)+1 > 5 {
			t.Errorf("%d cache behaviours; the CloudFront Free plan allows 5", len(behaviours)+1)
		}
		found := false
		for _, b := range behaviours {
			bm := b.(map[string]interface{})
			if bm["PathPattern"] == "/assets/*" {
				found = true
				if bm["CachePolicyId"] != "658327ea-f89d-4fab-a63d-7e88639e58f6" { // AWS managed CachingOptimized
					t.Errorf("/assets/* must use CachingOptimized, got %v", bm["CachePolicyId"])
				}
			}
		}
		if !found {
			t.Error("no /assets/* cache behaviour")
		}
		def := cfg["DefaultCacheBehavior"].(map[string]interface{})
		if def["CachePolicyId"] != "4135ea2d-6df8-44a3-9df3-4b5a84be39ad" { // CachingDisabled
			t.Errorf("pages must not be cached, got %v", def["CachePolicyId"])
		}
		return
	}
	t.Fatal("no distribution")
}

func jsonString(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
