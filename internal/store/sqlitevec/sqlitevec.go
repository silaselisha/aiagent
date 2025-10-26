package sqlitevec

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
    "math"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database used as a vector store.
type DB struct{ sql *sql.DB }

func Open(path string) (*DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil { return nil, err }
	if _, err := d.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`); err != nil { return nil, err }
	db := &DB{sql: d}
	if err := db.migrate(); err != nil { _ = d.Close(); return nil, err }
	return db, nil
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
	CREATE TABLE IF NOT EXISTS feature_windows (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  window_start INTEGER NOT NULL,
	  vector BLOB NOT NULL,
	  label REAL,
	  meta TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_fw_start ON feature_windows(window_start);
	CREATE TABLE IF NOT EXISTS events (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  ts INTEGER NOT NULL,
	  type TEXT NOT NULL,
	  payload TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts);
	CREATE TABLE IF NOT EXISTS calibration (
	  id INTEGER PRIMARY KEY CHECK (id=1),
	  threshold REAL
	);
	`)
	return err
}

// PutFeature stores a vector with optional label and meta.
func (d *DB) PutFeature(ctx context.Context, windowStart time.Time, vec []float32, label *float32, meta any) error {
	bvec := encodeF32(vec)
	var mstr *string
	if meta != nil {
		mb, _ := json.Marshal(meta)
		ms := string(mb)
		mstr = &ms
	}
	_, err := d.sql.ExecContext(ctx, `INSERT INTO feature_windows(window_start, vector, label, meta) VALUES(?,?,?,?)`, windowStart.Unix(), bvec, label, mstr)
	return err
}

// LoadFeatures returns vectors within [start,end).
func (d *DB) LoadFeatures(ctx context.Context, start, end time.Time) ([]time.Time, [][]float32, []float32, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT window_start, vector, COALESCE(label, -1) FROM feature_windows WHERE window_start>=? AND window_start<? ORDER BY window_start`, start.Unix(), end.Unix())
	if err != nil { return nil, nil, nil, err }
	defer rows.Close()
	var ts []time.Time
	var X [][]float32
	var y []float32
	for rows.Next() {
		var ws int64
		var vb []byte
		var lbl float32
		if err := rows.Scan(&ws, &vb, &lbl); err != nil { return nil, nil, nil, err }
		ts = append(ts, time.Unix(ws, 0).UTC())
		X = append(X, decodeF32(vb))
		y = append(y, lbl)
	}
	return ts, X, y, rows.Err()
}

// PutEvent stores an engagement event.
func (d *DB) PutEvent(ctx context.Context, ts time.Time, typ string, payload any) error {
	pb, _ := json.Marshal(payload)
	_, err := d.sql.ExecContext(ctx, `INSERT INTO events(ts, type, payload) VALUES(?,?,?)`, ts.Unix(), typ, string(pb))
	return err
}

// Event is a stored engagement event
type Event struct { TS time.Time; Type string; Payload string }

// LoadEventsRange returns events in [start, end)
func (d *DB) LoadEventsRange(ctx context.Context, start, end time.Time, typ string) ([]Event, error) {
    var rows *sql.Rows
    var err error
    if typ == "" {
        rows, err = d.sql.QueryContext(ctx, `SELECT ts, type, payload FROM events WHERE ts>=? AND ts<? ORDER BY ts`, start.Unix(), end.Unix())
    } else {
        rows, err = d.sql.QueryContext(ctx, `SELECT ts, type, payload FROM events WHERE ts>=? AND ts<? AND type=? ORDER BY ts`, start.Unix(), end.Unix(), typ)
    }
    if err != nil { return nil, err }
    defer rows.Close()
    var out []Event
    for rows.Next() {
        var ts int64; var typ string; var payload string
        if err := rows.Scan(&ts, &typ, &payload); err != nil { return nil, err }
        out = append(out, Event{TS: time.Unix(ts,0).UTC(), Type: typ, Payload: payload})
    }
    return out, rows.Err()
}

// LoadMetasRange returns meta JSON for windows in [start,end)
func (d *DB) LoadMetasRange(ctx context.Context, start, end time.Time) ([]string, error) {
    rows, err := d.sql.QueryContext(ctx, `SELECT meta FROM feature_windows WHERE window_start>=? AND window_start<? ORDER BY window_start`, start.Unix(), end.Unix())
    if err != nil { return nil, err }
    defer rows.Close()
    var out []string
    for rows.Next() {
        var m sql.NullString
        if err := rows.Scan(&m); err != nil { return nil, err }
        if m.Valid { out = append(out, m.String) }
    }
    return out, rows.Err()
}

// SaveThreshold stores a single threshold value.
func (d *DB) SaveThreshold(ctx context.Context, thr float64) error {
	_, err := d.sql.ExecContext(ctx, `INSERT INTO calibration(id, threshold) VALUES(1, ?) ON CONFLICT(id) DO UPDATE SET threshold=excluded.threshold`, thr)
	return err
}

func (d *DB) LoadThreshold(ctx context.Context) (float64, error) {
	row := d.sql.QueryRowContext(ctx, `SELECT threshold FROM calibration WHERE id=1`)
	var thr sql.NullFloat64
	if err := row.Scan(&thr); err != nil { return 0, err }
	if !thr.Valid { return 0, errors.New("no threshold") }
	return thr.Float64, nil
}

func encodeF32(v []float32) []byte {
    b := make([]byte, 4*len(v))
    for i := range v { binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v[i])) }
    return b
}

func decodeF32(b []byte) []float32 {
    n := len(b) / 4
    v := make([]float32, n)
    for i := 0; i < n; i++ { v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:])) }
    return v
}
