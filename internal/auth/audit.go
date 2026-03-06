package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// LogAction asynchronously inserts an audit record into the database.
// It is designed to be fire-and-forget so it never blocks the critical HTTP path.
func (s *Service) LogAction(keyID *uuid.UUID, action string, namespace string, details map[string]interface{}) {
	go func() {
		// background context with timeout for the insert operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var detailsJSON []byte
		var err error

		if details != nil {
			detailsJSON, err = json.Marshal(details)
			if err != nil {
				slog.Error("failed to serialize audit details", "error", err, "action", action)
				detailsJSON = []byte("{}")
			}
		} else {
			detailsJSON = []byte("{}")
		}

		query := `
			INSERT INTO audit_logs (api_key_id, action, namespace, details)
			VALUES ($1, $2, $3, $4)
		`

		_, err = s.db.Pool().Exec(ctx, query, keyID, action, namespace, detailsJSON)
		if err != nil {
			slog.Error("failed to write audit log", "error", err, "action", action, "key_id", keyID)
		}
	}()
}
