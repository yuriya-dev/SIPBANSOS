package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/saw"
)

func (h *Handler) ListPeriode(c *gin.Context) {
  data, err := h.reports.ListPeriods(c.Request.Context())
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load periode"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) ReportRanking(c *gin.Context) {
  periodeID := strings.TrimSpace(c.Query("periode_id"))
  if periodeID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "periode_id is required"})
    return
  }

  _ = h.ensureSAWCalculated(c.Request.Context(), periodeID)

  status := strings.TrimSpace(c.Query("status"))
  limit := parseOptionalLimit(c.Query("limit"))

  data, err := h.reports.GetRanking(c.Request.Context(), periodeID, status, limit)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load ranking"})
    return
  }

  summary, err := h.reports.GetSummary(c.Request.Context(), periodeID)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load summary"})
    return
  }

  c.JSON(http.StatusOK, gin.H{
    "data":    data,
    "summary": summary,
  })
}

func (h *Handler) ReportSummary(c *gin.Context) {
  periodeID := strings.TrimSpace(c.Query("periode_id"))
  if periodeID == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "periode_id is required"})
    return
  }

  _ = h.ensureSAWCalculated(c.Request.Context(), periodeID)

  summary, err := h.reports.GetSummary(c.Request.Context(), periodeID)
  if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load summary"})
    return
  }

  c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (h *Handler) ReportRekap(c *gin.Context) {
	periodeID := strings.TrimSpace(c.Query("periode_id"))
	if periodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periode_id is required"})
		return
	}

	_ = h.ensureSAWCalculated(c.Request.Context(), periodeID)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT 
			COALESCE(w.rt, '-'), COALESCE(w.rw, '-'),
			COUNT(CASE WHEN h.status = 'Penerima' THEN 1 END) as penerima,
			COUNT(CASE WHEN h.status = 'Cadangan' THEN 1 END) as cadangan,
			COUNT(CASE WHEN h.status = 'Tidak Lolos' THEN 1 END) as tidak_lolos,
			COUNT(*) as total
		FROM hasil_saw h
		JOIN warga w ON w.id = h.warga_id
		WHERE h.periode_id = $1
		GROUP BY w.rt, w.rw
		ORDER BY w.rt ASC, w.rw ASC
	`, periodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load rekap: " + err.Error()})
		return
	}
	defer rows.Close()

	type rekapItem struct {
		RT         string `json:"rt"`
		RW         string `json:"rw"`
		Penerima   int    `json:"penerima"`
		Cadangan   int    `json:"cadangan"`
		TidakLolos int    `json:"tidak_lolos"`
		Total      int    `json:"total"`
	}

	var list []rekapItem
	for rows.Next() {
		var item rekapItem
		err := rows.Scan(&item.RT, &item.RW, &item.Penerima, &item.Cadangan, &item.TidakLolos, &item.Total)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan rekap"})
			return
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	limitVal := 1000
	if l := strings.TrimSpace(c.Query("limit")); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limitVal = parsed
		}
	}

	rows, err := h.db.Query(ctx, `
		SELECT a.created_at, COALESCE(u.username, 'system'), a.aksi, COALESCE(a.tabel, ''), COALESCE(a.record_id::text, ''), COALESCE(a.ip_address::text, '')
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.user_id
		ORDER BY a.created_at DESC
		LIMIT $1
	`, limitVal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load audit logs: " + err.Error()})
		return
	}
	defer rows.Close()

	type auditItem struct {
		CreatedAt time.Time `json:"created_at"`
		Username  string    `json:"username"`
		Aksi      string    `json:"aksi"`
		Tabel     string    `json:"tabel"`
		RecordID  string    `json:"record_id"`
		IPAddress string    `json:"ip_address"`
	}

	var logs []auditItem
	for rows.Next() {
		var item auditItem
		err := rows.Scan(&item.CreatedAt, &item.Username, &item.Aksi, &item.Tabel, &item.RecordID, &item.IPAddress)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan audit log: " + err.Error()})
			return
		}
		logs = append(logs, item)
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func (h *Handler) ExportReport(c *gin.Context) {
	ctx := c.Request.Context()
	periodeID := strings.TrimSpace(c.Query("periode_id"))
	reportType := strings.ToLower(strings.TrimSpace(c.Query("type")))

	if reportType != "audit" && periodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periode_id is required"})
		return
	}

	if reportType != "audit" && periodeID != "" {
		_ = h.ensureSAWCalculated(ctx, periodeID)
	}

	filename := fmt.Sprintf("laporan_%s_%s.csv", reportType, time.Now().Format("20060102"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	switch reportType {
	case "sk":
		headers := []string{"Ranking", "NIK", "Nama Lengkap", "RT", "RW", "Skor Vi", "Status"}
		if err := writer.Write(headers); err != nil {
			return
		}

		rows, err := h.db.Query(ctx, `
			SELECT h.ranking, w.nik, w.nama_lengkap, COALESCE(w.rt, '-'), COALESCE(w.rw, '-'), h.nilai_vi, h.status
			FROM hasil_saw h
			JOIN warga w ON w.id = h.warga_id
			WHERE h.periode_id = $1 AND h.status = 'Penerima'
			ORDER BY h.ranking ASC
		`, periodeID)
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var rank int
			var nik, nama, rt, rw, status string
			var vi float64
			if err := rows.Scan(&rank, &nik, &nama, &rt, &rw, &vi, &status); err != nil {
				return
			}
			row := []string{
				strconv.Itoa(rank),
				nik,
				nama,
				rt,
				rw,
				fmt.Sprintf("%.4f", vi),
				status,
			}
			_ = writer.Write(row)
		}

	case "ranking":
		headers := []string{"Ranking", "NIK", "Nama Lengkap", "RT", "RW", "Skor Vi", "Status"}
		if err := writer.Write(headers); err != nil {
			return
		}

		rows, err := h.db.Query(ctx, `
			SELECT h.ranking, w.nik, w.nama_lengkap, COALESCE(w.rt, '-'), COALESCE(w.rw, '-'), h.nilai_vi, h.status
			FROM hasil_saw h
			JOIN warga w ON w.id = h.warga_id
			WHERE h.periode_id = $1
			ORDER BY h.ranking ASC
		`, periodeID)
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var rank int
			var nik, nama, rt, rw, status string
			var vi float64
			if err := rows.Scan(&rank, &nik, &nama, &rt, &rw, &vi, &status); err != nil {
				return
			}
			row := []string{
				strconv.Itoa(rank),
				nik,
				nama,
				rt,
				rw,
				fmt.Sprintf("%.4f", vi),
				status,
			}
			_ = writer.Write(row)
		}

	case "rekap":
		headers := []string{"RT", "RW", "Jumlah Penerima", "Jumlah Cadangan", "Jumlah Tidak Lolos", "Total Alternatif"}
		if err := writer.Write(headers); err != nil {
			return
		}

		rows, err := h.db.Query(ctx, `
			SELECT 
				COALESCE(w.rt, '-'), COALESCE(w.rw, '-'),
				COUNT(CASE WHEN h.status = 'Penerima' THEN 1 END) as penerima,
				COUNT(CASE WHEN h.status = 'Cadangan' THEN 1 END) as cadangan,
				COUNT(CASE WHEN h.status = 'Tidak Lolos' THEN 1 END) as tidak_lolos,
				COUNT(*) as total
			FROM hasil_saw h
			JOIN warga w ON w.id = h.warga_id
			WHERE h.periode_id = $1
			GROUP BY w.rt, w.rw
			ORDER BY w.rt ASC, w.rw ASC
		`, periodeID)
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var rt, rw string
			var pen, cad, tlos, tot int
			if err := rows.Scan(&rt, &rw, &pen, &cad, &tlos, &tot); err != nil {
				return
			}
			row := []string{
				rt, rw,
				strconv.Itoa(pen),
				strconv.Itoa(cad),
				strconv.Itoa(tlos),
				strconv.Itoa(tot),
			}
			_ = writer.Write(row)
		}

	case "audit":
		headers := []string{"Timestamp", "Username", "Aksi", "Tabel", "Record ID", "IP Address"}
		if err := writer.Write(headers); err != nil {
			return
		}

		rows, err := h.db.Query(ctx, `
			SELECT a.created_at, COALESCE(u.username, 'system'), a.aksi, COALESCE(a.tabel, ''), COALESCE(a.record_id::text, ''), COALESCE(a.ip_address::text, '')
			FROM audit_log a
			LEFT JOIN users u ON u.id = a.user_id
			ORDER BY a.created_at DESC
			LIMIT 1000
		`)
		if err != nil {
			return
		}
		defer rows.Close()

		for rows.Next() {
			var createdAt time.Time
			var user, aksi, tabel, record, ip string
			if err := rows.Scan(&createdAt, &user, &aksi, &tabel, &record, &ip); err != nil {
				return
			}
			row := []string{
				createdAt.Format("2006-01-02 15:04:05"),
				user,
				aksi,
				tabel,
				record,
				ip,
			}
			_ = writer.Write(row)
		}
	}
}

func parseOptionalLimit(value string) int {
  value = strings.TrimSpace(value)
  if value == "" {
    return 0
  }
  parsed, err := strconv.Atoi(value)
  if err != nil || parsed <= 0 {
    return 0
  }
  return parsed
}

func (h *Handler) GetWeeklyActivity(c *gin.Context) {
	ctx := c.Request.Context()
	rows, err := h.db.Query(ctx, `
		SELECT 
			TRIM(TO_CHAR(created_at, 'Dy')) AS label,
			COUNT(*) AS value
		FROM audit_log
		WHERE created_at >= NOW() - INTERVAL '7 days'
		GROUP BY TO_CHAR(created_at, 'Dy'), DATE_TRUNC('day', created_at)
		ORDER BY DATE_TRUNC('day', created_at) ASC;
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query weekly activity: " + err.Error()})
		return
	}
	defer rows.Close()

	queryMap := make(map[string]int)
	for rows.Next() {
		var label string
		var val int
		if err := rows.Scan(&label, &val); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan activity"})
			return
		}
		queryMap[label] = val
	}

	dayMapping := map[string]string{
		"Mon": "Sen",
		"Tue": "Sel",
		"Wed": "Rab",
		"Thu": "Kam",
		"Fri": "Jum",
		"Sat": "Sab",
		"Sun": "Min",
	}

	type activityItem struct {
		Label string `json:"label"`
		Value int    `json:"value"`
	}

	var list []activityItem
	now := time.Now()
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		engLabel := d.Format("Mon")
		indLabel, exists := dayMapping[engLabel]
		if !exists {
			indLabel = engLabel
		}
		val := queryMap[engLabel]
		list = append(list, activityItem{
			Label: indLabel,
			Value: val,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) GetFieldProgress(c *gin.Context) {
	ctx := c.Request.Context()

	var totalWarga, docsComplete, rtRwComplete, surveyedComplete int
	err := h.db.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN foto_ktp_url IS NOT NULL AND foto_ktp_url <> '' AND foto_kk_url IS NOT NULL AND foto_kk_url <> '' THEN 1 END),
			COUNT(CASE WHEN rt IS NOT NULL AND rt <> '' AND rw IS NOT NULL AND rw <> '' THEN 1 END),
			COUNT(CASE WHEN c1_value > 0 OR c2_value > 0 OR c10_value > 0 THEN 1 END)
		FROM warga
		WHERE deleted_at IS NULL
	`).Scan(&totalWarga, &docsComplete, &rtRwComplete, &surveyedComplete)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query field progress: " + err.Error()})
		return
	}

	var rtRwPct, docsPct, surveyedPct float64
	if totalWarga > 0 {
		rtRwPct = float64(rtRwComplete) / float64(totalWarga) * 100
		docsPct = float64(docsComplete) / float64(totalWarga) * 100
		surveyedPct = float64(surveyedComplete) / float64(totalWarga) * 100
	}

	type progressItem struct {
		Label string  `json:"label"`
		Value float64 `json:"value"`
	}

	list := []progressItem{
		{Label: "Lengkap data RT/RW", Value: rtRwPct},
		{Label: "Verifikasi dokumen", Value: docsPct},
		{Label: "Sinkronisasi lapangan", Value: surveyedPct},
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) ensureSAWCalculated(ctx context.Context, periodID string) error {
	if periodID == "" {
		return nil
	}

	// 1. Check if we already have records in hasil_saw for this period
	var count int
	err := h.db.QueryRow(ctx, "SELECT COUNT(*) FROM hasil_saw WHERE periode_id = $1", periodID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check hasil_saw: %w", err)
	}

	if count > 0 {
		return nil
	}

	// 2. Load period info
	var periodName string
	var periodKuota int
	var bobotID string
	err = h.db.QueryRow(ctx, "SELECT nama_periode, kuota, bobot_id FROM periode_bansos WHERE id = $1", periodID).Scan(&periodName, &periodKuota, &bobotID)
	if err != nil {
		return fmt.Errorf("failed to load active period: %w", err)
	}

	resolvedKuota := periodKuota
	if resolvedKuota <= 0 {
		resolvedKuota = 1
	}

	if bobotID == "" {
		err = h.db.QueryRow(ctx, "SELECT id FROM bobot_kriteria ORDER BY created_at DESC LIMIT 1").Scan(&bobotID)
		if err != nil {
			return fmt.Errorf("no bobot version found: %w", err)
		}
	}

	alternatifs, bobotSlice, isBenefit, err := h.prepareSAWData(ctx, bobotID)
	if err != nil {
		return err
	}

	if len(alternatifs) == 0 {
		return nil
	}

	// 5. Calculate SAW
	hasil := saw.HitungSAW(alternatifs, bobotSlice, resolvedKuota, isBenefit)

	// 6. Save results
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM hasil_saw WHERE periode_id = $1", periodID)
	if err != nil {
		return fmt.Errorf("failed to delete old hasil_saw: %w", err)
	}

	for _, res := range hasil {
		_, err = tx.Exec(ctx, `
			INSERT INTO hasil_saw (periode_id, warga_id, nilai_vi, ranking, status)
			VALUES ($1, $2, $3, $4, $5)
		`, periodID, res.AlternatifID, res.Vi, res.Ranking, res.Status)
		if err != nil {
			return fmt.Errorf("failed to insert hasil_saw for %s: %w", res.Nama, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
