package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge"
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

func FindThumbnails(db *sql.DB, ctx context.Context, find forge.ThumbnailFinder) ([]*forge.Thumbnail, error) {
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

func findThumbnails(tx *sql.Tx, ctx context.Context, find forge.ThumbnailFinder) ([]*forge.Thumbnail, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			thumbnails.id,
			thumbnails.data,
			entries.path
		FROM thumbnails
		LEFT JOIN entries ON thumbnails.entry_id = entries.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	thumbs := make([]*forge.Thumbnail, 0)
	for rows.Next() {
		thumb := &forge.Thumbnail{}
		err := rows.Scan(
			&thumb.ID,
			&thumb.Data,
			&thumb.EntryPath,
		)
		if err != nil {
			return nil, err
		}
		thumbs = append(thumbs, thumb)
	}
	return thumbs, nil
}

func GetThumbnail(db *sql.DB, ctx context.Context, path string) (*forge.Thumbnail, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	thumb, err := getThumbnail(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return thumb, nil
}

func getThumbnail(tx *sql.Tx, ctx context.Context, path string) (*forge.Thumbnail, error) {
	thumbs, err := findThumbnails(tx, ctx, forge.ThumbnailFinder{EntryPath: &path})
	if err != nil {
		return nil, err
	}
	if len(thumbs) == 0 {
		return nil, forge.NotFound("thumbnail not found")
	}
	return thumbs[0], nil
}

func AddThumbnail(db *sql.DB, ctx context.Context, thumb *forge.Thumbnail) error {
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

func addThumbnail(tx *sql.Tx, ctx context.Context, thumb *forge.Thumbnail) error {
	err := userWrite(tx, ctx, thumb.EntryPath)
	if err != nil {
		return err
	}
	entryID, err := getEntryID(tx, ctx, thumb.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO thumbnails (
			entry_id,
			data
		)
		VALUES (?, ?)
	`,
		entryID,
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
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: thumb.EntryPath,
		User:      user,
		Action:    "add",
		Category:  "thumbnail",
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateThumbnail(db *sql.DB, ctx context.Context, upd forge.ThumbnailUpdater) error {
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

func updateThumbnail(tx *sql.Tx, ctx context.Context, upd forge.ThumbnailUpdater) error {
	err := userWrite(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	thumb, err := getThumbnail(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.Data != nil {
		keys = append(keys, "data=?")
		vals = append(vals, upd.Data)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update thumbnail: %v", upd.EntryPath)
	}
	vals = append(vals, thumb.ID) // for where clause
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
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: upd.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "thumbnail",
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
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	entryID, err := getEntryID(tx, ctx, path)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM thumbnails
		WHERE entry_id=?
	`,
		entryID,
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
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: path,
		User:      user,
		Action:    "delete",
		Category:  "thumbnail",
	})
	if err != nil {
		return err
	}
	return nil
}
