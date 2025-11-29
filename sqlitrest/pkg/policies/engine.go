package policies

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/cl-ment/sqlitrest/pkg/auth"
	"github.com/cl-ment/sqlitrest/pkg/engine"
)

// Policy représente une politique de sécurité
type Policy struct {
	Name        string `json:"name"`
	Table       string `json:"table"`
	Action      string `json:"action"` // SELECT, INSERT, UPDATE, DELETE
	Expression  string `json:"expression"`
	Description string `json:"description"`
}

// PolicyEngine applique les politiques de sécurité (Row Level Security)
type PolicyEngine struct {
	db       *sql.DB
	policies map[string][]Policy // table -> policies
}

// NewPolicyEngine crée un nouveau moteur de politiques
func NewPolicyEngine(db *sql.DB) *PolicyEngine {
	return &PolicyEngine{
		db:       db,
		policies: make(map[string][]Policy),
	}
}

// LoadPolicies charge les politiques depuis la base de données
func (e *PolicyEngine) LoadPolicies() error {
	// Créer la table des politiques si elle n'existe pas
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS _policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		table_name TEXT NOT NULL,
		action TEXT NOT NULL CHECK (action IN ('SELECT', 'INSERT', 'UPDATE', 'DELETE')),
		expression TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := e.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create policies table: %w", err)
	}

	// Charger les politiques actives
	rows, err := e.db.Query("SELECT name, table_name, action, expression, description FROM _policies WHERE enabled = TRUE")
	if err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}
	defer rows.Close()

	e.policies = make(map[string][]Policy)
	for rows.Next() {
		var policy Policy
		err := rows.Scan(&policy.Name, &policy.Table, &policy.Action, &policy.Expression, &policy.Description)
		if err != nil {
			return fmt.Errorf("failed to scan policy: %w", err)
		}

		e.policies[policy.Table] = append(e.policies[policy.Table], policy)
	}

	return nil
}

// ApplyPolicies applique les politiques de sécurité à une requête SQL
func (e *PolicyEngine) ApplyPolicies(query string, params *engine.QueryParameters, authCtx *auth.AuthContext) (string, []interface{}, error) {
	table := params.Table
	action := e.determineActionFromQuery(query)

	// Récupérer les politiques pour cette table et action
	tablePolicies := e.getPoliciesForTable(table, action)
	if len(tablePolicies) == 0 {
		return query, nil, nil // Pas de politiques à appliquer
	}

	// Construire les conditions de sécurité
	var securityConditions []string
	var securityArgs []interface{}

	for _, policy := range tablePolicies {
		condition, args, err := e.evaluatePolicyExpression(policy.Expression, authCtx)
		if err != nil {
			return "", nil, fmt.Errorf("failed to evaluate policy %s: %w", policy.Name, err)
		}

		if condition != "" {
			securityConditions = append(securityConditions, condition)
			securityArgs = append(securityArgs, args...)
		}
	}

	if len(securityConditions) == 0 {
		return query, nil, nil
	}

	// Injecter les conditions de sécurité dans la requête
	return e.injectSecurityConditions(query, strings.Join(securityConditions, " OR "), securityArgs)
}

// getPoliciesForTable retourne les politiques pour une table et action spécifiques
func (e *PolicyEngine) getPoliciesForTable(table, action string) []Policy {
	policies, exists := e.policies[table]
	if !exists {
		return nil
	}

	var filtered []Policy
	for _, policy := range policies {
		if policy.Action == action || policy.Action == "ALL" {
			filtered = append(filtered, policy)
		}
	}

	return filtered
}

// determineActionFromQuery détermine l'action SQL depuis la requête
func (e *PolicyEngine) determineActionFromQuery(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))

	if strings.HasPrefix(query, "SELECT") {
		return "SELECT"
	} else if strings.HasPrefix(query, "INSERT") {
		return "INSERT"
	} else if strings.HasPrefix(query, "UPDATE") {
		return "UPDATE"
	} else if strings.HasPrefix(query, "DELETE") {
		return "DELETE"
	}

	return "UNKNOWN"
}

// evaluatePolicyExpression évalue une expression de politique avec le contexte d'authentification
func (e *PolicyEngine) evaluatePolicyExpression(expression string, authCtx *auth.AuthContext) (string, []interface{}, error) {
	// Évaluer les conditions simples d'abord
	if authCtx.Authenticated && authCtx.Role == "admin" {
		return "1=1", []interface{}{}, nil // Les admins contournent toutes les politiques
	}

	var args []interface{}

	// Substitution directe des fonctions contextuelles par les valeurs réelles
	if strings.Contains(expression, "current_user_id()") {
		if authCtx.Authenticated {
			expression = strings.ReplaceAll(expression, "current_user_id()", authCtx.UserID)
		} else {
			expression = strings.ReplaceAll(expression, "current_user_id()", "NULL")
		}
	}

	if strings.Contains(expression, "current_role()") {
		if authCtx.Authenticated {
			expression = strings.ReplaceAll(expression, "current_role()", fmt.Sprintf("'%s'", authCtx.Role))
		} else {
			expression = strings.ReplaceAll(expression, "current_role()", "'anonymous'")
		}
	}

	if strings.Contains(expression, "current_tenant_id()") {
		if authCtx.Authenticated && authCtx.TenantID != "" {
			expression = strings.ReplaceAll(expression, "current_tenant_id()", fmt.Sprintf("'%s'", authCtx.TenantID))
		} else {
			expression = strings.ReplaceAll(expression, "current_tenant_id()", "NULL")
		}
	}

	return expression, args, nil
}

