package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidInitData = errors.New("invalid Telegram init data")
	ErrExpiredInitData = errors.New("expired Telegram init data")
)

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

func ValidateInitData(raw, botToken string, maxAge time.Duration, now time.Time) (User, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return User{}, fmt.Errorf("%w: parse query", ErrInvalidInitData)
	}
	providedHash := values.Get("hash")
	if providedHash == "" || botToken == "" {
		return User{}, fmt.Errorf("%w: missing hash or bot token", ErrInvalidInitData)
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	dataCheckString := strings.Join(parts, "\n")

	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMAC.Write([]byte(botToken))
	secretKey := secretMAC.Sum(nil)
	checkMAC := hmac.New(sha256.New, secretKey)
	_, _ = checkMAC.Write([]byte(dataCheckString))
	expectedHash := checkMAC.Sum(nil)
	providedHashBytes, err := hex.DecodeString(providedHash)
	if err != nil || !hmac.Equal(expectedHash, providedHashBytes) {
		return User{}, ErrInvalidInitData
	}

	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil {
		return User{}, fmt.Errorf("%w: invalid auth_date", ErrInvalidInitData)
	}
	authTime := time.Unix(authUnix, 0)
	if authTime.After(now.Add(5*time.Minute)) || now.Sub(authTime) > maxAge {
		return User{}, ErrExpiredInitData
	}

	var user User
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil || user.ID == 0 {
		return User{}, fmt.Errorf("%w: invalid user", ErrInvalidInitData)
	}
	return user, nil
}
