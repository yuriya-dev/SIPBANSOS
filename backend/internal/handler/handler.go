package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/wahyutricahya/SIPBANSOS/backend/internal/auth"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/model"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/repository"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/saw"
)

type Handler struct {
	db       *pgxpool.Pool
	auth     *auth.Manager
	users    *repository.UserRepository
	warga    *repository.WargaRepository
	audit    *repository.AuditRepository
	kriteria *repository.KriteriaRepository
	reports  *repository.ReportRepository
}

func NewHandler(db *pgxpool.Pool, authManager *auth.Manager) *Handler {
	return &Handler{
		db:       db,
		auth:     authManager,
		users:    repository.NewUserRepository(db),
		warga:    repository.NewWargaRepository(db),
		audit:    repository.NewAuditRepository(db),
		kriteria: repository.NewKriteriaRepository(db),
		reports:  repository.NewReportRepository(db),
	}
}

func (h *Handler) prepareSAWData(ctx context.Context, bobotID string) ([]saw.Alternatif, []float64, []bool, error) {
	// 1. Fetch active criteria
	defs, err := h.getActiveKriteriaDefinitions(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load criteria definitions: %w", err)
	}
	if len(defs) == 0 {
		return nil, nil, nil, errors.New("no active criteria configured")
	}

	// 2. Fetch weights for the selected version
	var bobotValues map[string]float64
	var rawBobot []byte
	err = h.db.QueryRow(ctx, "SELECT bobot_values FROM bobot_kriteria WHERE id = $1", bobotID).Scan(&rawBobot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load criteria weights: %w", err)
	}
	if len(rawBobot) > 0 {
		_ = json.Unmarshal(rawBobot, &bobotValues)
	}
	if bobotValues == nil {
		bobotValues = make(map[string]float64)
	}

	// Build bobot slice and isBenefit slice aligned with defs
	bobots := make([]float64, len(defs))
	isBenefit := make([]bool, len(defs))
	for idx, def := range defs {
		bobots[idx] = bobotValues[def.Code]
		isBenefit[idx] = strings.ToLower(def.Type) == "benefit"
	}

	// 3. Fetch citizens and their dynamic values
	rows, err := h.db.Query(ctx, `
		SELECT id, nama_lengkap, kriteria_values
		FROM warga
		WHERE deleted_at IS NULL AND is_active = true AND foto_ktp_url IS NOT NULL AND foto_ktp_url <> '' AND foto_kk_url IS NOT NULL AND foto_kk_url <> ''
	`)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load warga: %w", err)
	}
	defer rows.Close()

	var alternatifs []saw.Alternatif
	for rows.Next() {
		var a saw.Alternatif
		var rawVals []byte
		err := rows.Scan(&a.ID, &a.Nama, &rawVals)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to scan warga: %w", err)
		}

		var vals map[string]float64
		if len(rawVals) > 0 {
			_ = json.Unmarshal(rawVals, &vals)
		}
		if vals == nil {
			vals = make(map[string]float64)
		}

		// Aligned values with defs
		a.Nilai = make([]float64, len(defs))
		for idx, def := range defs {
			a.Nilai[idx] = vals[def.Code]
		}
		alternatifs = append(alternatifs, a)
	}

	return alternatifs, bobots, isBenefit, nil
}

type loginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type wargaRequest struct {
	NIK            string             `json:"nik" binding:"required"`
	NoKK           string             `json:"no_kk" binding:"required"`
	NamaLengkap    string             `json:"nama_lengkap" binding:"required"`
	TanggalLahir   string             `json:"tanggal_lahir" binding:"required"`
	JenisKelamin   string             `json:"jenis_kelamin" binding:"required"`
	Alamat         string             `json:"alamat" binding:"required"`
	RT             string             `json:"rt"`
	RW             string             `json:"rw"`
	NoHP           string             `json:"no_hp"`
	FotoKtpURL     string             `json:"foto_ktp_url"`
	FotoKKURL      string             `json:"foto_kk_url"`
	KriteriaValues map[string]float64 `json:"-"`
}

func (r *wargaRequest) UnmarshalJSON(data []byte) error {
	type Alias wargaRequest
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = wargaRequest(aux)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.KriteriaValues = make(map[string]float64)
	for k, v := range raw {
		if strings.HasSuffix(k, "_value") {
			code := strings.ToUpper(strings.TrimSuffix(k, "_value"))
			if valFloat, ok := v.(float64); ok {
				r.KriteriaValues[code] = valFloat
			} else if valStr, ok := v.(string); ok {
				if f, err := strconv.ParseFloat(valStr, 64); err == nil {
					r.KriteriaValues[code] = f
				}
			}
		}
	}
	return nil
}

type sawRunRequest struct {
	Kuota     int    `json:"kuota"`
	PeriodeID string `json:"periode_id"`
	BobotID   string `json:"bobot_id"`
}

