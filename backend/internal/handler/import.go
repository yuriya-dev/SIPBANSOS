package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ImportValidationResult struct {
	Row     int    `json:"row"`
	NIK     string `json:"nik"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "Valid", "Error", "Duplikat"
	Detail  string `json:"detail"`
	Data    wargaRequest `json:"data,omitempty"`
}

type ImportSummary struct {
	Total     int `json:"total"`
	Valid     int `json:"valid"`
	Duplicate int `json:"duplicate"`
	Error     int `json:"error"`
}

func (h *Handler) ValidateImport(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".csv" && ext != ".xlsx" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file format. Only .csv and .xlsx allowed"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer f.Close()

	var records [][]string
	if ext == ".csv" {
		reader := csv.NewReader(f)
		records, err = reader.ReadAll()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV"})
			return
		}
	} else {
		xl, err := excelize.OpenReader(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read Excel"})
			return
		}
		sheets := xl.GetSheetList()
		if len(sheets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Excel file is empty"})
			return
		}
		records, err = xl.GetRows(sheets[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read Excel rows"})
			return
		}
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File does not contain any data rows"})
		return
	}

	var results []ImportValidationResult
	summary := ImportSummary{}

	// Skip header
	for i, row := range records[1:] {
		rowNum := i + 2
		summary.Total++

		if len(row) < 22 {
			results = append(results, ImportValidationResult{
				Row: rowNum, Status: "Error", Detail: "Incomplete columns",
			})
			summary.Error++
			continue
		}

		nik := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[2])

		req := wargaRequest{
			NIK: nik,
			NoKK: strings.TrimSpace(row[1]),
			NamaLengkap: name,
			TanggalLahir: strings.TrimSpace(row[3]),
			JenisKelamin: strings.TrimSpace(row[4]),
			Alamat: strings.TrimSpace(row[5]),
			RT: strings.TrimSpace(row[6]),
			RW: strings.TrimSpace(row[7]),
			NoHP: strings.TrimSpace(row[8]),
		}

		c1, err1 := strconv.ParseFloat(row[9], 64)
		c2, err2 := strconv.ParseFloat(row[10], 64)
		c3, err3 := strconv.ParseFloat(row[11], 64)
		c4, err4 := strconv.ParseFloat(row[12], 64)
		c5, err5 := strconv.ParseFloat(row[13], 64)
		c6, err6 := strconv.ParseFloat(row[14], 64)
		c7, err7 := strconv.ParseFloat(row[15], 64)
		c8, err8 := strconv.ParseFloat(row[16], 64)
		c9, err9 := strconv.ParseFloat(row[17], 64)
		c10, err10 := strconv.ParseFloat(row[18], 64)
		c11, err11 := strconv.ParseFloat(row[19], 64)
		c12, err12 := strconv.ParseFloat(row[20], 64)
		c13, err13 := strconv.ParseFloat(row[21], 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil || err8 != nil || err9 != nil || err10 != nil || err11 != nil || err12 != nil || err13 != nil {
			results = append(results, ImportValidationResult{
				Row: rowNum, NIK: nik, Name: name, Status: "Error", Detail: "Format nilai tidak valid",
			})
			summary.Error++
			continue
		}

		req.KriteriaValues = map[string]float64{
			"C1":  c1,
			"C2":  c2,
			"C3":  c3,
			"C4":  c4,
			"C5":  c5,
			"C6":  c6,
			"C7":  c7,
			"C8":  c8,
			"C9":  c9,
			"C10": c10,
			"C11": c11,
			"C12": c12,
			"C13": c13,
		}

		_, err = validateWargaRequest(req)
		if err != nil {
			results = append(results, ImportValidationResult{
				Row: rowNum, NIK: nik, Name: name, Status: "Error", Detail: err.Error(),
			})
			summary.Error++
			continue
		}

		// Check duplicate NIK (mock checking in memory could be added, here we just query DB)
		// For performance, we could load all NIKs first, but doing it one by one is simpler for now
		baseQuery := "SELECT id FROM warga WHERE nik = $1 AND deleted_at IS NULL"
		var existingID string
		err = h.db.QueryRow(c.Request.Context(), baseQuery, nik).Scan(&existingID)
		if err == nil && existingID != "" {
			results = append(results, ImportValidationResult{
				Row: rowNum, NIK: nik, Name: name, Status: "Duplikat", Detail: "NIK sudah terdaftar",
				Data: req,
			})
			summary.Duplicate++
			continue
		}

		results = append(results, ImportValidationResult{
			Row: rowNum, NIK: nik, Name: name, Status: "Valid", Detail: "-",
			Data: req,
		})
		summary.Valid++
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
		"preview": results,
	})
}

type ConfirmImportRequest struct {
	Data []wargaRequest `json:"data"`
}

func (h *Handler) ConfirmImport(c *gin.Context) {
	var req ConfirmImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	createdBy := ""
	if rawUserID, ok := c.Get("user_id"); ok {
		createdBy, _ = rawUserID.(string)
	}

	successCount := 0
	for _, w := range req.Data {
		dob, _ := validateWargaRequest(w)
		item := wargaToModel(w, dob, createdBy)
		item.IsActive = true
		_, err := h.warga.Create(c.Request.Context(), item)
		if err == nil {
			successCount++
		}
	}

	go func() {
		if err := h.RecalculateSAWForActivePeriod(context.Background()); err != nil {
			log.Printf("[SAW] Error recalculating on confirm import: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Berhasil mengimpor %d data warga", successCount),
		"imported": successCount,
	})
}
