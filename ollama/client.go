package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	ollamaURL   = "http://localhost:11434/api/generate"
	ollamaModel = "ideaboard"
	timeout     = 30 * time.Second
)

// BrainstormResult holds the AI-generated ideas.
type BrainstormResult struct {
	Ideas       []string  `json:"ideas"`
	GeneratedAt time.Time `json:"generated_at"`
}

// Client wraps the local Ollama API.
type Client struct {
	http    *http.Client
	enabled bool
	baseURL string
	model   string
}

// NewClient creates an Ollama client.
// The apiKey parameter is kept for backward compatibility but is unused.
func NewClient(apiKey string) *Client {
	c := &Client{
		http:    &http.Client{Timeout: timeout},
		baseURL: ollamaURL,
		model:   ollamaModel,
	}

	if c.ping() {
		c.enabled = true
		log.Printf("[ollama] connected — using local model %q", ollamaModel)
	} else {
		c.enabled = false
		log.Println("[ollama] server not reachable at", ollamaURL, "— brainstorm disabled")
		log.Println("[ollama] start Ollama with: ollama serve")
	}

	return c
}

// Enabled reports whether the Ollama client is active.
func (c *Client) Enabled() bool { return c.enabled }

// Synthesize takes the existing board ideas and generates new related ones.
func (c *Client) Synthesize(ctx context.Context, existingIdeas []string) (*BrainstormResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama not running — start with: ollama serve")
	}
	if len(existingIdeas) == 0 {
		return &BrainstormResult{Ideas: []string{}, GeneratedAt: time.Now().UTC()}, nil
	}
	if len(existingIdeas) > 100 {
		existingIdeas = existingIdeas[:100]
	}

	ideas, err := c.callOllama(ctx, buildPrompt(existingIdeas))
	if err != nil {
		return nil, fmt.Errorf("ollama synthesize: %w", err)
	}

	return &BrainstormResult{
		Ideas:       ideas,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// buildPrompt creates the instruction prompt matching the fine-tuning format.
func buildPrompt(ideas []string) string {
	return fmt.Sprintf(
		"### Instruction:\nGiven these ideas on a brainstorming board: %s — suggest 3 new related ideas.\n\n### Response:",
		strings.Join(ideas, ", "),
	)
}

// --- Ollama API wire types ---

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *Client) callOllama(ctx context.Context, prompt string) ([]string, error) {
	body, _ := json.Marshal(ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var or ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	ideas := parseIdeas(or.Response)
	if len(ideas) == 0 {
		return nil, fmt.Errorf("could not parse ideas from response: %q", or.Response)
	}

	return ideas, nil
}

// parseIdeas extracts numbered ideas from the model's text output.
func parseIdeas(raw string) []string {
	raw = strings.TrimSpace(raw)

	re := regexp.MustCompile(`(?m)^[\d]+[.)]\s*(.+)$|^[-•]\s*(.+)$`)
	matches := re.FindAllStringSubmatch(raw, -1)

	ideas := make([]string, 0, 3)
	for _, m := range matches {
		idea := strings.TrimSpace(m[1])
		if idea == "" {
			idea = strings.TrimSpace(m[2])
		}
		if idea != "" {
			ideas = append(ideas, idea)
		}
		if len(ideas) == 3 {
			break
		}
	}

	// Fallback: split by newlines
	if len(ideas) == 0 {
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				ideas = append(ideas, line)
			}
			if len(ideas) == 3 {
				break
			}
		}
	}

	return ideas
}

// ping checks if Ollama is running.
func (c *Client) ping() bool {
	resp, err := http.Get("http://localhost:11434")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
