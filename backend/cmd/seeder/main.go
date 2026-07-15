package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/wahyutricahya/SIPBANSOS/backend/internal/saw"
)

// Mapping functions for criteria
func mapC3(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	switch {
	case strings.Contains(val, "tidak sekolah") || strings.Contains(val, "sd"):
		return 1
	case strings.Contains(val, "smp"):
		return 2
	case strings.Contains(val, "sma"):
		return 3
	case strings.Contains(val, "diploma"):
		return 4
	case strings.Contains(val, "sarjana"):
		return 5
	default:
		return 1
	}
}

func mapC4(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	switch {
	case strings.Contains(val, "tidak tetap") || strings.Contains(val, "tidak bekerja") || strings.Contains(val, "pengangguran"):
		return 1
	case strings.Contains(val, "serabutan") || strings.Contains(val, "buruh harian"):
		return 2
	case strings.Contains(val, "petani") || strings.Contains(val, "nelayan"):
		return 3
	case strings.Contains(val, "karyawan") || strings.Contains(val, "pedagang") || strings.Contains(val, "wiraswasta"):
		return 4
	case strings.Contains(val, "pns") || strings.Contains(val, "tni") || strings.Contains(val, "polri") || strings.Contains(val, "bumn"):
		return 5
	default:
		return 1
	}
}

func mapC5(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	switch {
	case strings.Contains(val, "sendiri"):
		return 1
	case strings.Contains(val, "sewa") || strings.Contains(val, "kontrak"):
		return 2
	case strings.Contains(val, "numpang") || strings.Contains(val, "menumpang") || strings.Contains(val, "keluarga") || strings.Contains(val, "dinas"):
		return 3
	default:
		return 1
	}
}

func mapC12(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	switch {
	case strings.Contains(val, "bambu") || strings.Contains(val, "kayu"):
		return 1
	case strings.Contains(val, "semi permanen") || strings.Contains(val, "papan"):
		return 2
	case strings.Contains(val, "batako") || strings.Contains(val, "tanpa plester"):
		return 3
	case strings.Contains(val, "tembok") || strings.Contains(val, "permanen"):
		return 4
	default:
		return 4
	}
}

func mapC13(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	switch {
	case strings.Contains(val, "sungai") || strings.Contains(val, "hujan") || strings.Contains(val, "terbuka"):
		return 1
	case strings.Contains(val, "sumur") || strings.Contains(val, "gali") || strings.Contains(val, "bersama"):
		return 2
	case strings.Contains(val, "pdam") || strings.Contains(val, "bor") || strings.Contains(val, "air desa") || strings.Contains(val, "pribadi"):
		return 3
	default:
		return 3
	}
}

func parseFloat(val string) float64 {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0.0
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0.0
	}
	return parsed
}

// resolveToIPv4 parses the DSN, resolves the hostname to an IPv4 address,
// and returns a new DSN with the IPv4 address substituted in.
func resolveToIPv4(dsn string) string {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return dsn
	}

	host := config.ConnConfig.Host
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil || len(addrs) == 0 {
		log.Printf("warning: could not resolve %s to IPv4, using original DSN: %v", host, err)
		return dsn
	}

	ipv4 := addrs[0].String()
	log.Printf("resolved %s → %s (IPv4)", host, ipv4)
	return strings.Replace(dsn, host, ipv4, 1)
}

