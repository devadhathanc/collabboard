package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Models tried in order — first one that succeeds wins.
var modelURLs = []string{
	"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
	"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent",
}

const timeout = 20 * time.Second

// BrainstormResult holds the AI-generated ideas.
type BrainstormResult struct {
	Ideas       []string  `json:"ideas"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Client wraps the Gemini API.
type Client struct {
	apiKey  string
	http    *http.Client
	enabled bool
}

// NewClient creates a Gemini client. If apiKey is empty, the client is disabled.
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		log.Println("[gemini] GEMINI_API_KEY not set — brainstorm disabled")
	}
	return &Client{
		apiKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
		enabled: apiKey != "",
	}
}

// Enabled reports whether the Gemini client is active.
func (c *Client) Enabled() bool { return c.enabled }

// Synthesize takes the existing board ideas and generates new related ones.
func (c *Client) Synthesize(ctx context.Context, existingIdeas []string) (*BrainstormResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("gemini not configured")
	}
	if len(existingIdeas) == 0 {
		return &BrainstormResult{Ideas: []string{}, GeneratedAt: time.Now().UTC()}, nil
	}
	if len(existingIdeas) > 100 {
		log.Printf("[gemini] truncating %d ideas to 100", len(existingIdeas))
		existingIdeas = existingIdeas[:100]
	}

	for _, url := range modelURLs {
		result, err := c.callURL(ctx, url, buildPrompt(existingIdeas, false))
		if err == nil {
			result.GeneratedAt = time.Now().UTC()
			return result, nil
		}
		log.Printf("[gemini] %s failed: %v — trying next", url, err)

		result, err = c.callURL(ctx, url, buildPrompt(existingIdeas, true))
		if err == nil {
			result.GeneratedAt = time.Now().UTC()
			return result, nil
		}
		log.Printf("[gemini] %s strict retry failed: %v", url, err)
	}

	return nil, fmt.Errorf("all models exhausted")
}

func buildPrompt(ideas []string, strict bool) string {
	ideaList := strings.Join(ideas, "\n- ")
	base := fmt.Sprintf(`You are a sharp brainstorming assistant.
Given the following ideas, generate exactly 3 new, highly relevant ideas that are directly connected to the themes present.

Rules:
- Exactly 3 ideas, no more, no less.
- Each idea must be 1 to 3 words only (short noun phrases like "bike routes", "Ladakh trip", "youth clubs").
- Do NOT repeat or rephrase existing ideas.
- Be specific and directly related — not generic.

Existing ideas:
- %s

Return ONLY a JSON array of exactly 3 short strings. Example: ["bike routes", "Ladakh trip", "youth clubs"]
No markdown, no explanation — just the raw JSON array.`, ideaList)

	if strict {
		return base + "\n\nCRITICAL: Output ONLY the JSON array [ ... ]. Exactly 3 items. No other text."
	}
	return base
}

// --- Gemini API wire types ---

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (c *Client) callURL(ctx context.Context, apiURL string, prompt string) (*BrainstormResult, error) {
	body, _ := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	})

	req, err := http.NewRequestWithContext(ctx, "POST",
		apiURL+"?key="+c.apiKey,
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(b))
	}

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	raw := strings.TrimSpace(gr.Candidates[0].Content.Parts[0].Text)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var ideas []string
	if err := json.Unmarshal([]byte(raw), &ideas); err != nil {
		return nil, fmt.Errorf("parse JSON array: %w — raw: %s", err, raw)
	}
	return &BrainstormResult{Ideas: ideas}, nil
}
