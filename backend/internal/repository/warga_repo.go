package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/model"
)

var ErrWargaNotFound = errors.New("warga not found")

type WargaFilter struct {
	Page  int
	Limit int
	Query string
	RT    string
	RW    string
}

type WargaStats struct {
	Total            int `json:"total"`
	ActiveCount      int `json:"active_count"`
	PendingCount     int `json:"pending_count"`
	MissingDocsCount int `json:"missing_docs_count"`
}

type WargaRepository struct {
	db *pgxpool.Pool
}

func NewWargaRepository(db *pgxpool.Pool) *WargaRepository {
	return &WargaRepository{db: db}
}

func (r *WargaRepository) List(ctx context.Context, filter WargaFilter) ([]model.Warga, WargaStats, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	baseCount := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_count,
			COUNT(CASE WHEN (foto_ktp_url IS NULL OR foto_ktp_url = '') AND (foto_kk_url IS NULL OR foto_kk_url = '') THEN 1 END) as pending_count,
			COUNT(CASE WHEN (foto_ktp_url IS NULL OR foto_ktp_url = '') OR (foto_kk_url IS NULL OR foto_kk_url = '') THEN 1 END) as missing_docs_count
		FROM warga
		WHERE deleted_at IS NULL
	`

	baseList := `
		SELECT
			id, nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat,
			rt, rw, no_hp, foto_ktp_url, foto_kk_url,
			kriteria_values,
			is_active, created_by, created_at, updated_at
		FROM warga
		WHERE deleted_at IS NULL
	`

	clauses := []string{}
	args := []interface{}{}
	idx := 1

	if strings.TrimSpace(filter.Query) != "" {
		clauses = append(clauses, fmt.Sprintf("(nama_lengkap ILIKE $%d OR nik ILIKE $%d)", idx, idx))
		args = append(args, "%"+filter.Query+"%")
		idx++
	}
	if strings.TrimSpace(filter.RT) != "" {
		clauses = append(clauses, fmt.Sprintf("rt = $%d", idx))
		args = append(args, filter.RT)
		idx++
	}
	if strings.TrimSpace(filter.RW) != "" {
		clauses = append(clauses, fmt.Sprintf("rw = $%d", idx))
		args = append(args, filter.RW)
		idx++
	}

	filterClauses := ""
	if len(clauses) > 0 {
		filterClauses = " AND " + strings.Join(clauses, " AND ")
	}

	// 1. Get statistics
	var stats WargaStats
	err := r.db.QueryRow(ctx, baseCount+filterClauses, args...).Scan(
		&stats.Total,
		&stats.ActiveCount,
		&stats.PendingCount,
		&stats.MissingDocsCount,
	)
	if err != nil {
		return nil, stats, err
	}

	// 2. Get paginated list
	listArgs := make([]interface{}, len(args))
	copy(listArgs, args)

	listQuery := baseList + filterClauses + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, stats, err
	}
	defer rows.Close()

	result := []model.Warga{}
	for rows.Next() {
		item, err := scanWarga(rows.Scan)
		if err != nil {
			return nil, stats, err
		}
		result = append(result, item)
	}

	if rows.Err() != nil {
		return nil, stats, rows.Err()
	}

	return result, stats, nil
}

func (r *WargaRepository) GetByID(ctx context.Context, id string) (*model.Warga, error) {
	const q = `
		SELECT
			id, nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat,
			rt, rw, no_hp, foto_ktp_url, foto_kk_url,
			kriteria_values,
			is_active, created_by, created_at, updated_at
		FROM warga
		WHERE id = $1 AND deleted_at IS NULL
		LIMIT 1
	`

	item, err := scanWarga(r.db.QueryRow(ctx, q, id).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWargaNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *WargaRepository) Create(ctx context.Context, w model.Warga) (*model.Warga, error) {
	const q = `
		INSERT INTO warga (
			nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat,
			rt, rw, no_hp, foto_ktp_url, foto_kk_url,
			kriteria_values,
			is_active, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12,
			$13, $14
		)
		RETURNING
			id, nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat,
			rt, rw, no_hp, foto_ktp_url, foto_kk_url,
			kriteria_values,
			is_active, created_by, created_at, updated_at
	`

	rawKriteria, _ := json.Marshal(w.KriteriaValues)
	if rawKriteria == nil {
		rawKriteria = []byte("{}")
	}

	item, err := scanWarga(r.db.QueryRow(ctx, q,
		w.NIK,
		w.NoKK,
		w.NamaLengkap,
		w.TanggalLahir,
		w.JenisKelamin,
		w.Alamat,
		w.RT,
		w.RW,
		w.NoHP,
		w.FotoKtpURL,
		w.FotoKKURL,
		rawKriteria,
		w.IsActive,
		w.CreatedBy,
	).Scan)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *WargaRepository) Update(ctx context.Context, id string, w model.Warga) (*model.Warga, error) {
	const q = `
		UPDATE warga SET
			nik = $1,
			no_kk = $2,
			nama_lengkap = $3,
			tanggal_lahir = $4,
			jenis_kelamin = $5,
			alamat = $6,
			rt = $7,
			rw = $8,
			no_hp = $9,
			foto_ktp_url = $10,
			foto_kk_url = $11,
			kriteria_values = $12,
			is_active = $13,
			updated_at = NOW()
		WHERE id = $14 AND deleted_at IS NULL
		RETURNING
			id, nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat,
			rt, rw, no_hp, foto_ktp_url, foto_kk_url,
			kriteria_values,
			is_active, created_by, created_at, updated_at
	`

	rawKriteria, _ := json.Marshal(w.KriteriaValues)
	if rawKriteria == nil {
		rawKriteria = []byte("{}")
	}

	item, err := scanWarga(r.db.QueryRow(ctx, q,
		w.NIK,
		w.NoKK,
		w.NamaLengkap,
		w.TanggalLahir,
		w.JenisKelamin,
		w.Alamat,
		w.RT,
		w.RW,
		w.NoHP,
		w.FotoKtpURL,
		w.FotoKKURL,
		rawKriteria,
		w.IsActive,
		id,
	).Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWargaNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *WargaRepository) SoftDelete(ctx context.Context, id string) error {
	const q = `UPDATE warga SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWargaNotFound
	}
	return nil
}

func scanWarga(scan func(dest ...interface{}) error) (model.Warga, error) {
	var w model.Warga
	var rawKriteria []byte
	err := scan(
		&w.ID,
		&w.NIK,
		&w.NoKK,
		&w.NamaLengkap,
		&w.TanggalLahir,
		&w.JenisKelamin,
		&w.Alamat,
		&w.RT,
		&w.RW,
		&w.NoHP,
		&w.FotoKtpURL,
		&w.FotoKKURL,
		&rawKriteria,
		&w.IsActive,
		&w.CreatedBy,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if err != nil {
		return model.Warga{}, err
	}

	if len(rawKriteria) > 0 {
		_ = json.Unmarshal(rawKriteria, &w.KriteriaValues)
	}
	if w.KriteriaValues == nil {
		w.KriteriaValues = make(map[string]float64)
	}

	return w, nil
}
