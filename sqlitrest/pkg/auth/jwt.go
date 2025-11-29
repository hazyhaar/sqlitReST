package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims représente les claims JWT
type Claims struct {
	UserID      string   `json:"user_id"`
	Role        string   `json:"role"`
	TenantID    string   `json:"tenant_id,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

// JWTConfig contient la configuration JWT
type JWTConfig struct {
	Enabled   bool     `toml:"enabled"`
	Algorithm string   `toml:"algorithm"`
	Secret    string   `toml:"secret"`
	Issuer    string   `toml:"issuer"`
	Audience  []string `toml:"audience"`
}

// AuthContext contient le contexte d'authentification
type AuthContext struct {
	Authenticated bool
	UserID        string
	Role          string
	TenantID      string
	Permissions   []string
	Token         string
}

// JWTManager gère les tokens JWT
type JWTManager struct {
	config JWTConfig
	secret []byte
}

// NewJWTManager crée un nouveau gestionnaire JWT
func NewJWTManager(config JWTConfig) (*JWTManager, error) {
	if !config.Enabled {
		return &JWTManager{config: config}, nil
	}

	if config.Secret == "" {
		return nil, fmt.Errorf("JWT secret is required when JWT is enabled")
	}

	return &JWTManager{
		config: config,
		secret: []byte(config.Secret),
	}, nil
}

// GenerateToken génère un token JWT
func (j *JWTManager) GenerateToken(userID, role, tenantID string, permissions []string) (string, error) {
	if !j.config.Enabled {
		return "", fmt.Errorf("JWT is disabled")
	}

	now := time.Now()
	claims := Claims{
		UserID:      userID,
		Role:        role,
		TenantID:    tenantID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   userID,
			Audience:  j.config.Audience,
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)), // 24h
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(j.config.Algorithm), claims)
	return token.SignedString(j.secret)
}

// ValidateToken valide un token JWT et retourne les claims
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	if !j.config.Enabled {
		return &Claims{}, nil
	}

	// Enlever le préfixe "Bearer "
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = tokenString[7:]
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Valider l'algorithme
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GetSigningMethod retourne la méthode de signature
func (j *JWTManager) GetSigningMethod(algorithm string) jwt.SigningMethod {
	switch algorithm {
	case "HS256":
		return jwt.SigningMethodHS256
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}

// ExtractTokenFromRequest extrait le token depuis la requête HTTP
func ExtractTokenFromRequest(req *http.Request) string {
	// 1. Header Authorization
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		return authHeader
	}

	// 2. Query parameter ?token=
	token := req.URL.Query().Get("token")
	if token != "" {
		return fmt.Sprintf("Bearer %s", token)
	}

	// 3. Cookie
	cookie, err := req.Cookie("jwt_token")
	if err == nil && cookie.Value != "" {
		return fmt.Sprintf("Bearer %s", cookie.Value)
	}

	return ""
}

// AuthenticateRequest authentifie une requête HTTP
func (j *JWTManager) AuthenticateRequest(req *http.Request) (*AuthContext, error) {
	if !j.config.Enabled {
		return &AuthContext{Authenticated: false}, nil
	}

	tokenString := ExtractTokenFromRequest(req)
	if tokenString == "" {
		return &AuthContext{Authenticated: false}, nil
	}

	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return &AuthContext{Authenticated: false}, err
	}

	return &AuthContext{
		Authenticated: true,
		UserID:        claims.UserID,
		Role:          claims.Role,
		TenantID:      claims.TenantID,
		Permissions:   claims.Permissions,
		Token:         tokenString,
	}, nil
}

// HasPermission vérifie si l'utilisateur a une permission spécifique
func (a *AuthContext) HasPermission(permission string) bool {
	if !a.Authenticated {
		return false
	}

	for _, p := range a.Permissions {
		if p == permission {
			return true
		}
	}

	// Les admins ont toutes les permissions
	return a.Role == "admin"
}

// IsOwner vérifie si l'utilisateur est le propriétaire de la ressource
func (a *AuthContext) IsOwner(resourceID string) bool {
	if !a.Authenticated {
		return false
	}

	return a.UserID == resourceID || a.Role == "admin"
}

// CanAccessTable vérifie si l'utilisateur peut accéder à une table
func (a *AuthContext) CanAccessTable(tableName string) bool {
	if !a.Authenticated {
		// Tables publiques accessibles sans auth
		publicTables := []string{"public_posts", "products", "categories"}
		for _, table := range publicTables {
			if table == tableName {
				return true
			}
		}
		return false
	}

	// Les admins peuvent tout accéder
	if a.Role == "admin" {
		return true
	}

	// Vérifier les permissions spécifiques
	permission := fmt.Sprintf("read:%s", tableName)
	return a.HasPermission(permission)
}
