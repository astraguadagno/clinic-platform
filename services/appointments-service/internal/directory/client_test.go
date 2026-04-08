package directory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrentUserReturnsDecodedUserAndForwardsBearer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer session-token" {
			t.Fatalf("authorization = %q, want %q", got, "Bearer session-token")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1","email":"doctor@clinic.local","role":"doctor","professional_id":"550e8400-e29b-41d4-a716-446655440000","active":true}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	user, err := client.CurrentUser(context.Background(), "session-token")
	if err != nil {
		t.Fatalf("CurrentUser error = %v", err)
	}
	if user.Role != "doctor" {
		t.Fatalf("role = %q, want doctor", user.Role)
	}
	if user.ProfessionalID == nil || *user.ProfessionalID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("professional_id = %v, want doctor professional id", user.ProfessionalID)
	}
}

func TestCurrentUserReturnsUnauthorized(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	_, err = client.CurrentUser(context.Background(), "missing-token")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want %v", err, ErrUnauthorized)
	}
}

func TestCurrentUserReturnsUnavailableOnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	_, err = client.CurrentUser(context.Background(), "session-token")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want %v", err, ErrUnavailable)
	}
}
