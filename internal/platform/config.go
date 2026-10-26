package platform

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // Lambda's OS-only runtime has no time zone files

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Config is everything the site needs at run time (platform/README.md, Settings).
// Secrets come from SSM Parameter Store on AWS, where the deploy writes them, and from
// environment variables locally and in CI.
type Config struct {
	ClubName string
	Location *time.Location // the club's time zone (D-006)
	SiteURL  string         // https://… once known; trusted as a same-origin form source (D-025)

	OrganizerPassword string // the shared organizers' password (D-023). Never log it.
	SessionSecret     []byte // signs form timers and hashes session codes. Never log it.

	FormMinSeconds int // the shortest time to fill in a public form (D-011)
}

// The SSM SecureString parameters the deploy writes under SSM_PREFIX (cmd/secrets).
const (
	ParamOrganizerPassword = "organizer-password"
	ParamSessionSecret     = "session-secret"
)

// LoadConfig reads the configuration. With ssmClient nil, secrets come from the
// ORGANIZER_PASSWORD and SESSION_SECRET environment variables.
func LoadConfig(ctx context.Context, ssmClient *ssm.Client) (Config, error) {
	c := Config{
		ClubName:       os.Getenv("CLUB_NAME"),
		SiteURL:        strings.TrimRight(os.Getenv("SITE_URL"), "/"),
		FormMinSeconds: 3,
	}
	if c.ClubName == "" {
		return c, fmt.Errorf("the setting CLUB_NAME is missing")
	}
	zone := os.Getenv("CLUB_TIMEZONE")
	loc, err := time.LoadLocation(zone)
	if zone == "" || err != nil {
		return c, fmt.Errorf("CLUB_TIMEZONE must be a time zone name such as America/Chicago, got %q", zone)
	}
	c.Location = loc
	if v := os.Getenv("FORM_MIN_SECONDS"); v != "" {
		if c.FormMinSeconds, err = strconv.Atoi(v); err != nil || c.FormMinSeconds < 0 {
			return c, fmt.Errorf("FORM_MIN_SECONDS must be a whole number of seconds")
		}
	}

	secrets := map[string]string{
		ParamOrganizerPassword: os.Getenv("ORGANIZER_PASSWORD"),
		ParamSessionSecret:     os.Getenv("SESSION_SECRET"),
	}
	if ssmClient != nil {
		// Read once at start-up: a warm Lambda keeps these until the next deploy, which is
		// why changing a secret means deploying again (D-023).
		prefix := os.Getenv("SSM_PREFIX")
		out, err := ssmClient.GetParameters(ctx, &ssm.GetParametersInput{
			Names:          []string{prefix + ParamOrganizerPassword, prefix + ParamSessionSecret},
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return c, fmt.Errorf("reading SSM parameters: %w", err)
		}
		if len(out.InvalidParameters) > 0 {
			return c, fmt.Errorf("SSM parameters missing: %v", out.InvalidParameters) // names only, never values
		}
		for _, p := range out.Parameters {
			secrets[strings.TrimPrefix(aws.ToString(p.Name), prefix)] = aws.ToString(p.Value)
		}
	}
	c.OrganizerPassword = secrets[ParamOrganizerPassword]
	c.SessionSecret = []byte(secrets[ParamSessionSecret])
	if c.OrganizerPassword == "" || len(c.SessionSecret) < 16 {
		return c, fmt.Errorf("the organizers' password or the session secret (16 characters or more) is missing")
	}
	return c, nil
}