func (h *Handler) DebugDB(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.db.Ping(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "ping failed: " + err.Error(),
		})
		return
	}

	var one int
	err = h.db.QueryRow(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "query_error",
			"error":  "query failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"db":     "connected",
	})
}

// Auth handlers
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByIdentifier(c.Request.Context(), req.Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	access, refresh, err := h.auth.GenerateTokens(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	_ = h.users.UpdateLastLogin(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    int(h.auth.AccessTTL().Seconds()),
		"user":          user,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := h.auth.ParseToken(req.RefreshToken, "refresh")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		return
	}

	access, refresh, err := h.auth.GenerateTokens(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    int(h.auth.AccessTTL().Seconds()),
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user context"})
		return
	}

	user, err := h.users.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Warga CRUD
func (h *Handler) ListWarga(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 10)
	filter := repository.WargaFilter{
		Page:  page,
		Limit: limit,
		Query: c.Query("q"),
		RT:    c.Query("rt"),
		RW:    c.Query("rw"),
	}

	data, stats, err := h.warga.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list warga"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  data,
		"page":  page,
		"limit": limit,
		"total": stats.Total,
		"stats": stats,
	})
}

func (h *Handler) CreateWarga(c *gin.Context) {
	var req wargaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dob, err := validateWargaRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdBy := ""
	if rawUserID, ok := c.Get("user_id"); ok {
		createdBy, _ = rawUserID.(string)
	}

	item := wargaToModel(req, dob, createdBy)
	item.IsActive = true

	created, err := h.warga.Create(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create warga"})
		return
	}

	if userID, ok := c.Get("user_id"); ok {
		_ = h.audit.Create(c.Request.Context(), model.AuditLog{
			UserID:    userID.(string),
			Aksi:      "create",
			Tabel:     "warga",
			RecordID:  created.ID,
			DataBaru:  mustJSON(created),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
	}

	go func() {
		if err := h.RecalculateSAWForActivePeriod(context.Background()); err != nil {
			log.Printf("[SAW] Error recalculating on warga create: %v", err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *Handler) GetWarga(c *gin.Context) {
	id := c.Param("id")
	item, err := h.warga.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrWargaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warga"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *Handler) UpdateWarga(c *gin.Context) {
	id := c.Param("id")
	previous, err := h.warga.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrWargaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warga"})
		return
	}
	var req wargaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dob, err := validateWargaRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item := wargaToModel(req, dob, "")
	updated, err := h.warga.Update(c.Request.Context(), id, item)
	if err != nil {
		if errors.Is(err, repository.ErrWargaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update warga"})
		return
	}

	if userID, ok := c.Get("user_id"); ok {
		_ = h.audit.Create(c.Request.Context(), model.AuditLog{
			UserID:    userID.(string),
			Aksi:      "update",
			Tabel:     "warga",
			RecordID:  updated.ID,
			DataLama:  mustJSON(previous),
			DataBaru:  mustJSON(updated),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		})
	}

	go func() {
		if err := h.RecalculateSAWForActivePeriod(context.Background()); err != nil {
			log.Printf("[SAW] Error recalculating on warga update: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *Handler) GetWargaHistory(c *gin.Context) {
	id := c.Param("id")
	items, err := h.audit.ListByRecord(c.Request.Context(), "warga", id, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) DeleteWarga(c *gin.Context) {
	id := c.Param("id")
	if err := h.warga.SoftDelete(c.Request.Context(), id); err != nil {
		if errors.Is(err, repository.ErrWargaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warga not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete warga"})
		return
	}

	go func() {
		if err := h.RecalculateSAWForActivePeriod(context.Background()); err != nil {
			log.Printf("[SAW] Error recalculating on warga delete: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// SAW execution endpoint
func (h *Handler) RunSAW(c *gin.Context) {
	var req sawRunRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.PeriodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periode_id wajib diisi"})
		return
	}

	ctx := c.Request.Context()

	var periodName string
	var periodKuota int
	var bobotID string
	err := h.db.QueryRow(ctx, "SELECT nama_periode, kuota, bobot_id FROM periode_bansos WHERE id = $1", req.PeriodeID).Scan(&periodName, &periodKuota, &bobotID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Periode bansos tidak ditemukan"})
		return
	}

	resolvedKuota := req.Kuota
	if resolvedKuota <= 0 {
		resolvedKuota = periodKuota
	}
	if resolvedKuota <= 0 {
		resolvedKuota = 1
	}

	resolvedBobotID := req.BobotID
	if resolvedBobotID == "" {
		resolvedBobotID = bobotID
	}
	if resolvedBobotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "versi bobot wajib diisi atau ditentukan di periode"})
		return
	}

	alternatifs, bobot, isBenefit, err := h.prepareSAWData(ctx, resolvedBobotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(alternatifs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data warga terverifikasi (dokumen KTP & KK lengkap) untuk dihitung"})
		return
	}

	hasil := saw.HitungSAW(alternatifs, bobot, resolvedKuota, isBenefit)

	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM hasil_saw WHERE periode_id = $1", req.PeriodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan data perhitungan lama"})
		return
	}

	for _, res := range hasil {
		_, err = tx.Exec(ctx, `
			INSERT INTO hasil_saw (periode_id, warga_id, nilai_vi, ranking, status)
			VALUES ($1, $2, $3, $4, $5)
		`, req.PeriodeID, res.AlternatifID, res.Vi, res.Ranking, res.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan hasil perhitungan warga " + res.Nama + ": " + err.Error()})
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal meresmikan penyimpanan hasil transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"hasil": hasil, "kuota": resolvedKuota})
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	value := strings.TrimSpace(c.DefaultQuery(key, ""))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func validateWargaRequest(req wargaRequest) (time.Time, error) {
	if len(req.NIK) != 16 || !isDigits(req.NIK) {
		return time.Time{}, errors.New("nik harus 16 digit")
	}
	if len(req.NoKK) != 16 || !isDigits(req.NoKK) {
		return time.Time{}, errors.New("no_kk harus 16 digit")
	}
	if req.JenisKelamin != "L" && req.JenisKelamin != "P" {
		return time.Time{}, errors.New("jenis_kelamin harus L atau P")
	}
	for _, val := range req.KriteriaValues {
		if val < 0 {
			return time.Time{}, errors.New("nilai kriteria tidak boleh negatif")
		}
	}

	dob, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		return time.Time{}, errors.New("tanggal_lahir harus format YYYY-MM-DD")
	}

	return dob, nil
}

func between(value int, min int, max int) bool {
	return value >= min && value <= max
}

func betweenFloat(value float64, min float64, max float64) bool {
	return value >= min && value <= max
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func wargaToModel(req wargaRequest, dob time.Time, createdBy string) model.Warga {
	return model.Warga{
		NIK:            req.NIK,
		NoKK:           req.NoKK,
		NamaLengkap:    req.NamaLengkap,
		TanggalLahir:   dob,
		JenisKelamin:   req.JenisKelamin,
		Alamat:         req.Alamat,
		RT:             stringPointer(req.RT),
		RW:             stringPointer(req.RW),
		NoHP:           stringPointer(req.NoHP),
		FotoKtpURL:     stringPointer(req.FotoKtpURL),
		FotoKKURL:      stringPointer(req.FotoKKURL),
		CreatedBy:      stringPointer(createdBy),
		KriteriaValues: req.KriteriaValues,
	}
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return data
}

func (h *Handler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal menerima file: " + err.Error()})
		return
	}

	ext := strings.ToLower(file.Filename)
	if !strings.HasSuffix(ext, ".jpg") && !strings.HasSuffix(ext, ".jpeg") && !strings.HasSuffix(ext, ".png") && !strings.HasSuffix(ext, ".webp") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung. Gunakan JPG, JPEG, PNG, atau WEBP."})
		return
	}

	if file.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 5MB."})
		return
	}

	// On Vercel the root filesystem is read-only; use /tmp which is writable.
	// Note: /tmp is ephemeral and not shared across instances — for persistent
	// file storage, integrate with Supabase Storage or similar cloud storage.
	uploadDir := "./uploads"
	if os.Getenv("VERCEL") != "" {
		uploadDir = "/tmp/uploads"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat direktori upload: " + err.Error()})
		return
	}

	filename := strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + file.Filename
	dst := uploadDir + "/" + filename

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file: " + err.Error()})
		return
	}

	urlPath := "/api/v1/uploads/" + filename
	c.JSON(http.StatusOK, gin.H{
		"url": urlPath,
	})
}

// Background auto calculation helper
func (h *Handler) RecalculateSAWForActivePeriod(ctx context.Context) error {
	var periodID string
	var periodKuota int
	var bobotID string
	err := h.db.QueryRow(ctx, "SELECT id, kuota, bobot_id FROM periode_bansos WHERE status = 'aktif' LIMIT 1").Scan(&periodID, &periodKuota, &bobotID)
	if err != nil {
		// If no active period is set, bypass calculation
		return nil
	}

	if bobotID == "" {
		return errors.New("active period is missing bobot_id configuration")
	}

	alternatifs, bobot, isBenefit, err := h.prepareSAWData(ctx, bobotID)
	if err != nil {
		return err
	}

	if len(alternatifs) == 0 {
		// Clear old results if there are no citizens
		_, err = h.db.Exec(ctx, "DELETE FROM hasil_saw WHERE periode_id = $1", periodID)
		return err
	}

	hasil := saw.HitungSAW(alternatifs, bobot, periodKuota, isBenefit)

	tx, err := h.db.Begin(ctx)
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
