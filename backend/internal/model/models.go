package model

import (
	"encoding/json"
	"strings"
	"time"
)

type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
	PasswordHash string `json:"-"`
	IsActive     bool   `json:"is_active"`
}

type Warga struct {
	ID                 string             `json:"id"`
	NIK                string             `json:"nik"`
	NoKK               string             `json:"no_kk"`
	NamaLengkap        string             `json:"nama_lengkap"`
	TanggalLahir       time.Time          `json:"tanggal_lahir"`
	JenisKelamin       string             `json:"jenis_kelamin"`
	Alamat             string             `json:"alamat"`
	RT                 *string            `json:"rt,omitempty"`
	RW                 *string            `json:"rw,omitempty"`
	NoHP               *string            `json:"no_hp,omitempty"`
	FotoKtpURL         *string            `json:"foto_ktp_url,omitempty"`
	FotoKKURL          *string            `json:"foto_kk_url,omitempty"`
	IsActive           bool               `json:"is_active"`
	CreatedBy          *string            `json:"created_by,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	KriteriaValues     map[string]float64 `json:"-"`
}

func (w Warga) MarshalJSON() ([]byte, error) {
	type Alias Warga
	b, err := json.Marshal((*Alias)(&w))
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	for k, v := range w.KriteriaValues {
		key := strings.ToLower(k) + "_value"
		m[key] = v
	}

	return json.Marshal(m)
}

type AuditLog struct {
	ID        int64           `json:"id"`
	UserID    string          `json:"user_id"`
	ActorName string          `json:"actor_name"`
	Aksi      string          `json:"aksi"`
	Tabel     string          `json:"tabel"`
	RecordID  string          `json:"record_id"`
	DataLama  json.RawMessage `json:"data_lama,omitempty"`
	DataBaru  json.RawMessage `json:"data_baru,omitempty"`
	IPAddress string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type KriteriaBobot struct {
	ID          string             `json:"id"`
	Versi       string             `json:"versi"`
	Keterangan  *string            `json:"keterangan,omitempty"`
	IsActive    bool               `json:"is_active"`
	DibuatOleh  *string            `json:"dibuat_oleh,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	BobotValues map[string]float64 `json:"-"`
}

func (kb KriteriaBobot) MarshalJSON() ([]byte, error) {
	type Alias KriteriaBobot
	b, err := json.Marshal((*Alias)(&kb))
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	for k, v := range kb.BobotValues {
		key := "bobot_" + strings.ToLower(k)
		m[key] = v
	}

	return json.Marshal(m)
}

type PeriodeBansos struct {
	ID             string    `json:"id"`
	NamaPeriode    string    `json:"nama_periode"`
	TanggalMulai   time.Time `json:"tanggal_mulai"`
	TanggalSelesai time.Time `json:"tanggal_selesai"`
	Kuota          int       `json:"kuota"`
	BobotID        *string   `json:"bobot_id,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type HasilSAWReport struct {
	ID          string  `json:"id"`
	PeriodeID   string  `json:"periode_id"`
	WargaID     string  `json:"warga_id"`
	NamaLengkap string  `json:"nama_lengkap"`
	NIK         string  `json:"nik"`
	RT          *string `json:"rt,omitempty"`
	RW          *string `json:"rw,omitempty"`
	NilaiVI     float64 `json:"nilai_vi"`
	Ranking     int     `json:"ranking"`
	Status      string  `json:"status"`
}

type HasilSAWSummary struct {
	Total      int     `json:"total"`
	Penerima   int     `json:"penerima"`
	Cadangan   int     `json:"cadangan"`
	TidakLolos int     `json:"tidak_lolos"`
	RataRata   float64 `json:"rata_rata"`
}

type Schedule struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	Date      time.Time `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

