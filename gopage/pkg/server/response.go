package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/horos/gopage/pkg/engine"
)

// ResponseType indicates the type of response to send
type ResponseType int

const (
	ResponseHTML ResponseType = iota
	ResponseJSON
	ResponseRedirect
	ResponseFragment
	ResponseRefresh
	ResponseError
)

// SQLResponse holds the response configuration from SQL results
type SQLResponse struct {
	Type        ResponseType
	Redirect    string
	Fragment    string
	Target      string
	Swap        string
	Trigger     string
	PushURL     string
	Message     string
	MessageType string // success, error, warning, info
	Data        interface{}
}

// ParseSQLResponse extracts response directives from SQL results
func ParseSQLResponse(rows []engine.Row) *SQLResponse {
	resp := &SQLResponse{
		Type: ResponseHTML,
	}

	for _, row := range rows {
		// Check for redirect directive
		if redirect, ok := row["redirect"].(string); ok && redirect != "" {
			resp.Type = ResponseRedirect
			resp.Redirect = redirect
		}

		// Check for JSON response
		if jsonData, ok := row["json"]; ok {
			resp.Type = ResponseJSON
			resp.Data = jsonData
		}

		// Check for fragment response (partial HTML)
		if fragment, ok := row["fragment"].(string); ok && fragment != "" {
			resp.Type = ResponseFragment
			resp.Fragment = fragment
		}

		// Check for refresh directive
		if _, ok := row["refresh"]; ok {
			resp.Type = ResponseRefresh
		}

		// HTMX response headers
		if target, ok := row["hx_target"].(string); ok {
			resp.Target = target
		}
		if swap, ok := row["hx_swap"].(string); ok {
			resp.Swap = swap
		}
		if trigger, ok := row["hx_trigger"].(string); ok {
			resp.Trigger = trigger
		}
		if pushURL, ok := row["hx_push_url"].(string); ok {
			resp.PushURL = pushURL
		}

		// Flash messages
		if msg, ok := row["message"].(string); ok {
			resp.Message = msg
		}
		if msgType, ok := row["message_type"].(string); ok {
			resp.MessageType = msgType
		}

		// Error response
		if errMsg, ok := row["error"].(string); ok {
			resp.Type = ResponseError
			resp.Message = errMsg
		}
	}

	return resp
}

// WriteHTMXHeaders sets HTMX-specific response headers
func WriteHTMXHeaders(w http.ResponseWriter, resp *SQLResponse) {
	if resp.Target != "" {
		w.Header().Set("HX-Retarget", resp.Target)
	}
	if resp.Swap != "" {
		w.Header().Set("HX-Reswap", resp.Swap)
	}
	if resp.Trigger != "" {
		w.Header().Set("HX-Trigger", resp.Trigger)
	}
	if resp.PushURL != "" {
		w.Header().Set("HX-Push-Url", resp.PushURL)
	}
}

// HandleSQLResponse processes the SQL response and sends appropriate HTTP response
func (s *Server) HandleSQLResponse(w http.ResponseWriter, r *http.Request, resp *SQLResponse) bool {
	isHTMX := r.Header.Get("HX-Request") == "true"

	switch resp.Type {
	case ResponseRedirect:
		if isHTMX {
			w.Header().Set("HX-Redirect", resp.Redirect)
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, resp.Redirect, http.StatusSeeOther)
		}
		return true

	case ResponseJSON:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp.Data)
		return true

	case ResponseRefresh:
		if isHTMX {
			w.Header().Set("HX-Refresh", "true")
			w.WriteHeader(http.StatusOK)
		} else {
			http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		}
		return true

	case ResponseFragment:
		WriteHTMXHeaders(w, resp)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(resp.Fragment))
		return true

	case ResponseError:
		WriteHTMXHeaders(w, resp)
		if isHTMX {
			// Return error as HTML fragment
			errorHTML := fmt.Sprintf(`<div class="alert alert-error" role="alert">%s</div>`, resp.Message)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(errorHTML))
		} else {
			http.Error(w, resp.Message, http.StatusUnprocessableEntity)
		}
		return true
	}

	// Add flash message header if present
	if resp.Message != "" {
		msgType := resp.MessageType
		if msgType == "" {
			msgType = "info"
		}
		// Use HX-Trigger for toast notifications
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showMessage": {"message": "%s", "type": "%s"}}`, resp.Message, msgType))
	}

	return false
}

// AlertFragment generates an HTML alert fragment
func AlertFragment(message, alertType string) string {
	icon := ""
	switch alertType {
	case "success":
		icon = "✓"
	case "error":
		icon = "✗"
	case "warning":
		icon = "⚠"
	default:
		icon = "ℹ"
	}

	return fmt.Sprintf(`<div class="alert alert-%s" role="alert" x-data x-init="setTimeout(() => $el.remove(), 5000)">
		<span class="alert-icon">%s</span>
		<span class="alert-message">%s</span>
		<button type="button" class="alert-close" @click="$el.parentElement.remove()">&times;</button>
	</div>`, alertType, icon, message)
}

// SuccessFragment generates a success response fragment
func SuccessFragment(message string) string {
	return AlertFragment(message, "success")
}

// ErrorFragment generates an error response fragment
func ErrorFragment(message string) string {
	return AlertFragment(message, "error")
}
