package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cl-ment/sqlitrest/pkg/auth"
	"github.com/cl-ment/sqlitrest/pkg/config"
	"github.com/cl-ment/sqlitrest/pkg/db"
	"github.com/cl-ment/sqlitrest/pkg/engine"
	"github.com/cl-ment/sqlitrest/pkg/openapi"
	"github.com/cl-ment/sqlitrest/pkg/policies"
	"github.com/cl-ment/sqlitrest/pkg/rpc"
	"github.com/go-chi/chi/v5"
)

type Router struct {
	dbManager    *db.Manager
	config       *config.Config
	chi          *chi.Mux
	parser       *engine.QueryParser
	builder      *engine.SQLBuilder
	jwtManager   *auth.JWTManager
	policyEngine *policies.PolicyEngine
	openapiGen   *openapi.OpenAPIGenerator
	rpcHandler   *rpc.RPCHandler
	embedding    *engine.ResourceEmbedding
	schemaCache  *engine.SchemaCache
}

func New(dbManager *db.Manager, cfg *config.Config) *Router {
	// Créer le gestionnaire JWT
	jwtManager, err := auth.NewJWTManager(cfg.Auth.JWT)
	if err != nil {
		log.Printf("Failed to create JWT manager: %v", err)
	}

	// Créer le moteur de politiques
	var policyEngine *policies.PolicyEngine
	var openapiGen *openapi.OpenAPIGenerator
	var rpcHandler *rpc.RPCHandler
	var embedding *engine.ResourceEmbedding
	var schemaCache *engine.SchemaCache

	if db, err := dbManager.GetDB("main"); err == nil {
		policyEngine = policies.NewPolicyEngine(db.Writer)
		if err := policyEngine.LoadPolicies(); err != nil {
			log.Printf("Failed to load policies: %v", err)
		}

		openapiGen = openapi.NewOpenAPIGenerator(db.Writer)
		rpcHandler = rpc.NewRPCHandler(db.Writer, jwtManager)
		embedding = engine.NewResourceEmbedding(db.Writer)
		// Schema cache avec TTL de 5 minutes
		schemaCache = engine.NewSchemaCache(db.Writer, 5*time.Minute)
	}

	r := &Router{
		dbManager:    dbManager,
		config:       cfg,
		chi:          chi.NewRouter(),
		parser:       engine.NewQueryParser(),
		builder:      engine.NewSQLBuilder(),
		jwtManager:   jwtManager,
		policyEngine: policyEngine,
		openapiGen:   openapiGen,
		rpcHandler:   rpcHandler,
		embedding:    embedding,
		schemaCache:  schemaCache,
	}

	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	// Health check
	r.chi.Get("/health", r.handleHealth)

	// Debug endpoints
	r.chi.Route("/_debug", func(debugRouter chi.Router) {
		debugRouter.Get("/", r.handleDebugIndex)
		debugRouter.Get("/databases", r.handleDebugDatabases)
		debugRouter.Get("/schema", r.handleDebugSchema)
		debugRouter.Get("/policies", r.handleDebugPolicies)
		debugRouter.Get("/auth", r.handleDebugAuth)
	})

	// OpenAPI endpoint
	r.chi.Get("/", r.handleOpenAPI)
	r.chi.Get("/swagger.json", r.handleOpenAPIJSON)

	// RPC endpoints
	r.chi.Route("/rpc", func(rpcRouter chi.Router) {
		rpcRouter.Get("/*", r.rpcHandler.HandleRPC)
		rpcRouter.Post("/*", r.rpcHandler.HandleRPC)
		rpcRouter.Get("/", r.handleRPCList)
	})

	// Database routes
	databases := r.dbManager.ListDatabases()
	for dbName := range databases {
		r.chi.Route("/"+dbName, func(dbRouter chi.Router) {
			dbRouter.Get("/*", r.handleTableQuery(dbName))
			dbRouter.Post("/*", r.handleTableCreate(dbName))
			dbRouter.Patch("/*", r.handleTableUpdate(dbName))
			dbRouter.Delete("/*", r.handleTableDelete(dbName))
		})
	}
}

func (r *Router) Start(addr string) error {
	return http.ListenAndServe(addr, r.chi)
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"sqlitrest"}`))
}

func (r *Router) handleDebugIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"debug":"sqlitrest","version":"1.0.0"}`))
}

func (r *Router) handleDebugDatabases(w http.ResponseWriter, req *http.Request) {
	databases := r.dbManager.ListDatabases()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON response
	w.Write([]byte(`{"databases":[`))
	first := true
	for name := range databases {
		if !first {
			w.Write([]byte(","))
		}
		w.Write([]byte(`{"name":"` + name + `"}`))
		first = false
	}
	w.Write([]byte(`]}`))
}

