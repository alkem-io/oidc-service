package webhook

import (
	"context"
	"fmt"

	kratosClient "github.com/ory/client-go"
)

// KratosAdminClient wraps the Kratos API client to implement KratosAdmin interface.
type KratosAdminClient struct {
	api *kratosClient.APIClient
}

// NewKratosAdminClient creates a KratosAdminClient from an existing Ory client.
func NewKratosAdminClient(api *kratosClient.APIClient) *KratosAdminClient {
	return &KratosAdminClient{api: api}
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
