package rpc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/cl-ment/sqlitrest/pkg/auth"
)

// RPCFunction représente une fonction SQL exposée comme RPC
type RPCFunction struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	Returns     string   `json:"returns"`
	Method      string   `json:"method"` // GET, POST
}

// RPCHandler gère les appels RPC aux fonctions SQL
type RPCHandler struct {
	db         *sql.DB
	jwtManager *auth.JWTManager
	functions  map[string]RPCFunction
}

// NewRPCHandler crée un nouveau handler RPC
func NewRPCHandler(db *sql.DB, jwtManager *auth.JWTManager) *RPCHandler {
	handler := &RPCHandler{
		db:         db,
		jwtManager: jwtManager,
		functions:  make(map[string]RPCFunction),
	}

	// Charger les fonctions par défaut
	handler.loadDefaultFunctions()

	return handler
}

// loadDefaultFunctions charge les fonctions RPC par défaut
func (h *RPCHandler) loadDefaultFunctions() {
	defaultFunctions := []RPCFunction{
		{
			Name:        "hello",
			Description: "Simple hello world function",
			Parameters:  []string{"name"},
			Returns:     "string",
			Method:      "POST",
		},
		{
			Name:        "count_users",
			Description: "Count total users",
			Parameters:  []string{},
			Returns:     "integer",
			Method:      "GET",
		},
		{
			Name:        "user_stats",
			Description: "Get user statistics",
			Parameters:  []string{"user_id"},
			Returns:     "object",
			Method:      "POST",
		},
	}

	for _, fn := range defaultFunctions {
		h.functions[fn.Name] = fn
	}
}

// HandleRPC gère les requêtes RPC
func (h *RPCHandler) HandleRPC(w http.ResponseWriter, req *http.Request) {
	// Extraire le nom de la fonction depuis l'URL
	functionName := extractFunctionName(req.URL.Path)
	if functionName == "" {
		http.Error(w, `{"error":"Function name required"}`, http.StatusBadRequest)
		return
	}

	// Vérifier si la fonction existe
	function, exists := h.functions[functionName]
	if !exists {
		http.Error(w, fmt.Sprintf(`{"error":"Function %s not found"}`, functionName), http.StatusNotFound)
		return
	}

	// Vérifier la méthode HTTP
	if req.Method != function.Method {
		http.Error(w, fmt.Sprintf(`{"error":"Method %s not allowed, use %s"}`, req.Method, function.Method), http.StatusMethodNotAllowed)
		return
	}

	// Authentifier la requête
	authCtx, err := h.jwtManager.AuthenticateRequest(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Authentication failed: %s"}`, err.Error()), http.StatusUnauthorized)
		return
	}

	// Parser les paramètres
	var params map[string]interface{}
	if req.Method == "POST" {
		if err := json.NewDecoder(req.Body).Decode(&params); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		// Pour GET, parser les query params
		params = make(map[string]interface{})
		for key, values := range req.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}
	}

	// Exécuter la fonction
	result, err := h.executeFunction(function, params, authCtx)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Function execution failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Retourner le résultat
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// executeFunction exécute une fonction RPC spécifique
func (h *RPCHandler) executeFunction(function RPCFunction, params map[string]interface{}, authCtx *auth.AuthContext) (interface{}, error) {
	switch function.Name {
	case "hello":
		return h.helloFunction(params)
	case "count_users":
		return h.countUsersFunction(params, authCtx)
	case "user_stats":
		return h.userStatsFunction(params, authCtx)
	default:
		return nil, fmt.Errorf("unknown function: %s", function.Name)
	}
}

// helloFunction - fonction de démonstration
func (h *RPCHandler) helloFunction(params map[string]interface{}) (interface{}, error) {
	name, ok := params["name"].(string)
	if !ok {
		name = "World"
	}

	return map[string]interface{}{
		"message":   fmt.Sprintf("Hello, %s!", name),
		"timestamp": "2025-11-29T23:00:00Z",
	}, nil
}

// countUsersFunction - compte les utilisateurs
func (h *RPCHandler) countUsersFunction(params map[string]interface{}, authCtx *auth.AuthContext) (interface{}, error) {
	var count int64

	// Appliquer les politiques de sécurité
	var query string
	var args []interface{}

	if authCtx.Authenticated && authCtx.Role == "admin" {
		query = "SELECT COUNT(*) FROM users"
		args = nil
	} else if authCtx.Authenticated {
		query = "SELECT COUNT(*) FROM users WHERE id = ?"
		args = []interface{}{authCtx.UserID}
	} else {
		query = "SELECT COUNT(*) FROM users WHERE is_public = TRUE"
		args = nil
	}

	err := h.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	return map[string]interface{}{
		"count": count,
		"role":  authCtx.Role,
	}, nil
}

// userStatsFunction - statistiques utilisateur
func (h *RPCHandler) userStatsFunction(params map[string]interface{}, authCtx *auth.AuthContext) (interface{}, error) {
	userID, ok := params["user_id"].(string)
	if !ok {
		if authCtx.Authenticated {
			userID = authCtx.UserID
		} else {
			return nil, fmt.Errorf("user_id parameter required")
		}
	}

	// Vérifier les permissions
	if authCtx.UserID != userID && authCtx.Role != "admin" {
		return nil, fmt.Errorf("access denied")
	}

	var stats struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		PostCount int64  `json:"post_count"`
	}

	// Récupérer les infos utilisateur
	err := h.db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", userID).Scan(&stats.ID, &stats.Name, &stats.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Compter les posts (si la table existe)
	var postCount int64
	h.db.QueryRow("SELECT COUNT(*) FROM posts WHERE author_id = ?", userID).Scan(&postCount)
	stats.PostCount = postCount

	return stats, nil
}

// ListFunctions retourne la liste des fonctions disponibles
func (h *RPCHandler) ListFunctions() []RPCFunction {
	var functions []RPCFunction
	for _, fn := range h.functions {
		functions = append(functions, fn)
	}
	return functions
}

// extractFunctionName extrait le nom de la fonction depuis l'URL
func extractFunctionName(path string) string {
	// Format: /rpc/function_name
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "rpc" {
		return parts[1]
	}
	return ""
}

// RegisterCustomFunction enregistre une fonction personnalisée
func (h *RPCHandler) RegisterCustomFunction(fn RPCFunction) {
	h.functions[fn.Name] = fn
	log.Printf("Registered custom RPC function: %s", fn.Name)
}
