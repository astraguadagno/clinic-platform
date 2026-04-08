package directory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

var (
	ErrUnavailable  = errors.New("directory service unavailable")
	ErrUnauthorized = errors.New("directory unauthorized")
)

type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	ProfessionalID *string    `json:"professional_id,omitempty"`
	Active         bool       `json:"active"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

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

func (c *Client) CurrentUser(ctx context.Context, bearer string) (User, error) {
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return User{}, ErrUnauthorized
	}

	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, "auth", "me")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return User{}, fmt.Errorf("create directory request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var user User
		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			return User{}, fmt.Errorf("decode current user: %w", err)
		}
		return user, nil
	case http.StatusUnauthorized:
		return User{}, ErrUnauthorized
	default:
		return User{}, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
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
