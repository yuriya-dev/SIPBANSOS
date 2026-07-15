package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wahyutricahya/SIPBANSOS/backend/internal/model"
)

var ErrKriteriaNotFound = errors.New("kriteria not found")

type KriteriaRepository struct {
  db *pgxpool.Pool
}

func NewKriteriaRepository(db *pgxpool.Pool) *KriteriaRepository {
  return &KriteriaRepository{db: db}
}

func (r *KriteriaRepository) GetActiveOrLatest(ctx context.Context) (*model.KriteriaBobot, error) {
  const activeQuery = `
    SELECT
      id, versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh, created_at
    FROM bobot_kriteria
    WHERE is_active = TRUE
    ORDER BY created_at DESC
    LIMIT 1
  `

  item, err := scanKriteria(r.db.QueryRow(ctx, activeQuery).Scan)
  if err == nil {
    return &item, nil
  }

  if !errors.Is(err, pgx.ErrNoRows) {
    return nil, err
  }

  const latestQuery = `
    SELECT
      id, versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh, created_at
    FROM bobot_kriteria
    ORDER BY created_at DESC
    LIMIT 1
  `

  item, err = scanKriteria(r.db.QueryRow(ctx, latestQuery).Scan)
  if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
      return nil, ErrKriteriaNotFound
    }
    return nil, err
  }

  return &item, nil
}

func (r *KriteriaRepository) GetByID(ctx context.Context, id string) (*model.KriteriaBobot, error) {
  const q = `
    SELECT
      id, versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh, created_at
    FROM bobot_kriteria
    WHERE id = $1
    LIMIT 1
  `

  item, err := scanKriteria(r.db.QueryRow(ctx, q, id).Scan)
  if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
      return nil, ErrKriteriaNotFound
    }
    return nil, err
  }

  return &item, nil
}

func (r *KriteriaRepository) Create(ctx context.Context, data model.KriteriaBobot) (*model.KriteriaBobot, error) {
  const q = `
    INSERT INTO bobot_kriteria (
      versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh
    ) VALUES (
      $1, $2, $3, $4, $5
    )
    RETURNING
      id, versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh, created_at
  `

  rawBobot, _ := json.Marshal(data.BobotValues)
  if rawBobot == nil {
    rawBobot = []byte("{}")
  }

  item, err := scanKriteria(r.db.QueryRow(ctx, q,
    data.Versi,
    data.Keterangan,
    rawBobot,
    data.IsActive,
    data.DibuatOleh,
  ).Scan)
  if err != nil {
    return nil, err
  }

  // If this one is active, deactivate others
  if data.IsActive {
    _, _ = r.db.Exec(ctx, `UPDATE bobot_kriteria SET is_active = FALSE WHERE id <> $1`, item.ID)
  }

  return &item, nil
}

func (r *KriteriaRepository) Update(ctx context.Context, id string, data model.KriteriaBobot, activate bool) (*model.KriteriaBobot, error) {
  tx, err := r.db.Begin(ctx)
  if err != nil {
    return nil, err
  }
  defer tx.Rollback(ctx)

  if activate {
    if _, err := tx.Exec(ctx, `UPDATE bobot_kriteria SET is_active = FALSE WHERE is_active = TRUE AND id <> $1`, id); err != nil {
      return nil, err
    }
  }

  const q = `
    UPDATE bobot_kriteria SET
      versi = $1,
      keterangan = $2,
      bobot_values = $3,
      is_active = $4
    WHERE id = $5
    RETURNING
      id, versi, keterangan,
      bobot_values,
      is_active, dibuat_oleh, created_at
  `

  rawBobot, _ := json.Marshal(data.BobotValues)
  if rawBobot == nil {
    rawBobot = []byte("{}")
  }

  item, err := scanKriteria(tx.QueryRow(ctx, q,
    data.Versi,
    data.Keterangan,
    rawBobot,
    data.IsActive,
    id,
  ).Scan)
  if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
      return nil, ErrKriteriaNotFound
    }
    return nil, err
  }

  if err := tx.Commit(ctx); err != nil {
    return nil, err
  }

  return &item, nil
}

func scanKriteria(scan func(dest ...interface{}) error) (model.KriteriaBobot, error) {
  var item model.KriteriaBobot
  var rawBobot []byte
  err := scan(
    &item.ID,
    &item.Versi,
    &item.Keterangan,
    &rawBobot,
    &item.IsActive,
    &item.DibuatOleh,
    &item.CreatedAt,
  )
  if err != nil {
    return model.KriteriaBobot{}, err
  }

  if len(rawBobot) > 0 {
    _ = json.Unmarshal(rawBobot, &item.BobotValues)
  }
  if item.BobotValues == nil {
    item.BobotValues = make(map[string]float64)
  }

  return item, nil
}
