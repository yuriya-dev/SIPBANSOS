package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wahyutricahya/SIPBANSOS/backend/internal/model"
)

type ReportRepository struct {
  db *pgxpool.Pool
}

func NewReportRepository(db *pgxpool.Pool) *ReportRepository {
  return &ReportRepository{db: db}
}

func (r *ReportRepository) ListPeriods(ctx context.Context) ([]model.PeriodeBansos, error) {
  const q = `
    SELECT id, nama_periode, tanggal_mulai, tanggal_selesai, kuota, bobot_id, status, created_at
    FROM periode_bansos
    ORDER BY tanggal_mulai DESC
  `

  rows, err := r.db.Query(ctx, q)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  items := []model.PeriodeBansos{}
  for rows.Next() {
    var item model.PeriodeBansos
    if err := rows.Scan(
      &item.ID,
      &item.NamaPeriode,
      &item.TanggalMulai,
      &item.TanggalSelesai,
      &item.Kuota,
      &item.BobotID,
      &item.Status,
      &item.CreatedAt,
    ); err != nil {
      return nil, err
    }
    items = append(items, item)
  }

  if rows.Err() != nil {
    return nil, rows.Err()
  }

  return items, nil
}

func (r *ReportRepository) GetRanking(ctx context.Context, periodeID string, status string, limit int) ([]model.HasilSAWReport, error) {
  base := `
    SELECT
      w.no_kk, h.periode_id, h.warga_id,
      w.nama_lengkap, w.nik, w.rt, w.rw,
      h.nilai_vi, h.ranking, h.status
    FROM hasil_saw h
    JOIN warga w ON w.id = h.warga_id
    WHERE h.periode_id = $1
  `

  args := []interface{}{periodeID}
  idx := 2

  if strings.TrimSpace(status) != "" {
    base += fmt.Sprintf(" AND h.status = $%d", idx)
    args = append(args, status)
    idx++
  }

  base += " ORDER BY h.ranking ASC"

  if limit > 0 {
    base += fmt.Sprintf(" LIMIT $%d", idx)
    args = append(args, limit)
  }

  rows, err := r.db.Query(ctx, base, args...)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  items := []model.HasilSAWReport{}
  for rows.Next() {
    var item model.HasilSAWReport
    if err := rows.Scan(
      &item.ID,
      &item.PeriodeID,
      &item.WargaID,
      &item.NamaLengkap,
      &item.NIK,
      &item.RT,
      &item.RW,
      &item.NilaiVI,
      &item.Ranking,
      &item.Status,
    ); err != nil {
      return nil, err
    }
    items = append(items, item)
  }

  if rows.Err() != nil {
    return nil, rows.Err()
  }

  return items, nil
}

func (r *ReportRepository) GetSummary(ctx context.Context, periodeID string) (model.HasilSAWSummary, error) {
  const q = `
    SELECT status, COUNT(*)
    FROM hasil_saw
    WHERE periode_id = $1
    GROUP BY status
  `

  rows, err := r.db.Query(ctx, q, periodeID)
  if err != nil {
    return model.HasilSAWSummary{}, err
  }
  defer rows.Close()

  summary := model.HasilSAWSummary{}
  for rows.Next() {
    var status string
    var count int
    if err := rows.Scan(&status, &count); err != nil {
      return model.HasilSAWSummary{}, err
    }
    summary.Total += count
    switch strings.ToLower(strings.TrimSpace(status)) {
    case "penerima":
      summary.Penerima = count
    case "cadangan":
      summary.Cadangan = count
    case "tidak lolos", "tidak_lolos", "tidak-lolos":
      summary.TidakLolos = count
    }
  }

  if rows.Err() != nil {
    return model.HasilSAWSummary{}, rows.Err()
  }

  // Fetch average nilai_vi
  var avg float64
  err = r.db.QueryRow(ctx, "SELECT COALESCE(AVG(nilai_vi), 0.0) FROM hasil_saw WHERE periode_id = $1", periodeID).Scan(&avg)
  if err == nil {
    summary.RataRata = avg
  }

  return summary, nil
}
