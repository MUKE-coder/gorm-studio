package studio

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditEvent describes a single mutating action performed through Studio.
// It is passed to the AuditLogger configured in Config.
type AuditEvent struct {
	// Time is set by Studio when the event is emitted (UTC).
	Time time.Time `json:"time"`
	// Actor identifies who performed the action, if the auth middleware
	// recorded it in the context (see actorFromContext).
	Actor string `json:"actor,omitempty"`
	// Action is a short verb, e.g. "create_row", "update_row", "delete_row",
	// "bulk_delete", "sql", "import_data", "import_schema", "import_models",
	// "export_data".
	Action string `json:"action"`
	// Table is the affected table, when the action targets a single table.
	Table string `json:"table,omitempty"`
	// Tables lists affected tables for actions that touch several (imports).
	Tables []string `json:"tables,omitempty"`
	// RowID is the primary key of the affected row, for single-row actions.
	RowID string `json:"row_id,omitempty"`
	// Rows is the number of rows affected/returned.
	Rows int64 `json:"rows,omitempty"`
	// Query is the raw statement, for SQL editor actions.
	Query string `json:"query,omitempty"`
	// Success reports whether the action completed without error.
	Success bool `json:"success"`
	// Err holds the error message when Success is false.
	Err string `json:"err,omitempty"`
}

// auditActorKeys are context keys checked, in order, to identify the actor.
// An AuthMiddleware can set any of these; gin.BasicAuth sets gin.AuthUserKey.
var auditActorKeys = []string{"studio_user", "user", "username", gin.AuthUserKey}

func actorFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	for _, key := range auditActorKeys {
		if v, ok := c.Get(key); ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// audit emits an event to the configured AuditLogger, filling in the time and
// actor. It is a no-op when no logger is configured.
func (h *Handlers) audit(c *gin.Context, e AuditEvent) {
	if h.Audit == nil {
		return
	}
	e.Time = time.Now().UTC()
	if e.Actor == "" {
		e.Actor = actorFromContext(c)
	}
	h.Audit(e)
}

// DefaultAuditLogger writes audit events to the standard logger. It is a
// convenient value for Config.AuditLogger.
func DefaultAuditLogger(e AuditEvent) {
	actor := e.Actor
	if actor == "" {
		actor = "-"
	}
	target := e.Table
	if target == "" && len(e.Tables) > 0 {
		target = joinTables(e.Tables)
	}
	if e.Err != "" {
		log.Printf("[GORM Studio audit] actor=%s action=%s table=%s id=%s rows=%d success=%v err=%q",
			actor, e.Action, target, e.RowID, e.Rows, e.Success, e.Err)
		return
	}
	log.Printf("[GORM Studio audit] actor=%s action=%s table=%s id=%s rows=%d success=%v",
		actor, e.Action, target, e.RowID, e.Rows, e.Success)
}

func joinTables(tables []string) string {
	out := ""
	for i, t := range tables {
		if i > 0 {
			out += ","
		}
		out += t
	}
	return out
}
