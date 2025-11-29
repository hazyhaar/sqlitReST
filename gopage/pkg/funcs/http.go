package funcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zombiezen.com/go/sqlite"
)

// RegisterHTTPFunctions registers HTTP-related SQL functions
func (r *Registry) RegisterHTTPFunctions() {
	r.Register(&FuncDef{
		Name:          "http_get",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Makes an HTTP GET request and returns the response body",
		ScalarFunc:    r.funcHTTPGet,
	})

	r.Register(&FuncDef{
		Name:          "http_get_json",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Makes an HTTP GET request and returns parsed JSON",
		ScalarFunc:    r.funcHTTPGetJSON,
	})

	r.Register(&FuncDef{
		Name:          "http_post",
		NumArgs:       3, // url, content_type, body
		Deterministic: false,
		Description:   "Makes an HTTP POST request",
		ScalarFunc:    r.funcHTTPPost,
	})

	r.Register(&FuncDef{
		Name:          "http_post_json",
		NumArgs:       2, // url, json_body
		Deterministic: false,
		Description:   "Makes an HTTP POST request with JSON body",
		ScalarFunc:    r.funcHTTPPostJSON,
	})

	r.Register(&FuncDef{
		Name:          "http_post_form",
		NumArgs:       2, // url, form_data (key=value&key2=value2)
		Deterministic: false,
		Description:   "Makes an HTTP POST request with form data",
		ScalarFunc:    r.funcHTTPPostForm,
	})

	r.Register(&FuncDef{
		Name:          "http_head",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Makes an HTTP HEAD request and returns headers as JSON",
		ScalarFunc:    r.funcHTTPHead,
	})

	r.Register(&FuncDef{
		Name:          "webhook",
		NumArgs:       2, // url, json_payload
		Deterministic: false,
		Description:   "Sends a webhook notification (fire and forget)",
		ScalarFunc:    r.funcWebhook,
	})
}

// HTTPResponse represents an HTTP response
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// makeHTTPRequest makes an HTTP request
func (r *Registry) makeHTTPRequest(ctx context.Context, method, urlStr string, body io.Reader, contentType string) (*HTTPResponse, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("only http and https schemes are allowed")
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	req.Header.Set("User-Agent", "GoPage/0.1")

	client := &http.Client{
		Timeout: time.Duration(r.HTTPTimeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return &HTTPResponse{
			StatusCode: 0,
			Error:      err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	// Read body (limited to 10MB)
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return &HTTPResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("failed to read body: %v", err),
		}, nil
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(bodyBytes),
	}, nil
}

// funcHTTPGet makes a GET request
func (r *Registry) funcHTTPGet(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	urlStr := args[0].Text()
	if urlStr == "" {
		return nil, nil
	}

	resp, err := r.makeHTTPRequest(ctx, "GET", urlStr, nil, "")
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	return resp.Body, nil
}

// funcHTTPGetJSON makes a GET request and returns JSON
func (r *Registry) funcHTTPGetJSON(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	urlStr := args[0].Text()
	if urlStr == "" {
		return nil, nil
	}

	resp, err := r.makeHTTPRequest(ctx, "GET", urlStr, nil, "")
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	// Validate it's JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(resp.Body), &js); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}

	return resp.Body, nil
}

// funcHTTPPost makes a POST request
func (r *Registry) funcHTTPPost(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 3 {
		return nil, nil
	}

	urlStr := args[0].Text()
	contentType := args[1].Text()
	body := args[2].Text()

	if urlStr == "" {
		return nil, nil
	}

	resp, err := r.makeHTTPRequest(ctx, "POST", urlStr, strings.NewReader(body), contentType)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	return resp.Body, nil
}

// funcHTTPPostJSON makes a POST request with JSON
func (r *Registry) funcHTTPPostJSON(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	urlStr := args[0].Text()
	jsonBody := args[1].Text()

	if urlStr == "" {
		return nil, nil
	}

	// Validate JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(jsonBody), &js); err != nil {
		return nil, fmt.Errorf("body is not valid JSON: %w", err)
	}

	resp, err := r.makeHTTPRequest(ctx, "POST", urlStr, strings.NewReader(jsonBody), "application/json")
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	return resp.Body, nil
}

// funcHTTPPostForm makes a POST request with form data
func (r *Registry) funcHTTPPostForm(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	urlStr := args[0].Text()
	formData := args[1].Text()

	if urlStr == "" {
		return nil, nil
	}

	resp, err := r.makeHTTPRequest(ctx, "POST", urlStr, strings.NewReader(formData), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	return resp.Body, nil
}

// funcHTTPHead makes a HEAD request
func (r *Registry) funcHTTPHead(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 1 {
		return nil, nil
	}

	urlStr := args[0].Text()
	if urlStr == "" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "GoPage/0.1")

	client := &http.Client{
		Timeout: time.Duration(r.HTTPTimeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Build headers JSON
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	headers["Status-Code"] = fmt.Sprintf("%d", resp.StatusCode)

	jsonBytes, err := json.Marshal(headers)
	if err != nil {
		return nil, err
	}

	return string(jsonBytes), nil
}

// funcWebhook sends a webhook (fire and forget)
func (r *Registry) funcWebhook(ctx context.Context, args []sqlite.Value) (interface{}, error) {
	if len(args) < 2 {
		return nil, nil
	}

	urlStr := args[0].Text()
	payload := args[1].Text()

	if urlStr == "" {
		return false, nil
	}

	// Fire and forget - use goroutine with timeout
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(payload))
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "GoPage-Webhook/0.1")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()

	return true, nil
}