func (r *Router) handleDebugSchema(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"schema":"TODO"}`))
}

func (r *Router) handleDebugPolicies(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.policyEngine == nil {
		w.Write([]byte(`{"policies":null,"message":"Policy engine not initialized"}`))
		return
	}

	// Pour l'instant, retourner un message simple
	w.Write([]byte(`{"policies":"loaded","message":"Policy engine is active"}`))
}

func (r *Router) handleDebugAuth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authCtx, err := r.jwtManager.AuthenticateRequest(req)
	if err != nil {
		w.Write([]byte(`{"authenticated":false,"error":"` + err.Error() + `"}`))
		return
	}

	response := map[string]interface{}{
		"authenticated": authCtx.Authenticated,
		"user_id":       authCtx.UserID,
		"role":          authCtx.Role,
		"tenant_id":     authCtx.TenantID,
		"permissions":   authCtx.Permissions,
	}

	json.NewEncoder(w).Encode(response)
}

func (r *Router) handleOpenAPI(w http.ResponseWriter, req *http.Request) {
	if r.openapiGen == nil {
		http.Error(w, `{"error":"OpenAPI generator not available"}`, http.StatusServiceUnavailable)
		return
	}

	doc, err := r.openapiGen.Generate()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to generate OpenAPI: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (r *Router) handleOpenAPIJSON(w http.ResponseWriter, req *http.Request) {
	r.handleOpenAPI(w, req)
}

func (r *Router) handleRPCList(w http.ResponseWriter, req *http.Request) {
	if r.rpcHandler == nil {
		http.Error(w, `{"error":"RPC handler not available"}`, http.StatusServiceUnavailable)
		return
	}

	functions := r.rpcHandler.ListFunctions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"functions": functions,
	})
}

func (r *Router) handleTableQuery(dbName string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Authentifier la requête
		authCtx, err := r.jwtManager.AuthenticateRequest(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Authentication failed: %s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Parser les paramètres de la requête
		params, err := r.parser.ParseQuery(req.URL.Path, req.URL.Query())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Vérifier les permissions d'accès à la table
		if !authCtx.CanAccessTable(params.Table) {
			http.Error(w, `{"error":"Access denied to table"}`, http.StatusForbidden)
			return
		}

		// Obtenir la base de données
		database, err := r.dbManager.GetDB(dbName)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database %s not found"}`, dbName), http.StatusNotFound)
			return
		}

		// Debug: afficher les paramètres parsés
		log.Printf("Parsed params: table=%s, filters=%v, auth=%s", params.Table, params.Filters, authCtx.Role)

		// Construire la requête SQL avec embedding support
		var query string
		var args []interface{}

		// Utiliser BuildSelectWithEmbedding si embedding est disponible
		if r.embedding != nil {
			query, args, err = r.builder.BuildSelectWithEmbedding(params, r.embedding)
		} else {
			query, args, err = r.builder.BuildSelect(params)
		}

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Appliquer les politiques de sécurité
		if r.policyEngine != nil {
			secureQuery, secureArgs, err := r.policyEngine.ApplyPolicies(query, params, authCtx)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"Policy application failed: %s"}`, err.Error()), http.StatusInternalServerError)
				return
			}
			query = secureQuery
			// Combiner les arguments originaux avec les arguments de sécurité
			args = append(args, secureArgs...)
			log.Printf("Applied policies, new query: %s, combined args: %v", query, args)
		}

		// Debug: afficher le SQL généré
		r.builder.DebugSQLBuilder(params)

		// Exécuter la requête
		executor := engine.NewExecutor(database.Writer)
		result, err := executor.ExecuteSelect(query, args)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Query execution failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Gérer les différents Media Types (PostgREST compatible)
		acceptHeader := req.Header.Get("Accept")
		contentType := r.negotiateContentType(acceptHeader)

		switch contentType {
		case "text/csv":
			r.writeCSVResponse(w, result)
		case "application/vnd.pgrst.object":
			r.writeObjectResponse(w, result)
		case "application/vnd.pgrst.plan":
			r.writePlanResponse(w, query, args)
		default:
			// JSON par défaut
			r.writeJSONResponse(w, result)
		}
	}
}

// negotiateContentType détermine le content type selon l'header Accept (PostgREST compatible)
func (r *Router) negotiateContentType(acceptHeader string) string {
	if acceptHeader == "" {
		return "application/json"
	}

	// Support des types PostgREST
	if strings.Contains(acceptHeader, "text/csv") {
		return "text/csv"
	}
	if strings.Contains(acceptHeader, "application/vnd.pgrst.object") {
		return "application/vnd.pgrst.object"
	}
	if strings.Contains(acceptHeader, "application/vnd.pgrst.plan") {
		return "application/vnd.pgrst.plan"
	}

	// JSON par défaut
	return "application/json"
}

// writeJSONResponse écrit une réponse JSON
func (r *Router) writeJSONResponse(w http.ResponseWriter, result *engine.QueryResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if len(result.Rows) == 0 {
		w.Write([]byte("[]"))
		return
	}

	jsonData, err := json.Marshal(result.Rows)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"JSON encoding failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// writeCSVResponse écrit une réponse CSV
func (r *Router) writeCSVResponse(w http.ResponseWriter, result *engine.QueryResult) {
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)

	if len(result.Rows) == 0 {
		w.Write([]byte(""))
		return
	}

	// Écrire les en-têtes CSV
	if len(result.Columns) > 0 {
		headers := strings.Join(result.Columns, ",")
		w.Write([]byte(headers + "\n"))
	}

	// Écrire les données
	for _, row := range result.Rows {
		var values []string
		for _, col := range result.Columns {
			if val, exists := row[col]; exists {
				values = append(values, fmt.Sprintf("%v", val))
			} else {
				values = append(values, "")
			}
		}
		w.Write([]byte(strings.Join(values, ",") + "\n"))
	}
}

// writeObjectResponse écrit une réponse single object (PostgREST compatible)
func (r *Router) writeObjectResponse(w http.ResponseWriter, result *engine.QueryResult) {
	w.Header().Set("Content-Type", "application/json")

	if len(result.Rows) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`[]`))
		return
	}

	if len(result.Rows) > 1 {
		w.WriteHeader(http.StatusMultipleChoices)
	}

	jsonData, err := json.Marshal(result.Rows[0])
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"JSON encoding failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// writePlanResponse écrit une réponse EXPLAIN plan (PostgREST compatible)
func (r *Router) writePlanResponse(w http.ResponseWriter, query string, args []interface{}) {
	w.Header().Set("Content-Type", "application/json")

	plan := map[string]interface{}{
		"plan": map[string]interface{}{
			"query": query,
			"args":  args,
		},
	}

	jsonData, err := json.Marshal(plan)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"JSON encoding failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

func (r *Router) handleTableCreate(dbName string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Parser le corps de la requête JSON
		var data map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Obtenir la base de données
		database, err := r.dbManager.GetDB(dbName)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database %s not found"}`, dbName), http.StatusNotFound)
			return
		}

		// Parser les paramètres pour obtenir le nom de la table
		params, err := r.parser.ParseQuery(req.URL.Path, req.URL.Query())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Construire la requête INSERT
		query, args, err := r.builder.BuildInsert(params.Table, data)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Exécuter la requête
		executor := engine.NewExecutor(database.Writer)
		result, err := executor.ExecuteCommand(query, args)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Insert failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Retourner le succès
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		rowsAffected := int64(0)
		if len(result.Rows) > 0 {
			if ra, exists := result.Rows[0]["rows_affected"]; exists {
				if raInt, ok := ra.(int64); ok {
					rowsAffected = raInt
				}
			}
		}

		response := map[string]interface{}{
			"rows_affected": rowsAffected,
			"table":         params.Table,
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (r *Router) handleTableUpdate(dbName string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Parser le corps de la requête JSON
		var data map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&data); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Invalid JSON: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Obtenir la base de données
		database, err := r.dbManager.GetDB(dbName)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database %s not found"}`, dbName), http.StatusNotFound)
			return
		}

		// Parser les paramètres pour obtenir le nom de la table et les filtres
		params, err := r.parser.ParseQuery(req.URL.Path, req.URL.Query())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Construire la requête UPDATE
		query, args, err := r.builder.BuildUpdate(params.Table, data, params.Filters)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Exécuter la requête
		executor := engine.NewExecutor(database.Writer)
		result, err := executor.ExecuteCommand(query, args)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Update failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Retourner le succès
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		rowsAffected := int64(0)
		if len(result.Rows) > 0 {
			if ra, exists := result.Rows[0]["rows_affected"]; exists {
				if raInt, ok := ra.(int64); ok {
					rowsAffected = raInt
				}
			}
		}

		response := map[string]interface{}{
			"rows_affected": rowsAffected,
			"table":         params.Table,
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (r *Router) handleTableDelete(dbName string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Obtenir la base de données
		database, err := r.dbManager.GetDB(dbName)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database %s not found"}`, dbName), http.StatusNotFound)
			return
		}

		// Parser les paramètres pour obtenir le nom de la table et les filtres
		params, err := r.parser.ParseQuery(req.URL.Path, req.URL.Query())
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Construire la requête DELETE
		query, args, err := r.builder.BuildDelete(params.Table, params.Filters)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		// Exécuter la requête
		executor := engine.NewExecutor(database.Writer)
		result, err := executor.ExecuteCommand(query, args)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Delete failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		// Retourner le succès
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		rowsAffected := int64(0)
		if len(result.Rows) > 0 {
			if ra, exists := result.Rows[0]["rows_affected"]; exists {
				if raInt, ok := ra.(int64); ok {
					rowsAffected = raInt
				}
			}
		}

		response := map[string]interface{}{
			"rows_affected": rowsAffected,
			"table":         params.Table,
		}
		json.NewEncoder(w).Encode(response)
	}
}
