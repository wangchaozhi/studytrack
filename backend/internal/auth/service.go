package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID string `json:"sub"`
	Expiry int64  `json:"exp"`
	Issued int64  `json:"iat"`
}

type Service struct {
	secret []byte
	ttl    time.Duration
}

func NewService(secret []byte, ttl time.Duration) *Service {
	return &Service{secret: secret, ttl: ttl}
}

func (s *Service) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("密码至少需要 8 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *Service) IssueToken(userID string) (string, error) {
	now := time.Now()
	claims := Claims{UserID: userID, Issued: now.Unix(), Expiry: now.Add(s.ttl).Unix()}
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := encode(header) + "." + encode(payload)
	signature := s.sign(unsigned)
	return unsigned + "." + signature, nil
}

func (s *Service) ParseToken(token string) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := s.sign(unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return claims, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("decode token: %w", err)
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, fmt.Errorf("parse claims: %w", err)
	}
	if claims.UserID == "" || claims.Expiry < time.Now().Unix() {
		return claims, errors.New("token expired")
	}
	return claims, nil
}

func (s *Service) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(value))
	return encode(mac.Sum(nil))
}

func encode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
