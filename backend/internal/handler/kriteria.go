package handler

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wahyutricahya/SIPBANSOS/backend/internal/model"
	"github.com/wahyutricahya/SIPBANSOS/backend/internal/repository"
)

type kriteriaDefinition struct {
	Code string
	Name string
	Type string
}

type kriteriaItem struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Weight float64 `json:"weight"`
}

type kriteriaVersion struct {
	ID         string  `json:"id"`
	Versi      string  `json:"versi"`
	Keterangan *string `json:"keterangan,omitempty"`
	IsActive   bool    `json:"is_active"`
}

type kriteriaResponse struct {
	Version     *kriteriaVersion `json:"version,omitempty"`
	Criteria    []kriteriaItem   `json:"criteria"`
	TotalWeight float64          `json:"total_weight"`
}

type kriteriaUpdateRequest struct {
	Versi       string             `json:"versi"`
	Keterangan  *string            `json:"keterangan"`
	IsActive    *bool              `json:"is_active"`
	BobotValues map[string]float64 `json:"-"`
}

func (r *kriteriaUpdateRequest) UnmarshalJSON(data []byte) error {
	type Alias kriteriaUpdateRequest
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = kriteriaUpdateRequest(aux)

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.BobotValues = make(map[string]float64)
	for k, v := range raw {
		if strings.HasPrefix(k, "bobot_") {
			code := strings.ToUpper(strings.TrimPrefix(k, "bobot_"))
			if valFloat, ok := v.(float64); ok {
				r.BobotValues[code] = valFloat
			} else if valStr, ok := v.(string); ok {
				if f, err := strconv.ParseFloat(valStr, 64); err == nil {
					r.BobotValues[code] = f
				}
			}
		}
	}
	return nil
}

type kriteriaDefinitionRequest struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}

