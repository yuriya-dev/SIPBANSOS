CREATE TABLE IF NOT EXISTS kriteria (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(10) UNIQUE NOT NULL,
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(10) NOT NULL, -- 'Benefit' or 'Cost'
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Seed dengan kriteria awal (C1 - C13) jika belum ada
INSERT INTO kriteria (code, name, type, is_active) VALUES
('C1', 'Jumlah Anggota Keluarga', 'Benefit', TRUE),
('C2', 'Jumlah Tanggungan', 'Benefit', TRUE),
('C3', 'Pendidikan Kep. Keluarga', 'Cost', TRUE),
('C4', 'Pekerjaan Kep. Keluarga', 'Cost', TRUE),
('C5', 'Status Rumah', 'Benefit', TRUE),
('C6', 'Luas Rumah (m²)', 'Cost', TRUE),
('C7', 'Daya Listrik (VA)', 'Cost', TRUE),
('C8', 'Jumlah Kendaraan', 'Cost', TRUE),
('C9', 'Tabungan (Rupiah)', 'Cost', TRUE),
('C10', 'Penghasilan per Bulan (Rp)', 'Cost', TRUE),
('C11', 'Pengeluaran per Bulan (Rp)', 'Benefit', TRUE),
('C12', 'Kondisi Dinding', 'Cost', TRUE),
('C13', 'Akses Air', 'Cost', TRUE)
ON CONFLICT (code) DO NOTHING;
