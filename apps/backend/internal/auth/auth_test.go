package auth

import "testing"

func TestTokenKindEnforced(t *testing.T) {
	m := NewManager("test-secret")
	pair, err := m.Issue("user-123")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	uid, err := m.ParseAccess(pair.AccessToken)
	if err != nil || uid != "user-123" {
		t.Fatalf("parse access: uid=%q err=%v", uid, err)
	}
	if _, err := m.ParseRefresh(pair.AccessToken); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
	if _, err := m.ParseAccess(pair.RefreshToken); err == nil {
		t.Fatal("refresh token accepted as access token")
	}
}

func TestTokenWrongSecretRejected(t *testing.T) {
	pair, _ := NewManager("secret-a").Issue("u1")
	if _, err := NewManager("secret-b").ParseAccess(pair.AccessToken); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}
}
