package funcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"zombiezen.com/go/sqlite"
)

// LLMConfig holds configuration for LLM functions
type LLMConfig struct {
	Endpoint string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

// RegisterLLMFunctions registers LLM-related SQL functions
func (r *Registry) RegisterLLMFunctions(cfg LLMConfig) {
	r.LLMEndpoint = cfg.Endpoint
	r.LLMAPIKey = cfg.APIKey
	r.LLMModel = cfg.Model

	if cfg.Endpoint == "" {
		// Default to Cerebras
		r.LLMEndpoint = "https://api.cerebras.ai/v1/chat/completions"
	}

	if cfg.Model == "" {
		r.LLMModel = "llama-3.3-70b"
	}

	r.Register(&FuncDef{
		Name:          "llm_ask",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Asks an LLM a question and returns the response",
		ScalarFunc:    r.funcLLMAsk,
	})

	r.Register(&FuncDef{
		Name:          "llm_ask_with_system",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Asks an LLM with a system prompt",
		ScalarFunc:    r.funcLLMAskWithSystem,
	})

	r.Register(&FuncDef{
		Name:          "llm_summarize",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Summarizes text using LLM",
		ScalarFunc:    r.funcLLMSummarize,
	})

	r.Register(&FuncDef{
		Name:          "llm_translate",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Translates text to target language",
		ScalarFunc:    r.funcLLMTranslate,
	})

	r.Register(&FuncDef{
		Name:          "llm_extract_json",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Extracts structured JSON from text based on schema",
		ScalarFunc:    r.funcLLMExtractJSON,
	})

	r.Register(&FuncDef{
		Name:          "llm_classify",
		NumArgs:       2,
		Deterministic: false,
		Description:   "Classifies text into categories",
		ScalarFunc:    r.funcLLMClassify,
	})

	r.Register(&FuncDef{
		Name:          "llm_sentiment",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Analyzes sentiment of text (positive/negative/neutral)",
		ScalarFunc:    r.funcLLMSentiment,
	})
}

// ChatCompletionRequest represents an OpenAI-compatible chat request
type ChatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents an OpenAI-compatible chat response
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// callLLM makes a request to the LLM API
func (r *Registry) callLLM(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if r.LLMAPIKey == "" {
		return "", fmt.Errorf("LLM API key not configured")
	}

	messages := []ChatMessage{
		{Role: "user", Content: userPrompt},
	}

	if systemPrompt != "" {
		messages = []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		}
	}

	reqBody := ChatCompletionRequest{
		Model:       r.LLMModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.7,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.LLMEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.LLMAPIKey)

	client := &http.Client{
		Timeout: time.Duration(r.HTTPTimeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("LLM error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// funcLLMAsk asks an LLM a question
func (r *Registry) funcLLMAsk(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	prompt := args[0].Text()
	if prompt == "" {
		return nil, nil
	}

	return r.callLLM(ctx, "", prompt, 1000)
}

// funcLLMAskWithSystem asks with a system prompt
func (r *Registry) funcLLMAskWithSystem(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	systemPrompt := args[0].Text()
	userPrompt := args[1].Text()

	if userPrompt == "" {
		return nil, nil
	}

	return r.callLLM(ctx, systemPrompt, userPrompt, 1000)
}

// funcLLMSummarize summarizes text
func (r *Registry) funcLLMSummarize(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	text := args[0].Text()
	if text == "" {
		return nil, nil
	}

	systemPrompt := "Tu es un assistant qui résume les textes de manière concise et claire. Réponds uniquement avec le résumé, sans introduction."
	userPrompt := fmt.Sprintf("Résume ce texte en 2-3 phrases:\n\n%s", text)

	return r.callLLM(ctx, systemPrompt, userPrompt, 500)
}

// funcLLMTranslate translates text
func (r *Registry) funcLLMTranslate(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	text := args[0].Text()
	targetLang := args[1].Text()

	if text == "" {
		return nil, nil
	}

	systemPrompt := "Tu es un traducteur professionnel. Traduis le texte fourni dans la langue demandée. Réponds uniquement avec la traduction, sans explication."
	userPrompt := fmt.Sprintf("Traduis en %s:\n\n%s", targetLang, text)

	return r.callLLM(ctx, systemPrompt, userPrompt, 2000)
}

// funcLLMExtractJSON extracts structured data from text
func (r *Registry) funcLLMExtractJSON(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	text := args[0].Text()
	schema := args[1].Text()

	if text == "" {
		return nil, nil
	}

	systemPrompt := "Tu es un extracteur de données structurées. Extrais les informations du texte selon le schéma JSON fourni. Réponds UNIQUEMENT avec du JSON valide, sans aucun texte autour."
	userPrompt := fmt.Sprintf("Schéma attendu:\n%s\n\nTexte à analyser:\n%s", schema, text)

	return r.callLLM(ctx, systemPrompt, userPrompt, 1000)
}

// funcLLMClassify classifies text into categories
func (r *Registry) funcLLMClassify(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	text := args[0].Text()
	categories := args[1].Text() // Comma-separated list

	if text == "" {
		return nil, nil
	}

	systemPrompt := "Tu es un classificateur de texte. Réponds UNIQUEMENT avec le nom de la catégorie la plus appropriée, sans explication."
	userPrompt := fmt.Sprintf("Catégories possibles: %s\n\nTexte à classifier:\n%s\n\nCatégorie:", categories, text)

	return r.callLLM(ctx, systemPrompt, userPrompt, 50)
}

// funcLLMSentiment analyzes sentiment
func (r *Registry) funcLLMSentiment(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	text := args[0].Text()
	if text == "" {
		return nil, nil
	}

	systemPrompt := "Tu es un analyseur de sentiment. Analyse le sentiment du texte et réponds UNIQUEMENT avec un seul mot: 'positif', 'négatif', ou 'neutre'."
	userPrompt := fmt.Sprintf("Texte:\n%s", text)

	return r.callLLM(ctx, systemPrompt, userPrompt, 20)
}
