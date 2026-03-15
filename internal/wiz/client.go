/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wiz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client defines the operations the controller needs against the Wiz API.
type Client interface {
	// CreateRule creates a new Cloud Configuration Rule and returns its Wiz-assigned ID.
	CreateRule(ctx context.Context, rule Rule) (string, error)
	// UpdateRule updates an existing Cloud Configuration Rule by its Wiz ID.
	UpdateRule(ctx context.Context, id string, rule Rule) error
	// DeleteRule deletes a Cloud Configuration Rule by its Wiz ID.
	DeleteRule(ctx context.Context, id string) error
}

// Rule holds the fields sent to the Wiz API for a Cloud Configuration Rule.
type Rule struct {
	Name                string
	Description         string
	Severity            string
	ProjectScope        string
	FrameworkCategories []string
	Tags                map[string]string
	TargetNativeType    string
	MatcherTypes        []MatcherType
	OperationTypes      []string
	Code                string
	RemediationSteps    string
}

// MatcherType identifies the kind of matcher used to evaluate a Cloud Configuration Rule.
type MatcherType string

const (
	// MatcherTypeAdmissionsController evaluates resources at Kubernetes admission time.
	MatcherTypeAdmissionsController MatcherType = "ADMISSIONS_CONTROLLER"
)

// Config holds credentials and endpoints for the Wiz API.
type Config struct {
	// ClientID is the Wiz service account client ID.
	ClientID string
	// ClientSecret is the Wiz service account client secret.
	ClientSecret string
	// TokenEndpoint is the OAuth2 token URL, e.g. https://auth.app.wiz.io/oauth/token
	TokenEndpoint string
	// APIEndpoint is the Wiz GraphQL URL, e.g. https://api.us1.app.wiz.io/graphql
	APIEndpoint string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphqlResponse[T any] struct {
	Data   T `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type wizClient struct {
	cfg      Config
	http     *http.Client
	token    string
	tokenExp time.Time
}

// NewClient returns a Client backed by the Wiz GraphQL API.
func NewClient(cfg Config) Client {
	return &wizClient{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// authenticate fetches a new OAuth2 token if the current one has expired.
func (c *wizClient) authenticate(ctx context.Context) error {
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("audience", "wiz-api")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.TokenEndpoint,
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("fetching token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, body)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return fmt.Errorf("decoding token response: %w", err)
	}

	c.token = tr.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tr.ExpiresIn)*time.Second - 30*time.Second)
	return nil
}

// do sends a GraphQL request and decodes the response into out.
func (c *wizClient) do(ctx context.Context, query string, variables map[string]any, out any) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing graphql request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("graphql endpoint returned %d: %s", resp.StatusCode, b)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Check for GraphQL-level errors before decoding into the typed response.
	var errCheck graphqlResponse[json.RawMessage]
	if err := json.Unmarshal(raw, &errCheck); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(errCheck.Errors) > 0 {
		msgs := make([]string, len(errCheck.Errors))
		for i, e := range errCheck.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("graphql errors: %s", strings.Join(msgs, "; "))
	}

	return json.Unmarshal(raw, out)
}

// CreateRule creates a new Cloud Configuration Rule in Wiz and returns its ID.
func (c *wizClient) CreateRule(ctx context.Context, rule Rule) (string, error) {
	const mutation = `
		mutation CreateCloudConfigurationRule($input: CreateCloudConfigurationRuleInput!) {
			createCloudConfigurationRule(input: $input) {
				rule {
					id
				}
			}
		}`

	type createResult struct {
		CreateCloudConfigurationRule struct {
			Rule struct {
				ID string `json:"id"`
			} `json:"rule"`
		} `json:"createCloudConfigurationRule"`
	}

	var result graphqlResponse[createResult]
	err := c.do(ctx, mutation, map[string]any{
		"input": ruleToInput(rule),
	}, &result)
	if err != nil {
		return "", err
	}

	return result.Data.CreateCloudConfigurationRule.Rule.ID, nil
}

// UpdateRule updates an existing Cloud Configuration Rule in Wiz.
func (c *wizClient) UpdateRule(ctx context.Context, id string, rule Rule) error {
	const mutation = `
		mutation UpdateCloudConfigurationRule($input: UpdateCloudConfigurationRuleInput!) {
			updateCloudConfigurationRule(input: $input) {
				rule {
					id
				}
			}
		}`

	type updateResult struct {
		UpdateCloudConfigurationRule struct {
			Rule struct {
				ID string `json:"id"`
			} `json:"rule"`
		} `json:"updateCloudConfigurationRule"`
	}

	input := ruleToInput(rule)
	input["id"] = id

	var result graphqlResponse[updateResult]
	return c.do(ctx, mutation, map[string]any{"input": input}, &result)
}

// DeleteRule deletes a Cloud Configuration Rule from Wiz.
func (c *wizClient) DeleteRule(ctx context.Context, id string) error {
	const mutation = `
		mutation DeleteCloudConfigurationRule($input: DeleteCloudConfigurationRuleInput!) {
			deleteCloudConfigurationRule(input: $input) {
				_stub
			}
		}`

	type deleteResult struct{}

	var result graphqlResponse[deleteResult]
	return c.do(ctx, mutation, map[string]any{
		"input": map[string]any{"id": id},
	}, &result)
}

// ruleToInput converts a Rule into a GraphQL input map.
func ruleToInput(r Rule) map[string]any {
	input := map[string]any{
		"name":             r.Name,
		"severity":         r.Severity,
		"targetNativeType": r.TargetNativeType,
		"matcherTypes":     r.MatcherTypes,
		"opaPolicy":        r.Code,
		"operationTypes":   r.OperationTypes,
	}
	if r.Description != "" {
		input["description"] = r.Description
	}
	if r.ProjectScope != "" {
		input["projectId"] = r.ProjectScope
	}
	if len(r.FrameworkCategories) > 0 {
		input["frameworkCategories"] = r.FrameworkCategories
	}
	if len(r.Tags) > 0 {
		input["tags"] = r.Tags
	}
	if r.RemediationSteps != "" {
		input["remediationSteps"] = r.RemediationSteps
	}
	return input
}
