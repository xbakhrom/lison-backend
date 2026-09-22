// Package gemini mints short-lived Live API tokens so the Mini App can open a
// WebSocket to Google directly. The long-lived API key never leaves the server.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultEndpoint = "https://generativelanguage.googleapis.com/v1beta/auth_tokens"

// ErrNotConfigured is returned when no API key is set, so callers can answer
// with a clean "assistant is off" instead of a server error.
var ErrNotConfigured = errors.New("gemini: API key is not configured")

type Client struct {
	apiKey   string
	model    string
	endpoint string
	http     *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:   apiKey,
		model:    model,
		endpoint: defaultEndpoint,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Configured reports whether tokens can be minted at all.
func (c *Client) Configured() bool { return c != nil && c.apiKey != "" }

// SessionConstraints pin what the token may be used for. Everything listed here
// is fixed by the server: a client holding the token cannot swap the model or
// rewrite the persona.
type SessionConstraints struct {
	SystemInstruction string
	VoiceName         string
	Language          string
}

// Token is a minted ephemeral token: Name is passed to the SDK as the API key.
type Token struct {
	Name                string    `json:"name"`
	ExpiresAt           time.Time `json:"expiresAt"`
	NewSessionExpiresAt time.Time `json:"newSessionExpiresAt"`
	Model               string    `json:"model"`
}

// The wire names below are not the ones the Google SDKs expose. What an SDK
// calls `liveConnectConstraints{model, config}` is sent as a flat
// `bidiGenerateContentSetup`, and the Live setup message nests
// responseModalities and speechConfig under generationConfig. The API rejects
// the SDK-level spelling outright.
type tokenRequest struct {
	Uses                 int           `json:"uses"`
	ExpireTime           string        `json:"expireTime"`
	NewSessionExpireTime string        `json:"newSessionExpireTime"`
	Setup                *sessionSetup `json:"bidiGenerateContentSetup,omitempty"`
	// FieldMask lists exactly which setup fields the token pins. Without it the
	// API locks the whole session config, which would also freeze the tools and
	// the resumption handle the browser has to supply per connection.
	FieldMask string `json:"fieldMask,omitempty"`
}

type sessionSetup struct {
	Model                    string            `json:"model"`
	GenerationConfig         *generationConfig `json:"generationConfig,omitempty"`
	SystemInstruction        *content          `json:"systemInstruction,omitempty"`
	InputAudioTranscription  *emptyStruct      `json:"inputAudioTranscription,omitempty"`
	OutputAudioTranscription *emptyStruct      `json:"outputAudioTranscription,omitempty"`
	ContextWindowCompression *compressConfig   `json:"contextWindowCompression,omitempty"`
}

type generationConfig struct {
	ResponseModalities []string      `json:"responseModalities,omitempty"`
	SpeechConfig       *speechConfig `json:"speechConfig,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type speechConfig struct {
	VoiceConfig  *voiceConfig `json:"voiceConfig,omitempty"`
	LanguageCode string       `json:"languageCode,omitempty"`
}

type voiceConfig struct {
	PrebuiltVoiceConfig prebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}

type prebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type compressConfig struct {
	SlidingWindow emptyStruct `json:"slidingWindow"`
}

type emptyStruct struct{}

// CreateToken mints a single-use token. newSessionWindow is how long the client
// has to open the WebSocket; lifetime is how long the resulting session may run
// before the token stops working.
func (c *Client) CreateToken(ctx context.Context, constraints SessionConstraints, newSessionWindow, lifetime time.Duration) (Token, error) {
	if !c.Configured() {
		return Token{}, ErrNotConfigured
	}
	now := time.Now().UTC()
	expireAt := now.Add(lifetime)
	newSessionExpireAt := now.Add(newSessionWindow)

	setup := sessionSetup{
		Model:                    qualifiedModel(c.model),
		GenerationConfig:         &generationConfig{ResponseModalities: []string{"AUDIO"}},
		InputAudioTranscription:  &emptyStruct{},
		OutputAudioTranscription: &emptyStruct{},
		ContextWindowCompression: &compressConfig{},
	}
	mask := []string{"model", "generationConfig.responseModalities"}
	if constraints.VoiceName != "" || constraints.Language != "" {
		setup.GenerationConfig.SpeechConfig = &speechConfig{LanguageCode: constraints.Language}
		if constraints.VoiceName != "" {
			setup.GenerationConfig.SpeechConfig.VoiceConfig = &voiceConfig{PrebuiltVoiceConfig: prebuiltVoiceConfig{VoiceName: constraints.VoiceName}}
		}
		mask = append(mask, "generationConfig.speechConfig")
	}
	if constraints.SystemInstruction != "" {
		setup.SystemInstruction = &content{Parts: []part{{Text: constraints.SystemInstruction}}}
		mask = append(mask, "systemInstruction.parts")
	}
	mask = append(mask,
		"inputAudioTranscription",
		"outputAudioTranscription",
		"contextWindowCompression.slidingWindow",
	)

	payload, err := json.Marshal(tokenRequest{
		Uses:                 1,
		ExpireTime:           expireAt.Format(time.RFC3339),
		NewSessionExpireTime: newSessionExpireAt.Format(time.RFC3339),
		Setup:                &setup,
		FieldMask:            strings.Join(mask, ","),
	})
	if err != nil {
		return Token{}, fmt.Errorf("encode token request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return Token{}, fmt.Errorf("build token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-goog-api-key", c.apiKey)

	response, err := c.http.Do(request)
	if err != nil {
		return Token{}, fmt.Errorf("call auth_tokens: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Token{}, fmt.Errorf("read token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("auth_tokens returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var decoded struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Token{}, fmt.Errorf("decode token response: %w", err)
	}
	if decoded.Name == "" {
		return Token{}, errors.New("auth_tokens returned an empty token")
	}
	return Token{
		Name:                decoded.Name,
		ExpiresAt:           expireAt,
		NewSessionExpiresAt: newSessionExpireAt,
		Model:               c.model,
	}, nil
}

// qualifiedModel accepts both "gemini-3.8-live" and "models/gemini-3.8-live";
// the constraints field wants the fully qualified form.
func qualifiedModel(model string) string {
	if strings.HasPrefix(model, "models/") {
		return model
	}
	return "models/" + model
}