type CSVRecord struct {
	IdKeluarga             string
	JumlahAnggotaKeluarga  float64 // C1
	JumlahTanggungan       float64 // C2
	PendidikanKepKeluarga  float64 // C3
	PekerjaanKepKeluarga   float64 // C4
	StatusRumah            float64 // C5
	LuasRumah              float64 // C6
	DayaListrik            float64 // C7
	JumlahKendaraan        float64 // C8
	Tabungan               float64 // C9
	PenghasilanPerBulan    float64 // C10
	PengeluaranPerBulan    float64 // C11
	KondisiDinding         float64 // C12
	AksesAir               float64 // C13
}

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("missing DATABASE_URL")
	}

	// Resolve Supabase host issues on Mac
	resolvedDSN := resolveToIPv4(dsn)

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(resolvedDSN)
	if err != nil {
		log.Fatalf("failed to parse dsn: %v", err)
	}
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer pool.Close()

	// Parse CSV
	csvFile, err := os.Open("../Bantuan Sosial.csv")
	if err != nil {
		log.Fatalf("failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	// Read header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("failed to read CSV header: %v", err)
	}
	fmt.Printf("CSV Header: %v\n", header)

	var records []CSVRecord
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read CSV row: %v", err)
		}

		if len(row) < 14 {
			log.Printf("skipping incomplete row: %v", row)
			continue
		}

		rec := CSVRecord{
			IdKeluarga:            strings.TrimSpace(row[0]),
			JumlahAnggotaKeluarga: parseFloat(row[1]),
			JumlahTanggungan:      parseFloat(row[2]),
			PendidikanKepKeluarga: mapC3(row[3]),
			PekerjaanKepKeluarga:  mapC4(row[4]),
			StatusRumah:           mapC5(row[5]),
			LuasRumah:             parseFloat(row[6]),
			DayaListrik:           parseFloat(row[7]),
			JumlahKendaraan:       parseFloat(row[8]),
			Tabungan:              parseFloat(row[9]) * 1000000.0,
			PenghasilanPerBulan:   parseFloat(row[10]) * 1000000.0,
			PengeluaranPerBulan:   parseFloat(row[11]) * 1000000.0,
			KondisiDinding:        mapC12(row[12]),
			AksesAir:              mapC13(row[13]),
		}
		records = append(records, rec)
	}
	fmt.Printf("Parsed %d records from CSV.\n", len(records))

	// Truncate tables
	fmt.Println("Truncating tables (hasil_saw, warga, audit_log)...")
	_, err = pool.Exec(ctx, "TRUNCATE TABLE hasil_saw CASCADE")
	if err != nil {
		log.Fatalf("truncate hasil_saw failed: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE TABLE warga CASCADE")
	if err != nil {
		log.Fatalf("truncate warga failed: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE TABLE audit_log CASCADE")
	if err != nil {
		log.Fatalf("truncate audit_log failed: %v", err)
	}

	// Calculate 6 active dates in the last 7 calendar days (excluding Sunday)
	var activeDates []time.Time
	now := time.Now()
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -i)
		// Set to 10:00 AM local time on that day
		localDay := time.Date(d.Year(), d.Month(), d.Day(), 10, 0, 0, 0, time.Local)
		if localDay.Weekday() != time.Sunday {
			activeDates = append(activeDates, localDay)
		}
	}
	// Sort chronological
	sort.Slice(activeDates, func(i, j int) bool {
		return activeDates[i].Before(activeDates[j])
	})

	fmt.Println("Active dates for weekly activity distribution (excluding Sundays):")
	for i, d := range activeDates {
		fmt.Printf("- Day %d: %s (%s)\n", i+1, d.Format("2006-01-02 15:04:05"), d.Weekday().String())
	}

	// User IDs
	petugasID := "868a2fc1-8000-4aa9-827d-715c9197c849" // Petugas Survei (inputs data)
	operatorID := "4f0b9166-b697-48a4-81d7-12a7d8c11cbc" // Operator RW/RT (verifies data)

	// Batch Insert Citizens and Audit Logs
	fmt.Println("Inserting records...")
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to start transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	for i, rec := range records {
		idx := i % len(activeDates)
		createdDate := activeDates[idx]
		updatedDate := createdDate.Add(30 * time.Minute) // verified 30 minutes later

		nik := fmt.Sprintf("320101%010d", i+1)
		noKK := fmt.Sprintf("320102%010d", i+1)
		namaLengkap := fmt.Sprintf("Kepala Keluarga %s", rec.IdKeluarga)
		alamat := fmt.Sprintf("Jl. Mekar Jaya No. %d", i+1)
		rt := fmt.Sprintf("%03d", (i%5)+1) // RT 001 to 005
		rw := "001"
		noHP := fmt.Sprintf("08123456%04d", i+1)
		fotoKtp := "/api/v1/uploads/dummy_ktp.jpg"
		fotoKK := "/api/v1/uploads/dummy_kk.webp"

		var citizenID string
		err = tx.QueryRow(ctx, `
			INSERT INTO warga (
				nik, no_kk, nama_lengkap, tanggal_lahir, jenis_kelamin, alamat, rt, rw, no_hp,
				foto_ktp_url, foto_kk_url,
				c1_value, c2_value, c3_value, c4_value, c5_value, c6_value, c7_value, c8_value,
				c9_value, c10_value, c11_value, c12_value, c13_value,
				is_active, created_by, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
			RETURNING id
		`,
			nik, noKK, namaLengkap, time.Date(1985, 5, 15, 0, 0, 0, 0, time.Local), "L", alamat, rt, rw, noHP,
			fotoKtp, fotoKK,
			rec.JumlahAnggotaKeluarga, rec.JumlahTanggungan, rec.PendidikanKepKeluarga, rec.PekerjaanKepKeluarga, rec.StatusRumah,
			rec.LuasRumah, rec.DayaListrik, rec.JumlahKendaraan, rec.Tabungan, rec.PenghasilanPerBulan, rec.PengeluaranPerBulan,
			rec.KondisiDinding, rec.AksesAir,
			true, petugasID, createdDate, updatedDate,
		).Scan(&citizenID)

		if err != nil {
			log.Fatalf("failed to insert warga %s: %v", rec.IdKeluarga, err)
		}

		// Helper to marshal JSON
		dataBaruJSON, _ := json.Marshal(map[string]any{
			"id":            citizenID,
			"nik":           nik,
			"no_kk":         noKK,
			"nama_lengkap":  namaLengkap,
			"c1_value":      rec.JumlahAnggotaKeluarga,
			"c2_value":      rec.JumlahTanggungan,
			"c3_value":      rec.PendidikanKepKeluarga,
			"c4_value":      rec.PekerjaanKepKeluarga,
			"c5_value":      rec.StatusRumah,
			"c6_value":      rec.LuasRumah,
			"c7_value":      rec.DayaListrik,
			"c8_value":      rec.JumlahKendaraan,
			"c9_value":      rec.Tabungan,
			"c10_value":     rec.PenghasilanPerBulan,
			"c11_value":     rec.PengeluaranPerBulan,
			"c12_value":     rec.KondisiDinding,
			"c13_value":     rec.AksesAir,
			"is_active":     true,
			"foto_ktp_url":  fotoKtp,
			"foto_kk_url":   fotoKK,
		})

		// 1. Audit Log: Create (Input Warga)
		_, err = tx.Exec(ctx, `
			INSERT INTO audit_log (
				user_id, aksi, tabel, record_id, data_baru, ip_address, user_agent, created_at
			) VALUES ($1, $2, $3, $4, $5::jsonb, '127.0.0.1'::inet, 'Go-Seeder', $6)
		`, petugasID, "create", "warga", citizenID, string(dataBaruJSON), createdDate)
		if err != nil {
			log.Fatalf("failed to insert create audit log: %v", err)
		}

		// 2. Audit Log: Update (Verification)
		_, err = tx.Exec(ctx, `
			INSERT INTO audit_log (
				user_id, aksi, tabel, record_id, data_lama, data_baru, ip_address, user_agent, created_at
			) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, '127.0.0.1'::inet, 'Go-Seeder', $7)
		`, operatorID, "update", "warga", citizenID, string(dataBaruJSON), string(dataBaruJSON), updatedDate)
		if err != nil {
			log.Fatalf("failed to insert verify audit log: %v", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Fatalf("transaction commit failed: %v", err)
	}
	fmt.Printf("Successfully inserted %d citizens and %d audit logs across 6 days.\n", len(records), len(records)*2)

	// Recalculate SAW for active period
	fmt.Println("Recalculating SAW...")
	if err := recalculateSAW(ctx, pool); err != nil {
		log.Fatalf("SAW recalculation failed: %v", err)
	}
	fmt.Println("SAW Recalculated successfully!")
}

func recalculateSAW(ctx context.Context, db *pgxpool.Pool) error {
	var periodID string
	var periodKuota int
	var bobotID string
	err := db.QueryRow(ctx, "SELECT id, kuota, bobot_id FROM periode_bansos WHERE status = 'aktif' LIMIT 1").Scan(&periodID, &periodKuota, &bobotID)
	if err != nil {
		// If no active period is set, bypass calculation
		log.Println("No active period found. Skipping SAW calculation.")
		return nil
	}

	if bobotID == "" {
		return errors.New("active period is missing bobot_id configuration")
	}

	var b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13 float64
	err = db.QueryRow(ctx, `
		SELECT bobot_c1, bobot_c2, bobot_c3, bobot_c4, bobot_c5, bobot_c6, bobot_c7, bobot_c8, bobot_c9, bobot_c10, bobot_c11, bobot_c12, bobot_c13
		FROM bobot_kriteria WHERE id = $1
	`, bobotID).Scan(&b1, &b2, &b3, &b4, &b5, &b6, &b7, &b8, &b9, &b10, &b11, &b12, &b13)
	if err != nil {
		return err
	}
	bobot := [13]float64{b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13}

	rows, err := db.Query(ctx, `
		SELECT id, nama_lengkap, c1_value, c2_value, c3_value, c4_value, c5_value, c6_value, c7_value, c8_value, c9_value, c10_value, c11_value, c12_value, c13_value
		FROM warga
		WHERE deleted_at IS NULL AND is_active = true AND foto_ktp_url IS NOT NULL AND foto_ktp_url <> '' AND foto_kk_url IS NOT NULL AND foto_kk_url <> ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var alternatifs []saw.Alternatif
	for rows.Next() {
		var a saw.Alternatif
		var c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12, c13 float64
		err := rows.Scan(&a.ID, &a.Nama, &c1, &c2, &c3, &c4, &c5, &c6, &c7, &c8, &c9, &c10, &c11, &c12, &c13)
		if err != nil {
			return err
		}
		a.Nilai = [13]float64{c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12, c13}
		alternatifs = append(alternatifs, a)
	}

	if len(alternatifs) == 0 {
		_, err = db.Exec(ctx, "DELETE FROM hasil_saw WHERE periode_id = $1", periodID)
		return err
	}

	isBenefit := [13]bool{true, true, false, false, true, false, false, false, false, false, true, false, false}
	kRows, err := db.Query(ctx, "SELECT code, type FROM kriteria WHERE is_active = TRUE")
	if err == nil {
		defer kRows.Close()
		for kRows.Next() {
			var code, kType string
			if err := kRows.Scan(&code, &kType); err == nil {
				var index int
				if _, err := fmt.Sscanf(code, "C%d", &index); err == nil {
					index--
					if index >= 0 && index < 13 {
						isBenefit[index] = strings.ToLower(kType) == "benefit"
					}
				}
			}
		}
	}

	hasil := saw.HitungSAW(alternatifs, bobot, periodKuota, isBenefit)

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM hasil_saw WHERE periode_id = $1", periodID)
	if err != nil {
		return err
	}

	for _, res := range hasil {
		_, err = tx.Exec(ctx, `
			INSERT INTO hasil_saw (periode_id, warga_id, nilai_vi, ranking, status)
			VALUES ($1, $2, $3, $4, $5)
		`, periodID, res.AlternatifID, res.Vi, res.Ranking, res.Status)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
