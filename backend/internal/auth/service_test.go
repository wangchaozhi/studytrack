package auth

import (
	"testing"
	"time"
)

func TestPasswordAndToken(t *testing.T) {
	svc := NewService([]byte("test-secret"), time.Hour)
	hash, err := svc.HashPassword("password123")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ComparePassword(hash, "password123"); err != nil {
		t.Fatalf("password should match: %v", err)
	}
	token, err := svc.IssueToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("unexpected user: %s", claims.UserID)
	}
}

func TestRejectsWeakPasswordAndTamperedToken(t *testing.T) {
	svc := NewService([]byte("test-secret"), time.Hour)
	if _, err := svc.HashPassword("short"); err == nil {
		t.Fatal("weak password should be rejected")
	}
	token, err := svc.IssueToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ParseToken(token + "tampered"); err == nil {
		t.Fatal("tampered token should be rejected")
	}
}
