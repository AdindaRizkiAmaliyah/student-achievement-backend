package utils

// APIResponse adalah format standar JSON yang akan diterima Frontend.
// Contoh sukses  : { "status": true,  "message": "Login berhasil", "data": { ... } }
// Contoh gagal   : { "status": false, "message": "Gagal login",     "errors": "invalid credentials" }
type APIResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`   // omitempty: kalau data nil/kosong, field ini tidak dimunculkan
	Errors  interface{} `json:"errors,omitempty"` // bisa string / map / array tergantung kebutuhan
}

// BuildResponseSuccess digunakan saat request berhasil (HTTP 200/201).
// - message: deskripsi singkat keberhasilan (misal: "Login berhasil").
// - data   : payload utama yang ingin dikirim ke frontend.
func BuildResponseSuccess(message string, data interface{}) APIResponse {
	return APIResponse{
		Status:  true,
		Message: message,
		Data:    data,
	}
}

// BuildResponseFailed digunakan saat terjadi error (HTTP 400, 401, 500, dll).
// - message: pesan utama untuk user (misal: "Input tidak valid").
// - err    : detail error teknis (biasanya string, tapi bisa juga map jika mau lebih detail).
// - data   : data tambahan jika ada (biasanya nil).
func BuildResponseFailed(message string, err interface{}, data interface{}) APIResponse {
	return APIResponse{
		Status:  false,
		Message: message,
		Errors:  err,
		Data:    data,
	}
}
