package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createThumbnailsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS thumbnails (
			id INTEGER PRIMARY KEY,
			entry_id STRING NOT NULL UNIQUE,
			data BLOB NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_thumbnails_entry_id ON thumbnails (entry_id)`)
	return err
}

func FindThumbnails(db *sql.DB, ctx context.Context, find service.ThumbnailFinder) ([]*service.Thumbnail, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	users, err := findThumbnails(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func findThumbnails(tx *sql.Tx, ctx context.Context, find service.ThumbnailFinder) ([]*service.Thumbnail, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.EntryID != nil {
		keys = append(keys, "entry_id=?")
		vals = append(vals, *find.EntryID)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			entry_id,
			data
		FROM thumbnails
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	thumbs := make([]*service.Thumbnail, 0)
	for rows.Next() {
		thumb := &service.Thumbnail{}
		err := rows.Scan(
			&thumb.ID,
			&thumb.EntryID,
			&thumb.Data,
		)
		if err != nil {
			return nil, err
		}
		thumbs = append(thumbs, thumb)
	}
	return thumbs, nil
}

func GetThumbnail(db *sql.DB, ctx context.Context, path string) (*service.Thumbnail, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	thumb, err := getThumbnailByPath(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return thumb, nil
}

func getThumbnailByID(tx *sql.Tx, ctx context.Context, id int) (*service.Thumbnail, error) {
	thumbs, err := findThumbnails(tx, ctx, service.ThumbnailFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(thumbs) == 0 {
		return nil, service.NotFound("thumbnail not found")
	}
	return thumbs[0], nil
}

func getThumbnailByPath(tx *sql.Tx, ctx context.Context, path string) (*service.Thumbnail, error) {
	e, err := getEntryByPath(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	thumbs, err := findThumbnails(tx, ctx, service.ThumbnailFinder{EntryID: &e.ID})
	if err != nil {
		return nil, err
	}
	if len(thumbs) == 0 {
		return nil, service.NotFound("thumbnail not found")
	}
	return thumbs[0], nil
}

func AddThumbnail(db *sql.DB, ctx context.Context, thumb *service.Thumbnail) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addThumbnail(tx, ctx, thumb)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addThumbnail(tx *sql.Tx, ctx context.Context, thumb *service.Thumbnail) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO thumbnails (
			entry_id,
			data
		)
		VALUES (?, ?)
	`,
		thumb.EntryID,
		thumb.Data,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	thumb.ID = int(id)
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  thumb.EntryID,
		User:     user,
		Action:   "add",
		Category: "thumbnail",
	})
	return nil
}

func UpdateThumbnail(db *sql.DB, ctx context.Context, upd service.ThumbnailUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateThumbnail(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateThumbnail(tx *sql.Tx, ctx context.Context, upd service.ThumbnailUpdater) error {
	thumb, err := getThumbnailByID(tx, ctx, upd.ID)
	if err != nil {
		return err
	}
	err = userWrite(tx, ctx, thumb.EntryID)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Data != nil {
		keys = append(keys, "data=?")
		vals = append(vals, upd.Data)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
	result, err := tx.ExecContext(ctx, `
		UPDATE thumbnails
		SET `+strings.Join(keys, ", ")+`
		WHERE id=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 property affected, got %v", n)
	}
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  thumb.EntryID,
		User:     user,
		Action:   "update",
		Category: "thumbnail",
	})
	if err != nil {
		return nil
	}
	return nil
}

func DeleteThumbnail(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteThumbnail(tx, ctx, path)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteThumbnail(tx *sql.Tx, ctx context.Context, path string) error {
	e, err := getEntryByPath(tx, ctx, path)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM thumbnails
		WHERE entry_id=?
	`,
		e.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 thumbnail affected, got %v", n)
	}
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  e.ID,
		User:     user,
		Action:   "delete",
		Category: "thumbnail",
	})
	return nil
}