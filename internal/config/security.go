package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Security struct {
	SessionTTL  time.Duration
	TokenBytes  int
	PasswordMin int
	RequireTLS  bool
}

func DefaultSecurity() Security {
	return Security{SessionTTL: 8 * time.Hour, TokenBytes: 18, PasswordMin: 12, RequireTLS: false}
}
func (s Security) Validate() error {
	if s.SessionTTL < time.Minute {
		return fmt.Errorf("session ttl too short")
	}
	if s.TokenBytes < 16 {
		return fmt.Errorf("token too short")
	}
	if s.PasswordMin < 8 {
		return fmt.Errorf("password policy weak")
	}
	return nil
}
func (s Security) RandomToken() (string, error) {
	b := make([]byte, s.TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
