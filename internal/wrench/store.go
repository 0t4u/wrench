package wrench

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hexops/wrench/internal/errors"
	"github.com/keegancsmith/sqlf"

	_ "modernc.org/sqlite" // from https://gitlab.com/cznic/sqlite
)

type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errors.Wrap(err, "Open")
	}
	s := &Store{db: db}
	if err := s.ensureSchema(); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "ensureSchema")
	}
	return s, nil
}

func (s *Store) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			logid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			id TEXT NOT NULL,
			message TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS stats (
			statid INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			id TEXT NOT NULL,
			value INTEGER NOT NULL,
			type TEXT NOT NULL,
			metadata BLOB NOT NULL
		);
		CREATE TABLE IF NOT EXISTS runners (
			id TEXT PRIMARY KEY NOT NULL,
			arch TEXT NOT NULL,
			env TEXT NOT NULL,
			registered_at TIMESTAMP NOT NULL,
			last_seen_at TIMESTAMP NOT NULL
		);
		CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY NOT NULL,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS cache (
			cache_name TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP,
			PRIMARY KEY (cache_name, key)
		);
		CREATE TABLE IF NOT EXISTS runner_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
			state TEXT NOT NULL,
			title TEXT NOT NULL,
			target_runner_id TEXT NOT NULL,
			target_runner_arch TEXT NOT NULL,
			payload TEXT NOT NULL,
			scheduled_start_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_logs_id ON logs (id);

		CREATE INDEX IF NOT EXISTS idx_stats_id ON stats (id);

		CREATE INDEX IF NOT EXISTS idx_cache_cache_name ON cache (cache_name);
		CREATE INDEX IF NOT EXISTS idx_cache_key ON cache (key);

		CREATE INDEX IF NOT EXISTS idx_runner_jobs_state ON runner_jobs (state);
		CREATE INDEX IF NOT EXISTS idx_runner_jobs_title ON runner_jobs (title);
		CREATE INDEX IF NOT EXISTS idx_runner_jobs_target_runner_id ON runner_jobs (target_runner_id);
		CREATE INDEX IF NOT EXISTS idx_runner_jobs_id ON runner_jobs (id);
	`)
	return err
}

func (s *Store) Log(ctx context.Context, id, message string) error {
	q := sqlf.Sprintf(
		"INSERT INTO logs(timestamp, id, message) VALUES(%v, %v, %v)",
		time.Now(),
		id,
		strings.TrimSpace(message),
	)
	_, err := s.db.ExecContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	return err
}

type Log struct {
	Time    time.Time
	Message string
}

func (s *Store) Logs(ctx context.Context, id string) ([]Log, error) {
	q := sqlf.Sprintf(`SELECT * FROM (SELECT timestamp, message FROM logs WHERE id=%v) ORDER BY timestamp`, id)

	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "QueryContext")
	}

	var logs []Log
	for rows.Next() {
		var log Log
		if err = rows.Scan(&log.Time, &log.Message); err != nil {
			return nil, errors.Wrap(err, "Scan")
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (s *Store) LogIDs(ctx context.Context) ([]string, error) {
	q := sqlf.Sprintf(`SELECT * FROM (SELECT DISTINCT id FROM logs) ORDER BY id`)

	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "QueryContext")
	}

	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "Scan")
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) StatIDs(ctx context.Context) ([]string, error) {
	q := sqlf.Sprintf(`SELECT * FROM (SELECT DISTINCT id FROM stats) ORDER BY id`)

	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "QueryContext")
	}

	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "Scan")
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

var ErrNotFound = errors.New("not found")

func (s *Store) CacheSet(ctx context.Context, cacheName, key, value string, expires *time.Time) error {
	now := time.Now()
	q := sqlf.Sprintf(
		`INSERT INTO cache(cache_name, key, value, updated_at, created_at, expires_at) VALUES (%v, %v, %v, %v, %v, %v)
		ON CONFLICT(cache_name, key) DO UPDATE SET
			value = %v,
			updated_at = %v,
			expires_at = %v
		WHERE cache_name = %v AND key = %v`,
		cacheName, key, value, now, now, expires,
		value, now, expires,
		cacheName, key,
	)
	_, err := s.db.ExecContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	return err
}

type CacheEntry struct {
	Value   string
	Updated time.Time
	Created time.Time
	Expires *time.Time
}

func (s *Store) CacheKey(ctx context.Context, cacheName, key string) (*CacheEntry, error) {
	q := sqlf.Sprintf(`SELECT value, updated_at, created_at, expires_at
		FROM cache WHERE cache_name = %v AND key = %v`, cacheName, key)

	row := s.db.QueryRowContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	var e CacheEntry
	if err := row.Scan(&e.Value, &e.Updated, &e.Created, &e.Expires); err != nil {
		return nil, errors.Wrap(err, "Scan")
	}
	return &e, nil
}

type Secret struct {
	ID    string
	Value string
}

// Redaction stringer in case it ever gets printed anywhere.
func (s Secret) String() string { return "<redacted>" }

func (s *Store) Secret(ctx context.Context, id string) (Secret, error) {
	q := sqlf.Sprintf(`SELECT value FROM secrets WHERE id = %v`, id)

	row := s.db.QueryRowContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	var value string
	if err := row.Scan(&value); err != nil {
		return Secret{}, errors.Wrap(err, "Scan")
	}
	return Secret{ID: id, Value: value}, nil
}

func (s *Store) Secrets(ctx context.Context) ([]Secret, error) {
	q := sqlf.Sprintf(`SELECT id, value FROM secrets ORDER BY id DESC`)

	rows, err := s.db.QueryContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	if err != nil {
		return nil, errors.Wrap(err, "QueryContext")
	}

	var secrets []Secret
	for rows.Next() {
		var id, value string
		if err = rows.Scan(&id, &value); err != nil {
			return nil, errors.Wrap(err, "Scan")
		}
		secrets = append(secrets, Secret{ID: id, Value: value})
	}
	return secrets, rows.Err()
}

func (s *Store) UpsertSecret(ctx context.Context, id, value string) error {
	q := sqlf.Sprintf(
		`INSERT INTO secrets(id, value) VALUES (%v, %v)
		ON CONFLICT(id) DO UPDATE SET value = %v WHERE id=%v`,
		id, value,
		value, id,
	)
	_, err := s.db.ExecContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	return err
}

func (s *Store) DeleteSecret(ctx context.Context, id string) error {
	q := sqlf.Sprintf(`DELETE FROM secrets WHERE id = %v`, id)
	_, err := s.db.ExecContext(ctx, q.Query(sqlf.SimpleBindVar), q.Args()...)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