// injectSecurityConditions injecte les conditions de sécurité dans une requête SQL
func (e *PolicyEngine) injectSecurityConditions(query, securityCondition string, securityArgs []interface{}) (string, []interface{}, error) {
	// Pour les requêtes SELECT, injecter dans WHERE
	if strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		if strings.Contains(strings.ToUpper(query), " WHERE ") {
			// Il y a déjà une clause WHERE, ajouter avec AND
			query = fmt.Sprintf("%s AND (%s)", query, securityCondition)
		} else {
			// Pas de clause WHERE, en ajouter une
			// Trouver où insérer (après FROM mais avant ORDER/LIMIT/OFFSET)
			query = e.insertWhereAfterFrom(query, securityCondition)
		}
		return query, securityArgs, nil
	}

	// Pour UPDATE et DELETE, injecter dans WHERE
	if strings.HasPrefix(strings.ToUpper(query), "UPDATE") || strings.HasPrefix(strings.ToUpper(query), "DELETE") {
		if strings.Contains(strings.ToUpper(query), " WHERE ") {
			query = fmt.Sprintf("%s AND (%s)", query, securityCondition)
		} else {
			// Pour UPDATE/DELETE sans WHERE, ajouter WHERE pour sécurité
			query = fmt.Sprintf("%s WHERE %s", query, securityCondition)
		}
		return query, securityArgs, nil
	}

	// Pour INSERT, pas de modification (les politiques INSERT gèrent les colonnes)
	return query, securityArgs, nil
}

// insertWhereAfterFrom insère une clause WHERE après FROM
func (e *PolicyEngine) insertWhereAfterFrom(query, condition string) string {
	// Simplifié : trouver la fin de FROM et insérer WHERE
	upperQuery := strings.ToUpper(query)
	fromIndex := strings.Index(upperQuery, " FROM ")
	if fromIndex == -1 {
		return query
	}

	// Trouver la fin de la clause FROM (prochain WHERE, ORDER, LIMIT, etc.)
	afterFrom := query[fromIndex+6:] // +6 pour " FROM "

	// Chercher les clauses qui suivent
	clauses := []string{" WHERE ", " ORDER BY ", " GROUP BY ", " HAVING ", " LIMIT ", " OFFSET "}
	insertPos := len(afterFrom)

	for _, clause := range clauses {
		if pos := strings.Index(strings.ToUpper(afterFrom), clause); pos != -1 && pos < insertPos {
			insertPos = pos
		}
	}

	if insertPos == len(afterFrom) {
		// Pas d'autres clauses, ajouter à la fin
		return query + " WHERE " + condition
	}

	// Insérer avant la clause trouvée
	beforeClause := afterFrom[:insertPos]
	afterClause := afterFrom[insertPos:]

	return query[:fromIndex+6] + beforeClause + " WHERE " + condition + afterClause
}

// CreateDefaultPolicies crée les politiques par défaut pour la démo
func (e *PolicyEngine) CreateDefaultPolicies() error {
	defaultPolicies := []Policy{
		{
			Name:        "users_select_own",
			Table:       "users",
			Action:      "SELECT",
			Expression:  "id = current_user_id() OR current_role() = 'admin'",
			Description: "Users can see their own profile, admins can see all",
		},
		{
			Name:        "users_update_own",
			Table:       "users",
			Action:      "UPDATE",
			Expression:  "id = current_user_id() OR current_role() = 'admin'",
			Description: "Users can update their own profile, admins can update all",
		},
		{
			Name:        "users_delete_admin_only",
			Table:       "users",
			Action:      "DELETE",
			Expression:  "current_role() = 'admin'",
			Description: "Only admins can delete users",
		},
		{
			Name:        "posts_select_public_or_own",
			Table:       "posts",
			Action:      "SELECT",
			Expression:  "is_public = TRUE OR author_id = current_user_id() OR current_role() = 'admin'",
			Description: "Public posts visible to all, own posts to authors, all to admins",
		},
		{
			Name:        "posts_insert_authenticated",
			Table:       "posts",
			Action:      "INSERT",
			Expression:  "current_user_id() IS NOT NULL",
			Description: "Only authenticated users can create posts",
		},
	}

	for _, policy := range defaultPolicies {
		_, err := e.db.Exec(`
			INSERT OR IGNORE INTO _policies (name, table_name, action, expression, description)
			VALUES (?, ?, ?, ?, ?)
		`, policy.Name, policy.Table, policy.Action, policy.Expression, policy.Description)

		if err != nil {
			return fmt.Errorf("failed to create policy %s: %w", policy.Name, err)
		}
	}

	return e.LoadPolicies()
}

// RegisterUDFs enregistre les fonctions utilisateur pour les politiques
func (e *PolicyEngine) RegisterUDFs(db *sql.DB, authCtx *auth.AuthContext) error {
	// Enregistrer les fonctions contextuelles
	// Note: SQLite supporte les fonctions personnalisées via Go
	// Pour l'instant, nous utilisons le remplacement de texte dans evaluatePolicyExpression

	return nil
}
