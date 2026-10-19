package store

import (
	"context"
	"time"
)

// AgentCheckin is what an installed agent reports about itself (roadmap M1/M10).
type AgentCheckin struct {
	DeviceID     string `json:"device_id"`
	AgentVersion string `json:"agent_version"`
	OS           string `json:"os"` // windows | macos | linux
	OSVersion    string `json:"os_version"`
	Hostname     string `json:"hostname"`
}

// RecordAgentCheckin upserts the device and its health fields for the user.
func (s *Store) RecordAgentCheckin(ctx context.Context, userID string, a AgentCheckin) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO devices (id, user_id, label, last_seen_at, last_checkin_at, agent_version, os, os_version, hostname)
		VALUES ($1::uuid, $2, NULLIF($6, ''), now(), now(), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''))
		ON CONFLICT (id) DO UPDATE SET
		  user_id = EXCLUDED.user_id, last_checkin_at = now(), last_seen_at = now(),
		  agent_version = COALESCE(EXCLUDED.agent_version, devices.agent_version),
		  os = COALESCE(EXCLUDED.os, devices.os),
		  os_version = COALESCE(EXCLUDED.os_version, devices.os_version),
		  hostname = COALESCE(EXCLUDED.hostname, devices.hostname)`,
		a.DeviceID, userID, a.AgentVersion, a.OS, a.OSVersion, a.Hostname)
	return err
}

// Agent is one installed agent as shown in the admin's Agent health view.
type Agent struct {
	DeviceID       string     `json:"device_id"`
	UserID         string     `json:"user_id"`
	DisplayName    string     `json:"display_name"`
	Email          string     `json:"email"`
	DepartmentName string     `json:"department_name"`
	Hostname       string     `json:"hostname"`
	OS             string     `json:"os"`
	OSVersion      string     `json:"os_version"`
	AgentVersion   string     `json:"agent_version"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
}

// ListAgents returns the agents of a business's active members; departmentID
// ("" = all) limits it for managers.
func (s *Store) ListAgents(ctx context.Context, businessID, departmentID string) ([]Agent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.id::text, u.id::text, u.display_name, COALESCE(u.email, ''), COALESCE(m.department_name, ''),
		       COALESCE(d.hostname, d.label, ''), COALESCE(d.os, ''), COALESCE(d.os_version, ''),
		       COALESCE(d.agent_version, ''), d.last_seen_at
		  FROM devices d
		  JOIN memberships m ON m.user_id = d.user_id AND m.business_id = $1 AND m.deactivated_at IS NULL
		  JOIN users u ON u.id = d.user_id
		 WHERE ($2 = '' OR m.department_id = $2)
		 ORDER BY d.last_seen_at DESC NULLS LAST`, businessID, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Agent{}
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.DeviceID, &a.UserID, &a.DisplayName, &a.Email, &a.DepartmentName,
			&a.Hostname, &a.OS, &a.OSVersion, &a.AgentVersion, &a.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
