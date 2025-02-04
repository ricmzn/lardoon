package lardoon

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

const schema = `
CREATE TABLE IF NOT EXISTS replays (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT UNIQUE,
	reference_time TEXT,
	recording_time TEXT,
	title TEXT,
	data_source TEXT,
	data_recorder TEXT,
	duration INTEGER,
	size INTEGER
);

CREATE TABLE IF NOT EXISTS replay_objects (
	replay_id INTEGER,
	object_id INTEGER,
	types TEXT,
	name TEXT,
	pilot TEXT,
	created_offset INTEGER,
	deleted_offset INTEGER,

	UNIQUE(replay_id, object_id),
	FOREIGN KEY(replay_id) REFERENCES replays(id)
);
`

type Replay struct {
	Id            int    `json:"id"`
	Path          string `json:"path"`
	ReferenceTime string `json:"reference_time"`
	RecordingTime string `json:"recording_time"`
	Title         string `json:"title"`
	DataSource    string `json:"data_source"`
	DataRecorder  string `json:"data_recorder"`
	Duration      *int   `json:"duration"`
	Size          int    `json:"size"`
}

type ReplayObject struct {
	Id            int    `json:"id"`
	ReplayId      int    `json:"replay_id"`
	Types         string `json:"types"`
	Name          string `json:"name"`
	Pilot         string `json:"pilot"`
	CreatedOffset int    `json:"created_offset"`
	DeletedOffset int    `json:"deleted_offset"`
}

type ReplayWithObjects struct {
	Replay

	Objects []*ReplayObject `json:"objects"`
}

func beginTransaction() (*sql.Tx, error) {
	return db.Begin()
}

func createReplayObject(tx *sql.Tx, replayId int, objectId int, types string, name string, pilot string, createdOffset int, deletedOffset int) error {
	_, err := tx.Exec(`
		INSERT INTO replay_objects VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (replay_id, object_id) DO UPDATE
			SET types = EXCLUDED.types, name = EXCLUDED.name, pilot = EXCLUDED.pilot, created_offset = EXCLUDED.created_offset, deleted_offset = EXCLUDED.deleted_offset
	`, replayId, objectId, types, name, pilot, createdOffset, deletedOffset)
	return err
}

func setReplayDuration(tx *sql.Tx, replayId, duration int) error {
	_, err := tx.Exec(`UPDATE replays SET duration=? WHERE id=?`, duration, replayId)
	return err
}

func createReplay(tx *sql.Tx, path string, referenceTime string, recordingTime string, title string, dataSource string, dataRecorder string, size int) (int, error) {
	row, err := tx.Query(`SELECT id, duration, size FROM replays WHERE path=?`, path)
	if err != nil {
		return -1, err
	}

	if row.Next() {
		var id int
		var dur *int
		var exSize int
		err = row.Scan(&id, &dur, &exSize)
		row.Close()
		if err != nil {
			return -1, err
		}

		if dur != nil && size == exSize {
			return -1, nil
		}

		_, err = tx.Exec(
			`UPDATE replays SET path=?, reference_time=?, recording_time=?, title=?, data_source=?, data_recorder=?, size=? WHERE id=?`,
			path, referenceTime, recordingTime, title, dataSource, dataRecorder, size, id,
		)
		return id, err
	} else {
		row.Close()
	}

	row, err = tx.Query(`
		INSERT INTO replays (path, reference_time, recording_time, title, data_source, data_recorder, size) VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, path, referenceTime, recordingTime, title, dataSource, dataRecorder, size)
	if err != nil {
		return -1, err
	}
	defer row.Close()

	row.Next()

	var id int
	err = row.Scan(&id)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func InitDatabase(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	_, err = db.Exec(schema)
	return err
}
