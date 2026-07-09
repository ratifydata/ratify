package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateJWT(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	tokenString, err := GenerateJWT(userID, orgID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tokenString == "" {
		t.Fatal("expected token string to not be empty")
	}

	claims, err := VerifyToken(tokenString)
	if err != nil {
		t.Fatalf("expected generated token to verify, got %v", err)
	}

	if claims.UserId != userID {
		t.Errorf("got user id %s, want %s", claims.UserId, userID)
	}

	if claims.OrgId != orgID {
		t.Errorf("got org id %s, want %s", claims.OrgId, orgID)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected expires at to be set")
	}
	if time.Until(claims.ExpiresAt.Time) <= 23*time.Hour {
		t.Errorf("expected token expiry to be close to 24 hours, got %s", time.Until(claims.ExpiresAt.Time))
	}
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	if claims, err := VerifyToken("not-a-jwt"); err == nil {
		t.Fatalf("expected invalid token to return error, got claims %+v", claims)
	}
}

func TestVerifyToken_ExpiredToken(t *testing.T) {
	tokenString := signTestToken(t, CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
		UserId: uuid.New(),
		OrgId:  uuid.New(),
	})

	if claims, err := VerifyToken(tokenString); err == nil {
		t.Fatalf("expected expired token to return error, got claims %+v", claims)
	}
}

func TestVerifyToken_UnexpectedSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodNone, CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		UserId: uuid.New(),
		OrgId:  uuid.New(),
	})

	tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign none-alg token: %v", err)
	}

	if claims, err := VerifyToken(tokenString); err == nil {
		t.Fatalf("expected unexpected signing method to return error, got claims %+v", claims)
	}
}

func signTestToken(t *testing.T, claims CustomClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}

	return tokenString
}
