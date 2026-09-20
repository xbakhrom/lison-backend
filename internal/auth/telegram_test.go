package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestValidateInitData(t *testing.T) {
	now := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	token := "123:secret"
	values := url.Values{}
	values.Set("auth_date", "1789552800")
	values.Set("query_id", "query")
	values.Set("user", `{"id":42,"first_name":"Ali","username":"ali"}`)
	values.Set("hash", sign(values, token))

	user, err := ValidateInitData(values.Encode(), token, 24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 42 || user.FirstName != "Ali" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func sign(values url.Values, token string) string {
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
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	_, _ = mac.Write([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}
