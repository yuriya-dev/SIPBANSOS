package saw

import (
	"sort"
)

type Alternatif struct {
	ID    string
	Nama  string
	Nilai []float64
}

type HasilSAW struct {
	AlternatifID string  `json:"id"`
	Nama         string  `json:"nama"`
	Vi           float64 `json:"vi"`
	Ranking      int     `json:"ranking"`
	Status       string  `json:"status"`
}

func HitungSAW(alternatifs []Alternatif, bobot []float64, kuota int, isBenefit []bool) []HasilSAW {
	m := len(alternatifs)
	if m == 0 {
		return nil
	}

	numCriteria := len(bobot)
	if numCriteria == 0 {
		return nil
	}

	// build column max/min
	maxv := make([]float64, numCriteria)
	minv := make([]float64, numCriteria)
	for j := 0; j < numCriteria; j++ {
		maxv[j] = alternatifs[0].Nilai[j]
		minv[j] = alternatifs[0].Nilai[j]
	}
	for i := 0; i < m; i++ {
		for j := 0; j < numCriteria; j++ {
			// safety check to prevent panic if array sizes differ
			if j >= len(alternatifs[i].Nilai) {
				continue
			}
			v := alternatifs[i].Nilai[j]
			if v > maxv[j] {
				maxv[j] = v
			}
			if v < minv[j] {
				minv[j] = v
			}
		}
	}

	// normalize
	norm := make([][]float64, m)
	for i := 0; i < m; i++ {
		norm[i] = make([]float64, numCriteria)
		for j := 0; j < numCriteria; j++ {
			var isBen bool
			if j < len(isBenefit) {
				isBen = isBenefit[j]
			}
			var val float64
			if j < len(alternatifs[i].Nilai) {
				val = alternatifs[i].Nilai[j]
			}

			if isBen {
				if maxv[j] == 0 {
					norm[i][j] = 0
				} else {
					norm[i][j] = val / maxv[j]
				}
			} else {
				if val == 0 {
					norm[i][j] = 0
				} else {
					norm[i][j] = minv[j] / val
				}
			}
		}
	}

	hasil := make([]HasilSAW, m)
	for i := 0; i < m; i++ {
		var vi float64
		for j := 0; j < numCriteria; j++ {
			vi += bobot[j] * norm[i][j]
		}
		hasil[i] = HasilSAW{AlternatifID: alternatifs[i].ID, Nama: alternatifs[i].Nama, Vi: vi}
	}

	sort.Slice(hasil, func(i, j int) bool { return hasil[i].Vi > hasil[j].Vi })

	for i := range hasil {
		hasil[i].Ranking = i + 1
		switch {
		case i < kuota:
			hasil[i].Status = "Penerima"
		case i < kuota+5:
			hasil[i].Status = "Cadangan"
		default:
			hasil[i].Status = "Tidak Lolos"
		}
	}

	return hasil
}
