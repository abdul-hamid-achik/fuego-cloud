// Package cloudflare provides DNS management via Cloudflare API.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Cloudflare API interactions
type Client struct {
	apiToken string
	zoneID   string
	http     *http.Client
}

// NewClient creates a new Cloudflare client
func NewClient(apiToken, zoneID string) *Client {
	return &Client{
		apiToken: apiToken,
		zoneID:   zoneID,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	Priority int    `json:"priority,omitempty"`
}

// APIResponse represents a Cloudflare API response
type APIResponse struct {
	Success  bool        `json:"success"`
	Errors   []APIError  `json:"errors"`
	Messages []string    `json:"messages"`
	Result   interface{} `json:"result"`
}

// APIError represents a Cloudflare API error
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CreateCNAME creates a CNAME record pointing to the platform domain
func (c *Client) CreateCNAME(ctx context.Context, subdomain, target string) (*DNSRecord, error) {
	record := DNSRecord{
		Type:    "CNAME",
		Name:    subdomain,
		Content: target,
		TTL:     1, // Auto TTL
		Proxied: true,
	}

	body, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", c.zoneID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare request failed")
	}

	// Parse the result
	resultBytes, err := json.Marshal(apiResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var createdRecord DNSRecord
	if err := json.Unmarshal(resultBytes, &createdRecord); err != nil {
		return nil, fmt.Errorf("failed to parse created record: %w", err)
	}

	return &createdRecord, nil
}

// DeleteRecord deletes a DNS record by ID
func (c *Client) DeleteRecord(ctx context.Context, recordID string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.zoneID, recordID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return fmt.Errorf("cloudflare error: %s", apiResp.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare request failed")
	}

	return nil
}

// GetRecordByName finds a DNS record by name
func (c *Client) GetRecordByName(ctx context.Context, name string) (*DNSRecord, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", c.zoneID, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp struct {
		Success bool        `json:"success"`
		Errors  []APIError  `json:"errors"`
		Result  []DNSRecord `json:"result"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("cloudflare request failed")
	}

	if len(apiResp.Result) == 0 {
		return nil, nil // Not found
	}

	return &apiResp.Result[0], nil
}

// DomainVerification represents domain verification status.
type DomainVerification struct {
	Domain    string `json:"domain"`
	Verified  bool   `json:"verified"`
	DNSRecord string `json:"dns_record,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Message   string `json:"message"`
}

// VerifyDomain checks if the domain points to the correct target
func (c *Client) VerifyDomain(ctx context.Context, domain, expectedTarget string) (*DomainVerification, error) {
	record, err := c.GetRecordByName(ctx, domain)
	if err != nil {
		return &DomainVerification{
			Domain:   domain,
			Verified: false,
			Expected: expectedTarget,
			Message:  fmt.Sprintf("Failed to check DNS: %v", err),
		}, nil
	}

	if record == nil {
		return &DomainVerification{
			Domain:   domain,
			Verified: false,
			Expected: expectedTarget,
			Message:  "No DNS record found. Please add a CNAME record.",
		}, nil
	}

	if record.Content != expectedTarget {
		return &DomainVerification{
			Domain:    domain,
			Verified:  false,
			DNSRecord: record.Content,
			Expected:  expectedTarget,
			Message:   fmt.Sprintf("DNS record points to %s instead of %s", record.Content, expectedTarget),
		}, nil
	}

	return &DomainVerification{
		Domain:    domain,
		Verified:  true,
		DNSRecord: record.Content,
		Expected:  expectedTarget,
		Message:   "Domain is properly configured",
	}, nil
}

// SetupAppDomain creates the DNS record for an app subdomain
func (c *Client) SetupAppDomain(ctx context.Context, appName, platformDomain string) (*DNSRecord, error) {
	subdomain := appName + "." + platformDomain
	return c.CreateCNAME(ctx, subdomain, platformDomain)
}
