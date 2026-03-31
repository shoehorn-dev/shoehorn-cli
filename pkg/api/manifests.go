package api

import (
	"context"
	"fmt"
)

// ManifestValidationResult represents the validation result from the API
type ManifestValidationResult struct {
	Valid  bool                      `json:"valid"`
	Errors []ManifestValidationError `json:"errors"`
}

// ManifestValidationError represents a validation error
type ManifestValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ManifestConversionRequest represents a conversion request
type ManifestConversionRequest struct {
	Content    string `json:"content"`
	TargetType string `json:"targetType"` // "shoehorn", "backstage", or "mold"
	Validate   bool   `json:"validate"`
}

// ManifestConversionResponse represents the conversion response
type ManifestConversionResponse struct {
	Success    bool                      `json:"success"`
	Content    string                    `json:"content,omitempty"` // For shoehorn/backstage
	Mold       map[string]any            `json:"mold,omitempty"`    // For mold
	Format     string                    `json:"format"`
	Validation *ManifestValidationResult `json:"validation,omitempty"`
}

// ValidateManifestRequest represents a validation request
type ValidateManifestRequest struct {
	Content string `json:"content"`
}

// ValidateManifestResponse represents the validation response
type ValidateManifestResponse struct {
	Valid  bool                      `json:"valid"`
	Errors []ManifestValidationError `json:"errors"`
}

// ValidateManifest validates a manifest file via the API.
// The validate endpoint returns 422 when validation fails (not an error - it's a valid result),
// so we use doIgnoreStatus to always parse the response body.
func (c *Client) ValidateManifest(ctx context.Context, content string) (*ValidateManifestResponse, error) {
	req := ValidateManifestRequest{
		Content: content,
	}

	var resp ValidateManifestResponse
	statusCode, err := c.doIgnoreStatus(ctx, "POST", "/api/v1/manifests/validate", req, &resp)
	if err != nil {
		return nil, err
	}

	// 5xx = server error
	if statusCode >= 500 {
		return nil, fmt.Errorf("API server error (%d)", statusCode)
	}

	// 401/403 = auth error
	if statusCode == 401 || statusCode == 403 {
		return nil, fmt.Errorf("API error (%d): not authorized", statusCode)
	}

	// 2xx or 4xx with a parsed body = validation result (valid or invalid)
	return &resp, nil
}

// ConvertManifest converts a manifest between formats via the API
func (c *Client) ConvertManifest(ctx context.Context, content string, targetType string, validate bool) (*ManifestConversionResponse, error) {
	req := ManifestConversionRequest{
		Content:    content,
		TargetType: targetType,
		Validate:   validate,
	}

	var resp ManifestConversionResponse
	if err := c.Post(ctx, "/api/v1/manifests/convert", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
