package github

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/logging"
	"golang.org/x/oauth2"
)

type Config struct {
	Token        string
	Organization string
	BaseURL      string
	Insecure     bool
}

type Organization struct {
	name   string
	client *github.Client

	readLabel        func(ctx context.Context, owner, repo, name string) (*github.Label, error)
	readCollaborator func(ctx context.Context, owner, repo, name string) (*github.User, error)

	StopContext context.Context
}

// Client configures and returns a fully initialized GithubClient
func (c *Config) Client() (interface{}, error) {
	var org Organization
	org.name = c.Organization
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)

	ctx := context.Background()

	if c.Insecure {
		insecureClient := insecureHttpClient()
		ctx = context.WithValue(ctx, oauth2.HTTPClient, insecureClient)
	}

	tc := oauth2.NewClient(ctx, ts)

	tc.Transport = NewEtagTransport(tc.Transport)

	tc.Transport = NewRateLimitTransport(tc.Transport)

	tc.Transport = logging.NewTransport("Github", tc.Transport)

	org.client = github.NewClient(tc)
	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, err
		}
		org.client.BaseURL = u
	}

	// this could probably be 1 second
	const batchTiming = 2 * time.Second

	org.readLabel = batchReadLabel(batchTiming, org.client.Issues)
	org.readCollaborator = batchReadCollaborator(batchTiming, org.client.Repositories)

	return &org, nil
}

func insecureHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}
