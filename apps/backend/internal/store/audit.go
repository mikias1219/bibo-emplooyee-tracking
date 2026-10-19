package store

import (
	"context"
	"encoding/json"
	"time"
)

// AuditEntry is one audit-log row (roadmap M9: who viewed or changed what, when).
type AuditEntry struct {
	ID          int64           `json:"id"`
	ActorUserID string          `json:"actor_user_id"`
	ActorName   string          `json:"actor_name"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    string          `json:"target_id"`
	Meta        json.RawMessage `json:"meta,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// RecordAudit appends an audit entry. businessID may be "" for org-less actions.
func (s *Store) RecordAudit(ctx context.Context, businessID, actorID, action, targetType, targetID string, meta any) error {
	var metaJSON []byte
	if meta != nil {
		metaJSON, _ = json.Marshal(meta)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_log (business_id, actor_user_id, action, target_type, target_id, meta)
		VALUES (NULLIF($1, '')::uuid, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), NULLIF($5, ''), $6)`,
		businessID, actorID, action, targetType, targetID, metaJSON)
	return err
}

// ListAudit returns the newest audit entries of a business (paged).
func (s *Store) ListAudit(ctx context.Context, businessID string, limit, offset int) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, COALESCE(a.actor_user_id::text, ''), COALESCE(u.display_name, ''), a.action,
		       COALESCE(a.target_type, ''), COALESCE(a.target_id, ''), a.meta, a.created_at
		  FROM audit_log a LEFT JOIN users u ON u.id = a.actor_user_id
		 WHERE a.business_id = $1
		 ORDER BY a.created_at DESC
		 LIMIT $2 OFFSET $3`, businessID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorName, &e.Action, &e.TargetType, &e.TargetID, &e.Meta, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// BusinessOfOwnedEmployee returns the business through which ownerID sees empID
// ("" if none) — used to file report views under the right organization.
func (s *Store) BusinessOfOwnedEmployee(ctx context.Context, ownerID, empID string) string {
	var id string
	_ = s.pool.QueryRow(ctx, `
		SELECT m.business_id::text FROM memberships m
		 WHERE m.user_id = $1 AND m.business_id IN `+ownerOf+` LIMIT 1`, empID, ownerID).Scan(&id)
	return id
}

// BusinessOfScreenshot returns the business a screenshot belongs to ("" if unknown).
func (s *Store) BusinessOfScreenshot(ctx context.Context, clientUUID string) (businessID, userID string) {
	_ = s.pool.QueryRow(ctx, `SELECT business_id::text, user_id::text FROM screenshots WHERE client_uuid = $1::uuid`, clientUUID).
		Scan(&businessID, &userID)
	return businessID, userID
}
