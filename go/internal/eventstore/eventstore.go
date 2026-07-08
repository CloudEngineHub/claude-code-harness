package eventstore

import (
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

const BridgeEventsTable = "bridge_events"

const BridgeEventsSchema = `CREATE TABLE IF NOT EXISTS bridge_events (
  event_id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  event_type TEXT NOT NULL,
  lane TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  ts INTEGER NOT NULL
);`

// Event is the shared row shape stored by Bridge Daemon mailbox databases.
type Event struct {
	EventID     string
	Source      string
	EventType   string
	PayloadJSON string
	TS          int64
}

// Open opens the Bridge event sqlite database.
func Open(dbPath string) (*sql.DB, error) {
	return sql.Open("sqlite", dbPath)
}

// EnsureSchema creates the shared bridge_events table.
func EnsureSchema(db *sql.DB) error {
	_, err := db.Exec(BridgeEventsSchema)
	return err
}

// MailboxExists reports whether dbPath exists.
func MailboxExists(dbPath string) (bool, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// MaxTimestamp returns the newest bridge event timestamp.
func MaxTimestamp(db *sql.DB) (sql.NullInt64, error) {
	var maxTS sql.NullInt64
	err := db.QueryRow("SELECT MAX(ts) FROM bridge_events").Scan(&maxTS)
	return maxTS, err
}

// Events returns bridge events in ascending timestamp order.
func Events(db *sql.DB) ([]Event, error) {
	rows, err := db.Query(`SELECT event_id, source, event_type, payload_json, ts FROM bridge_events ORDER BY ts ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.EventID, &event.Source, &event.EventType, &event.PayloadJSON, &event.TS); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
