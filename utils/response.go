package utils

// APIResponse adalah format standar JSON yang akan diterima Frontend.
// Contoh: { "status": true, "message": "Login berhasil", "data": { ... } }
type APIResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // omitempty: kalau data kosong, field ini hilang
	Errors  interface{} `json:"errors,omitempty"`
}

// BuildResponseSuccess digunakan saat request berhasil (HTTP 200/201).
func BuildResponseSuccess(message string, data interface{}) APIResponse {
	return APIResponse{
		Status:  true,
		Message: message,
		Data:    data,
	}
}

// BuildResponseFailed digunakan saat terjadi error (HTTP 400, 401, 500, dll).
func BuildResponseFailed(message string, err string, data interface{}) APIResponse {
	return APIResponse{
		Status:  false,
		Message: message,
		Errors:  err,
		Data:    data,
	}
}