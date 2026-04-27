package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("admin12345")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !CheckPassword(hash, "admin12345") {
		t.Fatal("expected password to match hash")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	userID := uuid.New()
	companyID := uuid.New()

	token, _, err := GenerateAccessToken("test-secret", time.Minute, userID, companyID, "ADMIN")
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	claims, err := ParseAccessToken("test-secret", token)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}
	if claims.UserID != userID || claims.CompanyID != companyID || claims.Role != "ADMIN" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
