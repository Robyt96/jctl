package auth

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: jenkins-cli-tool, Property 13: Profile Credential Isolation
// Validates: Requirements 9.5, 9.6
// For any two different profiles, when credentials are stored for each profile,
// operations using one profile should never use credentials from another profile.
func TestProperty_ProfileCredentialIsolation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Generator for profile names
	genProfileName := gen.OneConstOf(
		"development",
		"staging",
		"production",
		"test",
		"ci",
		"local",
		"qa",
		"demo",
		"profile1",
		"profile2",
	)

	// Generator for token values
	genTokenValue := gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) >= 10 && len(s) <= 50
	})

	// Generator for usernames
	genUsername := gen.OneConstOf(
		"user1",
		"user2",
		"admin",
		"developer",
		"tester",
		"jenkins-user",
		"ci-user",
		"build-user",
	)

	// Generator for expiration timestamps
	// Use 0 for non-expiring tokens, or future timestamps (year 2030+)
	genExpiresAt := gen.OneGenOf(
		gen.Const(int64(0)),                    // Non-expiring token
		gen.Int64Range(1893456000, 2524608000), // Between 2030 and 2050
	)

	// Property 1: Storing credentials for different profiles should not interfere
	properties.Property("storing credentials for different profiles should not interfere", prop.ForAll(
		func(profileSuffix1 int, profileSuffix2 int, token1 string, token2 string, user1 string, user2 string, expires1 int64, expires2 int64) bool {
			// Create unique profile names using suffixes (guaranteed different by generator)
			profile1 := fmt.Sprintf("profile-%d", profileSuffix1)
			profile2 := fmt.Sprintf("profile-%d", profileSuffix2)

			// Create temporary credentials file
			tmpDir := t.TempDir()
			credFile := filepath.Join(tmpDir, "credentials")
			manager := NewManager(credFile)

			// Store token for profile1
			token1Obj := &Token{
				Value:     token1,
				Type:      "api_token",
				Username:  user1,
				ExpiresAt: expires1,
			}
			if err := manager.StoreToken(profile1, token1Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile1, err)
				return false
			}

			// Store token for profile2
			token2Obj := &Token{
				Value:     token2,
				Type:      "api_token",
				Username:  user2,
				ExpiresAt: expires2,
			}
			if err := manager.StoreToken(profile2, token2Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile2, err)
				return false
			}

			// Retrieve token for profile1
			retrievedToken1, err := manager.GetToken(profile1)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile1, err)
				return false
			}

			// Retrieve token for profile2
			retrievedToken2, err := manager.GetToken(profile2)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile2, err)
				return false
			}

			// Verify profile1 token matches what was stored for profile1
			if retrievedToken1.Value != token1 {
				t.Logf("Profile '%s' token mismatch: expected '%s', got '%s'", profile1, token1, retrievedToken1.Value)
				return false
			}
			if retrievedToken1.Username != user1 {
				t.Logf("Profile '%s' username mismatch: expected '%s', got '%s'", profile1, user1, retrievedToken1.Username)
				return false
			}
			if retrievedToken1.ExpiresAt != expires1 {
				t.Logf("Profile '%s' expires_at mismatch: expected %d, got %d", profile1, expires1, retrievedToken1.ExpiresAt)
				return false
			}

			// Verify profile2 token matches what was stored for profile2
			if retrievedToken2.Value != token2 {
				t.Logf("Profile '%s' token mismatch: expected '%s', got '%s'", profile2, token2, retrievedToken2.Value)
				return false
			}
			if retrievedToken2.Username != user2 {
				t.Logf("Profile '%s' username mismatch: expected '%s', got '%s'", profile2, user2, retrievedToken2.Username)
				return false
			}
			if retrievedToken2.ExpiresAt != expires2 {
				t.Logf("Profile '%s' expires_at mismatch: expected %d, got %d", profile2, expires2, retrievedToken2.ExpiresAt)
				return false
			}

			// Verify profile1 did not get profile2's credentials
			if retrievedToken1.Value == token2 && token1 != token2 {
				t.Logf("Profile '%s' incorrectly has profile '%s' token", profile1, profile2)
				return false
			}

			// Verify profile2 did not get profile1's credentials
			if retrievedToken2.Value == token1 && token1 != token2 {
				t.Logf("Profile '%s' incorrectly has profile '%s' token", profile2, profile1)
				return false
			}

			return true
		},
		gen.IntRange(0, 999),
		gen.IntRange(1000, 1999), // Different range to ensure different values
		genTokenValue,
		genTokenValue,
		genUsername,
		genUsername,
		genExpiresAt,
		genExpiresAt,
	))

	// Property 2: Updating one profile's credentials should not affect another profile
	properties.Property("updating one profile's credentials should not affect another profile", prop.ForAll(
		func(profileSuffix1 int, profileSuffix2 int, initialToken1 string, initialToken2 string, updatedToken1 string, user1 string, user2 string) bool {
			// Create unique profile names using suffixes (guaranteed different by generator)
			profile1 := fmt.Sprintf("profile-%d", profileSuffix1)
			profile2 := fmt.Sprintf("profile-%d", profileSuffix2)

			// Create temporary credentials file
			tmpDir := t.TempDir()
			credFile := filepath.Join(tmpDir, "credentials")
			manager := NewManager(credFile)

			// Store initial tokens for both profiles
			token1Obj := &Token{
				Value:     initialToken1,
				Type:      "api_token",
				Username:  user1,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile1, token1Obj); err != nil {
				t.Logf("Failed to store initial token for profile '%s': %v", profile1, err)
				return false
			}

			token2Obj := &Token{
				Value:     initialToken2,
				Type:      "api_token",
				Username:  user2,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile2, token2Obj); err != nil {
				t.Logf("Failed to store initial token for profile '%s': %v", profile2, err)
				return false
			}

			// Update profile1's token
			updatedToken1Obj := &Token{
				Value:     updatedToken1,
				Type:      "api_token",
				Username:  user1,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile1, updatedToken1Obj); err != nil {
				t.Logf("Failed to update token for profile '%s': %v", profile1, err)
				return false
			}

			// Retrieve both tokens
			retrievedToken1, err := manager.GetToken(profile1)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile1, err)
				return false
			}

			retrievedToken2, err := manager.GetToken(profile2)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile2, err)
				return false
			}

			// Verify profile1 has the updated token
			if retrievedToken1.Value != updatedToken1 {
				t.Logf("Profile '%s' should have updated token '%s', got '%s'", profile1, updatedToken1, retrievedToken1.Value)
				return false
			}

			// Verify profile2 still has its original token (unchanged)
			if retrievedToken2.Value != initialToken2 {
				t.Logf("Profile '%s' token should remain '%s', got '%s'", profile2, initialToken2, retrievedToken2.Value)
				return false
			}

			return true
		},
		gen.IntRange(0, 999),
		gen.IntRange(1000, 1999), // Different range to ensure different values
		genTokenValue,
		genTokenValue,
		genTokenValue,
		genUsername,
		genUsername,
	))

	// Property 3: Deleting one profile's credentials should not affect another profile
	properties.Property("deleting one profile's credentials should not affect another profile", prop.ForAll(
		func(profileSuffix1 int, profileSuffix2 int, token1 string, token2 string, user1 string, user2 string) bool {
			// Create unique profile names using suffixes (guaranteed different by generator)
			profile1 := fmt.Sprintf("profile-%d", profileSuffix1)
			profile2 := fmt.Sprintf("profile-%d", profileSuffix2)

			// Create temporary credentials file
			tmpDir := t.TempDir()
			credFile := filepath.Join(tmpDir, "credentials")
			manager := NewManager(credFile)

			// Store tokens for both profiles
			token1Obj := &Token{
				Value:     token1,
				Type:      "api_token",
				Username:  user1,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile1, token1Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile1, err)
				return false
			}

			token2Obj := &Token{
				Value:     token2,
				Type:      "api_token",
				Username:  user2,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile2, token2Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile2, err)
				return false
			}

			// Delete profile1's credentials
			if err := manager.ClearToken(profile1); err != nil {
				t.Logf("Failed to clear token for profile '%s': %v", profile1, err)
				return false
			}

			// Verify profile1's credentials are gone
			_, err := manager.GetToken(profile1)
			if err == nil {
				t.Logf("Profile '%s' credentials should be deleted but still exist", profile1)
				return false
			}

			// Verify profile2's credentials still exist and are unchanged
			retrievedToken2, err := manager.GetToken(profile2)
			if err != nil {
				t.Logf("Profile '%s' credentials should still exist after deleting profile '%s': %v", profile2, profile1, err)
				return false
			}

			if retrievedToken2.Value != token2 {
				t.Logf("Profile '%s' token should remain '%s', got '%s'", profile2, token2, retrievedToken2.Value)
				return false
			}
			if retrievedToken2.Username != user2 {
				t.Logf("Profile '%s' username should remain '%s', got '%s'", profile2, user2, retrievedToken2.Username)
				return false
			}

			return true
		},
		gen.IntRange(0, 999),
		gen.IntRange(1000, 1999), // Different range to ensure different values
		genTokenValue,
		genTokenValue,
		genUsername,
		genUsername,
	))

	// Property 4: Multiple profiles can coexist with independent credentials
	properties.Property("multiple profiles can coexist with independent credentials", prop.ForAll(
		func(profiles []string, tokens []string, usernames []string) bool {
			// Need at least 2 profiles
			if len(profiles) < 2 || len(tokens) < 2 || len(usernames) < 2 {
				return true
			}

			// Ensure profiles are unique
			profileSet := make(map[string]bool)
			uniqueProfiles := []string{}
			for _, p := range profiles {
				if !profileSet[p] {
					profileSet[p] = true
					uniqueProfiles = append(uniqueProfiles, p)
				}
			}

			// Need at least 2 unique profiles
			if len(uniqueProfiles) < 2 {
				return true
			}

			// Use only first 2 unique profiles for simplicity
			profile1 := uniqueProfiles[0]
			profile2 := uniqueProfiles[1]
			token1 := tokens[0]
			token2 := tokens[1]
			user1 := usernames[0]
			user2 := usernames[1]

			// Create temporary credentials file
			tmpDir := t.TempDir()
			credFile := filepath.Join(tmpDir, "credentials")
			manager := NewManager(credFile)

			// Store credentials for both profiles
			token1Obj := &Token{
				Value:     token1,
				Type:      "api_token",
				Username:  user1,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile1, token1Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile1, err)
				return false
			}

			token2Obj := &Token{
				Value:     token2,
				Type:      "api_token",
				Username:  user2,
				ExpiresAt: 0,
			}
			if err := manager.StoreToken(profile2, token2Obj); err != nil {
				t.Logf("Failed to store token for profile '%s': %v", profile2, err)
				return false
			}

			// List all profiles with credentials
			profileList, err := manager.ListProfiles()
			if err != nil {
				t.Logf("Failed to list profiles: %v", err)
				return false
			}

			// Verify both profiles are in the list
			foundProfile1 := false
			foundProfile2 := false
			for _, p := range profileList {
				if p == profile1 {
					foundProfile1 = true
				}
				if p == profile2 {
					foundProfile2 = true
				}
			}

			if !foundProfile1 {
				t.Logf("Profile '%s' not found in profile list", profile1)
				return false
			}
			if !foundProfile2 {
				t.Logf("Profile '%s' not found in profile list", profile2)
				return false
			}

			// Verify each profile has its own credentials
			retrievedToken1, err := manager.GetToken(profile1)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile1, err)
				return false
			}

			retrievedToken2, err := manager.GetToken(profile2)
			if err != nil {
				t.Logf("Failed to retrieve token for profile '%s': %v", profile2, err)
				return false
			}

			// Verify credentials are correct and isolated
			if retrievedToken1.Value != token1 {
				t.Logf("Profile '%s' token mismatch", profile1)
				return false
			}
			if retrievedToken2.Value != token2 {
				t.Logf("Profile '%s' token mismatch", profile2)
				return false
			}

			// Verify no cross-contamination (if tokens are different)
			if token1 != token2 {
				if retrievedToken1.Value == token2 {
					t.Logf("Profile '%s' has profile '%s' token", profile1, profile2)
					return false
				}
				if retrievedToken2.Value == token1 {
					t.Logf("Profile '%s' has profile '%s' token", profile2, profile1)
					return false
				}
			}

			return true
		},
		gen.SliceOf(genProfileName),
		gen.SliceOf(genTokenValue),
		gen.SliceOf(genUsername),
	))

	properties.TestingRun(t)
}
