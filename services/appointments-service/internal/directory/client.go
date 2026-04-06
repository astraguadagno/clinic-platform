package directory

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("directory service unavailable")

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse directory base url: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("parse directory base url: invalid base url")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}

	return &Client{
		baseURL:    parsedURL,
		httpClient: httpClient,
	}, nil
}

func (c *Client) ProfessionalExists(ctx context.Context, professionalID string) (bool, error) {
	return c.lookup(ctx, path.Join("professionals", professionalID))
}

func (c *Client) PatientExists(ctx context.Context, patientID string) (bool, error) {
	return c.lookup(ctx, path.Join("patients", patientID))
}

func (c *Client) lookup(ctx context.Context, resourcePath string) (bool, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, resourcePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, fmt.Errorf("create directory request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
}
