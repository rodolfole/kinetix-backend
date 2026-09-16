package json

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Write writes a JSON response with the given status code and data
func Write(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// AllowUnknownFields controls whether Read rejects unknown JSON fields.
// Por defecto (false) mantenemos DisallowUnknownFields para detectar payloads
// desalineados con el schema, pero quienes necesiten backward-compatibility
// (p.ej. payloads opcionales) deben llamar ReadAllowUnknown explícitamente.
var AllowUnknownFields = false

// Read decodes a JSON request body into the provided data structure.
// Devuelve el error completo del decoder en lugar de un string opaco,
// así el handler puede mostrar al cliente el problema real (campo
// desconocido, syntax error en JSON, etc).
func Read(r *http.Request, data any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if !AllowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(data); err != nil {
		return err
	}
	// Rechaza si quedan bytes extra (p.ej. "{}{}" o payload duplicado)
	if decoder.More() {
		return &json.SyntaxError{Offset: int64(len(body))}
	}
	return nil
}

// ReadAllowUnknown decodes un payload JSON permitiendo campos desconocidos.
// Útil para endpoints que aceptan datos extensibles.
func ReadAllowUnknown(r *http.Request, data any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(data); err != nil {
		return err
	}
	return nil
}

// WriteError writes a JSON error response
func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