func (h *Handler) getActiveKriteriaDefinitions(ctx context.Context) ([]kriteriaDefinition, error) {
	rows, err := h.db.Query(ctx, `
		SELECT code, name, type
		FROM kriteria
		WHERE is_active = TRUE
		ORDER BY CAST(SUBSTRING(code FROM '[0-9]+') AS INTEGER) ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []kriteriaDefinition
	for rows.Next() {
		var item kriteriaDefinition
		if err := rows.Scan(&item.Code, &item.Name, &item.Type); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}

func (h *Handler) ListKriteria(c *gin.Context) {
	defs, err := h.getActiveKriteriaDefinitions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load criteria definitions: " + err.Error()})
		return
	}

	data, err := h.kriteria.GetActiveOrLatest(c.Request.Context())
	if err != nil {
		if errors.Is(err, repository.ErrKriteriaNotFound) {
			response := buildKriteriaResponse(nil, defs)
			c.JSON(http.StatusOK, response)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load kriteria weights"})
		return
	}

	response := buildKriteriaResponse(data, defs)
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateKriteria(c *gin.Context) {
	var req kriteriaUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var total float64
	for _, w := range req.BobotValues {
		total += w
	}
	if math.Abs(total-1) > 0.01 { // payload dikirim dalam desimal (0-1), pastikan total 1 (bukan 100)
		c.JSON(http.StatusBadRequest, gin.H{"error": "total bobot harus 1 (100%)"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	createdBy := ""
	if rawUserID, ok := c.Get("user_id"); ok {
		createdBy, _ = rawUserID.(string)
	}

	newItem := model.KriteriaBobot{
		Versi:       req.Versi,
		Keterangan:  req.Keterangan,
		IsActive:    isActive,
		DibuatOleh:  &createdBy,
		BobotValues: req.BobotValues,
	}

	saved, err := h.kriteria.Create(c.Request.Context(), newItem)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create kriteria: " + err.Error()})
		return
	}

	defs, err := h.getActiveKriteriaDefinitions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load criteria definitions: " + err.Error()})
		return
	}

	response := buildKriteriaResponse(saved, defs)
	c.JSON(http.StatusCreated, response)
}

func (h *Handler) UpdateKriteria(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing kriteria id"})
		return
	}

	var req kriteriaUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.kriteria.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrKriteriaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "kriteria not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load kriteria"})
		return
	}

	var total float64
	for _, w := range req.BobotValues {
		total += w
	}
	if math.Abs(total-1) > 0.01 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "total bobot harus 1 (100%)"})
		return
	}

	updated := *existing
	if strings.TrimSpace(req.Versi) != "" {
		updated.Versi = req.Versi
	}
	if req.Keterangan != nil {
		updated.Keterangan = req.Keterangan
	}
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}

	updated.BobotValues = req.BobotValues

	saved, err := h.kriteria.Update(c.Request.Context(), id, updated, updated.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrKriteriaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "kriteria not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update kriteria: " + err.Error()})
		return
	}

	defs, err := h.getActiveKriteriaDefinitions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load criteria definitions: " + err.Error()})
		return
	}

	response := buildKriteriaResponse(saved, defs)
	c.JSON(http.StatusOK, response)
}

func buildKriteriaResponse(bobot *model.KriteriaBobot, defs []kriteriaDefinition) kriteriaResponse {
	var version *kriteriaVersion
	bobotValues := make(map[string]float64)

	if bobot != nil {
		bobotValues = bobot.BobotValues
		version = &kriteriaVersion{
			ID:         bobot.ID,
			Versi:      bobot.Versi,
			Keterangan: bobot.Keterangan,
			IsActive:   bobot.IsActive,
		}
	}

	total := 0.0
	items := make([]kriteriaItem, 0, len(defs))
	for _, def := range defs {
		weight := bobotValues[def.Code]
		total += weight
		items = append(items, kriteriaItem{
			Code:   def.Code,
			Name:   def.Name,
			Type:   def.Type,
			Weight: weight,
		})
	}

	return kriteriaResponse{
		Version:     version,
		Criteria:    items,
		TotalWeight: total,
	}
}

func (h *Handler) ListKriteriaVersions(c *gin.Context) {
	rows, err := h.db.Query(c.Request.Context(), "SELECT id, versi, keterangan, is_active FROM bobot_kriteria ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list weight versions: " + err.Error()})
		return
	}
	defer rows.Close()

	type versionItem struct {
		ID         string  `json:"id"`
		Versi      string  `json:"versi"`
		Keterangan *string `json:"keterangan,omitempty"`
		IsActive   bool    `json:"is_active"`
	}

	var list []versionItem
	for rows.Next() {
		var item versionItem
		if err := rows.Scan(&item.ID, &item.Versi, &item.Keterangan, &item.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan weight version"})
			return
		}
		list = append(list, item)
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) GetKriteriaByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing kriteria id"})
		return
	}

	data, err := h.kriteria.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrKriteriaNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "kriteria not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load kriteria: " + err.Error()})
		return
	}

	defs, err := h.getActiveKriteriaDefinitions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load criteria definitions: " + err.Error()})
		return
	}

	response := buildKriteriaResponse(data, defs)
	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateKriteriaDefinition(c *gin.Context) {
	var req kriteriaDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Type = strings.Title(strings.ToLower(req.Type))
	if req.Type != "Benefit" && req.Type != "Cost" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe kriteria harus 'Benefit' atau 'Cost'"})
		return
	}

	// Dynamic calculation of next code (no limit)
	var maxNum int
	rows, err := h.db.Query(c.Request.Context(), "SELECT code FROM kriteria")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err == nil {
				if strings.HasPrefix(code, "C") {
					if num, err := strconv.Atoi(strings.TrimPrefix(code, "C")); err == nil {
						if num > maxNum {
							maxNum = num
						}
					}
				}
			}
		}
	}
	chosenCode := "C" + strconv.Itoa(maxNum + 1)

	_, err = h.db.Exec(c.Request.Context(), `
		INSERT INTO kriteria (code, name, type, is_active, updated_at)
		VALUES ($1, $2, $3, TRUE, NOW())
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name, type = EXCLUDED.type, is_active = TRUE, updated_at = NOW()
	`, chosenCode, req.Name, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan kriteria baru: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Kriteria baru berhasil ditambahkan", "code": chosenCode})
}

func (h *Handler) UpdateKriteriaDefinition(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing kriteria ID"})
		return
	}

	var req kriteriaDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Type = strings.Title(strings.ToLower(req.Type))
	if req.Type != "Benefit" && req.Type != "Cost" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe kriteria harus 'Benefit' atau 'Cost'"})
		return
	}

	var code string
	err := h.db.QueryRow(c.Request.Context(), `
		UPDATE kriteria
		SET name = $1, type = $2, updated_at = NOW()
		WHERE id::text = $3 OR code = $3
		RETURNING code
	`, req.Name, req.Type, id).Scan(&code)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kriteria: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kriteria berhasil diperbarui", "code": code})
}

func (h *Handler) DeleteKriteriaDefinition(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing kriteria ID"})
		return
	}

	var code string
	err := h.db.QueryRow(c.Request.Context(), `
		UPDATE kriteria
		SET is_active = FALSE, updated_at = NOW()
		WHERE id::text = $1 OR code = $1
		RETURNING code
	`, id).Scan(&code)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kriteria: " + err.Error()})
		return
	}

	// Update bobot_values to set the deleted criteria weight to 0 in all versions
	_, _ = h.db.Exec(c.Request.Context(), `
		UPDATE bobot_kriteria
		SET bobot_values = jsonb_set(bobot_values, ARRAY[$1], '0'::jsonb)
	`, code)

	c.JSON(http.StatusOK, gin.H{"message": "Kriteria berhasil dihapus", "code": code})
}
