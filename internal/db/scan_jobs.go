// internal/db/scan_jobs.go
package db

import "time"

type ScanJob struct {
	ID           int64
	Type         string
	StartedAt    time.Time
	FinishedAt   *time.Time
	FilesScanned int64
	FilesAdded   int64
	FilesUpdated int64
	FilesRemoved int64
	Error        *string
}

func (d *DB) StartScanJob(kind string) (int64, error) {
	res, err := d.Exec(`INSERT INTO scan_jobs(type,started_at) VALUES(?,CURRENT_TIMESTAMP)`, kind)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) FinishScanJob(id int64, scanned, added, updated, removed int64, errMsg string) error {
	var errVal any
	if errMsg != "" {
		errVal = errMsg
	}
	_, err := d.Exec(`
		UPDATE scan_jobs SET finished_at=CURRENT_TIMESTAMP,
		  files_scanned=?, files_added=?, files_updated=?, files_removed=?, error=?
		WHERE id=?
	`, scanned, added, updated, removed, errVal, id)
	return err
}

func (d *DB) LastScanJob(kind string) (*ScanJob, error) {
	row := d.QueryRow(`
		SELECT id,type,started_at,finished_at,files_scanned,files_added,files_updated,files_removed,error
		FROM scan_jobs WHERE type=? ORDER BY id DESC LIMIT 1
	`, kind)
	var j ScanJob
	var fin *time.Time
	var emsg *string
	if err := row.Scan(&j.ID, &j.Type, &j.StartedAt, &fin, &j.FilesScanned, &j.FilesAdded, &j.FilesUpdated, &j.FilesRemoved, &emsg); err != nil {
		return nil, err
	}
	j.FinishedAt = fin
	j.Error = emsg
	return &j, nil
}
