package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestCreateTokenLocksTheSessionDown(t *testing.T) {
	var received map[string]any
	var apiKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey = r.Header.Get("x-goog-api-key")
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"auth_tokens/abc123"}`))
	}))
	defer server.Close()

	client := NewClient("secret-key", "gemini-3.8-live")
	client.endpoint = server.URL

	token, err := client.CreateToken(context.Background(), SessionConstraints{
		SystemInstruction: "You are Maks.",
		VoiceName:         "Puck",
		Language:          "ru-RU",
	}, time.Minute, 15*time.Minute)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	if token.Name != "auth_tokens/abc123" {
		t.Errorf("token name = %q, want auth_tokens/abc123", token.Name)
	}
	if apiKey != "secret-key" {
		t.Errorf("api key header = %q, want secret-key", apiKey)
	}
	if uses, _ := received["uses"].(float64); uses != 1 {
		t.Errorf("uses = %v, want 1", received["uses"])
	}
	if _, ok := received["newSessionExpireTime"].(string); !ok {
		t.Error("newSessionExpireTime is missing, so a leaked token would stay usable")
	}

	// The API rejects the SDK-level "liveConnectConstraints" spelling outright;
	// the wire field is a flat bidiGenerateContentSetup.
	if _, wrong := received["liveConnectConstraints"]; wrong {
		t.Error("request uses liveConnectConstraints, which the API does not accept")
	}
	setup, ok := received["bidiGenerateContentSetup"].(map[string]any)
	if !ok {
		t.Fatal("bidiGenerateContentSetup is missing, so the client could pick any model")
	}
	if setup["model"] != "models/gemini-3.8-live" {
		t.Errorf("model = %v, want models/gemini-3.8-live", setup["model"])
	}

	// responseModalities and speechConfig live under generationConfig in the
	// Live setup message, not at the top level.
	generation, ok := setup["generationConfig"].(map[string]any)
	if !ok {
		t.Fatal("generationConfig is missing")
	}
	modalities, _ := generation["responseModalities"].([]any)
	if len(modalities) != 1 || modalities[0] != "AUDIO" {
		t.Errorf("responseModalities = %v, want [AUDIO]", generation["responseModalities"])
	}
	if _, ok := generation["speechConfig"]; !ok {
		t.Error("speechConfig is missing, so the voice is not pinned")
	}

	instruction, _ := setup["systemInstruction"].(map[string]any)
	parts, _ := instruction["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("system instruction parts = %v, want exactly one", parts)
	}
	if first, _ := parts[0].(map[string]any); first["text"] != "You are Maks." {
		t.Errorf("system instruction = %v, want the persona", parts[0])
	}
	for _, field := range []string{"inputAudioTranscription", "outputAudioTranscription", "contextWindowCompression"} {
		if _, ok := setup[field]; !ok {
			t.Errorf("setup is missing %s", field)
		}
	}
	// The client has to stay free to resume, so resumption must not be pinned.
	if _, pinned := setup["sessionResumption"]; pinned {
		t.Error("sessionResumption is pinned, which would block reconnecting with a handle")
	}

	// Without a field mask the API locks the whole session config, which would
	// also freeze the tools and the resumption handle the browser must set.
	mask, _ := received["fieldMask"].(string)
	for _, field := range []string{
		"model",
		"generationConfig.responseModalities",
		"generationConfig.speechConfig",
		"systemInstruction.parts",
		"inputAudioTranscription",
		"outputAudioTranscription",
		"contextWindowCompression.slidingWindow",
	} {
		if !slices.Contains(strings.Split(mask, ","), field) {
			t.Errorf("fieldMask %q does not pin %s", mask, field)
		}
	}
	for _, free := range []string{"tools", "sessionResumption"} {
		if slices.Contains(strings.Split(mask, ","), free) {
			t.Errorf("fieldMask %q pins %s, which the client has to set per connection", mask, free)
		}
	}
}

func TestCreateTokenWithoutAVoiceLeavesSpeechUnpinned(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_, _ = w.Write([]byte(`{"name":"auth_tokens/abc123"}`))
	}))
	defer server.Close()

	client := NewClient("secret-key", "gemini-3.8-live")
	client.endpoint = server.URL
	if _, err := client.CreateToken(context.Background(), SessionConstraints{}, time.Minute, time.Minute); err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	setup, _ := received["bidiGenerateContentSetup"].(map[string]any)
	generation, _ := setup["generationConfig"].(map[string]any)
	if _, ok := generation["speechConfig"]; ok {
		t.Error("speechConfig was sent even though no voice or language was requested")
	}
	mask, _ := received["fieldMask"].(string)
	if slices.Contains(strings.Split(mask, ","), "generationConfig.speechConfig") {
		t.Errorf("fieldMask %q pins a speechConfig that was never sent", mask)
	}
}

func TestCreateTokenWithoutAPIKey(t *testing.T) {
	client := NewClient("", "gemini-3.8-live")
	if client.Configured() {
		t.Error("Configured() = true without an API key")
	}
	if _, err := client.CreateToken(context.Background(), SessionConstraints{}, time.Minute, time.Minute); err != ErrNotConfigured {
		t.Errorf("CreateToken error = %v, want ErrNotConfigured", err)
	}
}

func TestCreateTokenReportsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"permission denied"}}`))
	}))
	defer server.Close()

	client := NewClient("secret-key", "gemini-3.8-live")
	client.endpoint = server.URL
	if _, err := client.CreateToken(context.Background(), SessionConstraints{}, time.Minute, time.Minute); err == nil {
		t.Fatal("CreateToken succeeded on a 403")
	}
}

func TestQualifiedModel(t *testing.T) {
	for input, want := range map[string]string{
		"gemini-3.8-live":        "models/gemini-3.8-live",
		"models/gemini-3.8-live": "models/gemini-3.8-live",
	} {
		if got := qualifiedModel(input); got != want {
			t.Errorf("qualifiedModel(%q) = %q, want %q", input, got, want)
		}
	}
}
