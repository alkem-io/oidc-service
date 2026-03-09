package webhook

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	kratosClient "github.com/ory/client-go"
)

// KratosAdminClient wraps the Kratos API client to implement KratosAdmin interface.
type KratosAdminClient struct {
	api *kratosClient.APIClient
}

// NewKratosAdminClient creates a KratosAdminClient from an existing Ory client.
// Returns an error if the api client is nil.
func NewKratosAdminClient(api *kratosClient.APIClient) (*KratosAdminClient, error) {
	if api == nil {
		return nil, fmt.Errorf("kratos api client is nil")
	}
	return &KratosAdminClient{api: api}, nil
}

// IdentityPage holds a page of identities and the token to fetch the next page.
type IdentityPage struct {
	Identities []kratosClient.Identity
	NextToken  string // empty when no more pages
}

// ListIdentities fetches a page of identities from the Kratos Admin API.
// Use an empty pageToken for the first page.
func (c *KratosAdminClient) ListIdentities(ctx context.Context, pageSize int64, pageToken string) (*IdentityPage, error) {
	req := c.api.IdentityAPI.ListIdentities(ctx).PageSize(pageSize)
	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	identities, resp, err := req.Execute()
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("list identities failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("list identities failed: %w", err)
	}

	nextToken := parseNextPageToken(resp)

	return &IdentityPage{
		Identities: identities,
		NextToken:  nextToken,
	}, nil
}

// parseNextPageToken extracts the next page token from the Link header.
// Ory returns: <url?page_token=TOKEN&page_size=N>; rel="next"
func parseNextPageToken(resp *http.Response) string {
	if resp == nil {
		return ""
	}

	for _, link := range resp.Header.Values("Link") {
		for _, part := range strings.Split(link, ",") {
			part = strings.TrimSpace(part)
			if !strings.Contains(part, `rel="next"`) {
				continue
			}
			// Extract URL from angle brackets
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start < 0 || end < 0 || end <= start {
				continue
			}
			u, err := url.Parse(part[start+1 : end])
			if err != nil {
				continue
			}
			if token := u.Query().Get("page_token"); token != "" {
				return token
			}
		}
	}

	return ""
}

// PatchIdentity updates identity metadata using JSON Patch.
func (c *KratosAdminClient) PatchIdentity(ctx context.Context, identityID string, patches []kratosClient.JsonPatch) error {
	_, resp, err := c.api.IdentityAPI.PatchIdentity(ctx, identityID).
		JsonPatch(patches).
		Execute()

	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}

	if err != nil {
		if resp != nil {
			return fmt.Errorf("patch identity failed with status %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("patch identity failed: %w", err)
	}

	return nil
}
