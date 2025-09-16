package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test to verify authentication flow step by step
func TestAuthenticationFlow(t *testing.T) {
	client := NewAPIClient("http://localhost:8080")

	t.Run("Complete authentication flow", func(t *testing.T) {
		testEmail := "test@example.com"

		// Step 1: Send magic link
		requestBody := map[string]string{
			"email": testEmail,
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), "POST", client.baseURL+"/v1/auth/magic-link", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Magic link send should succeed
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var magicResp struct {
			Sent             bool  `json:"sent"`
			ExpiresInSeconds int32 `json:"expiresInSeconds"`
		}
		err = json.NewDecoder(resp.Body).Decode(&magicResp)
		require.NoError(t, err)

		require.True(t, magicResp.Sent, "Magic link should be sent successfully")
		require.Greater(t, magicResp.ExpiresInSeconds, int32(0), "Should have positive expiration time")

		t.Logf("✅ Magic link sent successfully to %s", testEmail)

		// Step 2: Wait briefly for email delivery
		time.Sleep(100 * time.Millisecond)

		// Step 3: Extract API key from MailHog
		emailReq, err := http.NewRequestWithContext(context.Background(), "GET", "http://localhost:8025/api/v2/messages", http.NoBody)
		require.NoError(t, err)

		emailResp, err := client.client.Do(emailReq)
		require.NoError(t, err)
		defer emailResp.Body.Close()

		var mailhogResp struct {
			Items []struct {
				To []struct {
					Mailbox string `json:"Mailbox"`
					Domain  string `json:"Domain"`
				} `json:"To"`
				Content struct {
					Body string `json:"Body"`
				} `json:"Content"`
			} `json:"items"`
		}

		err = json.NewDecoder(emailResp.Body).Decode(&mailhogResp)
		require.NoError(t, err)

		var apiKey string
		// Find the latest email to our test address
		for _, item := range mailhogResp.Items {
			for _, to := range item.To {
				if to.Mailbox+"@"+to.Domain == testEmail {
					// Extract API key from email body
					body := item.Content.Body
					if strings.Contains(body, "apikey=") {
						start := strings.Index(body, "apikey=") + 7
						end := strings.Index(body[start:], "\r")
						if end == -1 {
							end = strings.Index(body[start:], "\n")
						}
						if end == -1 {
							end = len(body[start:])
						}
						apiKey = body[start : start+end]
						break
					}
				}
			}
			if apiKey != "" {
				break
			}
		}

		require.NotEmpty(t, apiKey, "Should extract API key from email")
		t.Logf("📧 Extracted API key: %s", apiKey)

		// Step 4: Validate the API key and get authentication token
		validateBody := map[string]string{
			"token": apiKey,
		}

		validateJSON, err := json.Marshal(validateBody)
		require.NoError(t, err)

		validateReq, err := http.NewRequestWithContext(context.Background(), "POST", client.baseURL+"/v1/auth/validate", bytes.NewBuffer(validateJSON))
		require.NoError(t, err)
		validateReq.Header.Set("Content-Type", "application/json")

		validateResp, err := client.client.Do(validateReq)
		require.NoError(t, err)
		defer validateResp.Body.Close()

		require.Equal(t, http.StatusOK, validateResp.StatusCode, "API key validation should succeed")

		// FIXED: Use ApiToken field name that matches the actual API response
		var authResp struct {
			ApiToken string `json:"apiToken"`
			User     struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		}
		err = json.NewDecoder(validateResp.Body).Decode(&authResp)
		require.NoError(t, err)

		require.NotEmpty(t, authResp.ApiToken, "Should receive authentication token")
		require.Equal(t, testEmail, authResp.User.Email, "User email should match")

		t.Logf("🔐 Authentication successful! Token: %s", authResp.ApiToken[:20]+"...")
		t.Logf("👤 User ID: %s", authResp.User.ID)

		// Step 5: Test authenticated request - create a club
		clubBody := map[string]string{
			"name": "Test Club via API",
		}

		clubJSON, err := json.Marshal(clubBody)
		require.NoError(t, err)

		clubReq, err := http.NewRequestWithContext(context.Background(), "POST", client.baseURL+"/v1/clubs", bytes.NewBuffer(clubJSON))
		require.NoError(t, err)
		clubReq.Header.Set("Content-Type", "application/json")
		clubReq.Header.Set("Authorization", "Bearer "+authResp.ApiToken)

		clubResp, err := client.client.Do(clubReq)
		require.NoError(t, err)
		defer clubResp.Body.Close()

		require.Equal(t, http.StatusOK, clubResp.StatusCode, "Authenticated club creation should succeed")

		var createdClub struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		err = json.NewDecoder(clubResp.Body).Decode(&createdClub)
		require.NoError(t, err)

		require.NotEmpty(t, createdClub.ID, "Created club should have an ID")
		require.Equal(t, "Test Club via API", createdClub.Name, "Club name should match")

		t.Logf("🏓 Club created successfully! ID: %s, Name: %s", createdClub.ID, createdClub.Name)
	})
}

// Test to verify that club creation properly requires authentication.
// This test validates that the service implementation correctly enforces
// authentication requirements as designed, preventing unauthorized club creation.
//
// Note: The original bug we were investigating (AddClubMembership with SetUpsert(false))
// can only be properly tested with authenticated requests. These tests confirm that
// the security layer is working correctly by requiring authentication first.
func TestClubCreationAuthentication(t *testing.T) {
	client := NewAPIClient("http://localhost:8080")

	t.Run("Verify club creation requires authentication", func(t *testing.T) {
		clubName := "Bug Test Club"

		// Create club request
		reqBody := map[string]string{"name": clubName}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), "POST",
			client.baseURL+"/v1/clubs", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Club creation should require authentication
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		t.Logf("✅ Club creation correctly requires authentication (401 Unauthorized)")
		t.Logf("This confirms the service implementation matches the security requirements")
	})
}

// Test to verify that all player and club operations require authentication
func TestPlayerOperationsAuthentication(t *testing.T) {
	client := NewAPIClient("http://localhost:8080")

	t.Run("Verify authentication is required for all operations", func(t *testing.T) {
		// First test club creation
		clubName := "Merge Test Club"
		reqBody := map[string]string{"name": clubName}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(context.Background(), "POST",
			client.baseURL+"/v1/clubs", bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		t.Logf("✅ Club creation requires authentication")

		// Test player creation also requires authentication
		createPlayerBody := map[string]string{
			"displayName": "Test Player",
			"clubId":      "dummy-club-id",
		}
		body2, err := json.Marshal(createPlayerBody)
		require.NoError(t, err)

		req2, err := http.NewRequestWithContext(context.Background(), "POST",
			client.baseURL+"/v1/players", bytes.NewBuffer(body2))
		require.NoError(t, err)
		req2.Header.Set("Content-Type", "application/json")

		resp2, err := client.client.Do(req2)
		require.NoError(t, err)