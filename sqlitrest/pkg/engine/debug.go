package engine

import (
	"log"
)

// DebugSQLBuilder affiche le SQL généré pour debugging
func (b *SQLBuilder) DebugSQLBuilder(params *QueryParameters) {
	query, args, err := b.BuildSelect(params)
	if err != nil {
		log.Printf("Error building SQL: %v", err)
		return
	}

	log.Printf("Generated SQL: %s", query)
	log.Printf("Args: %v", args)
}
