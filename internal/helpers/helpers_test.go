package helpers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeBodyJSON_Valid(t *testing.T) {
	type dest struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	body := `{"name":"test","count":42}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Name != "test" || d.Count != 42 {
		t.Fatalf("unexpected decoded value: %+v", d)
	}
}

func TestDecodeBodyJSON_NoContentType(t *testing.T) {
	// Missing Content-Type header should still decode successfully.
	type dest struct {
		Value string `json:"value"`
	}
	body := `{"value":"ok"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Value != "ok" {
		t.Fatalf("expected 'ok', got %q", d.Value)
	}
}

func TestDecodeBodyJSON_InvalidContentType(t *testing.T) {
	body := `{"value":"ok"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	var d struct{ Value string }
	err := DecodeBodyJSON(w, r, &d)
	if err == nil {
		t.Fatal("expected error for invalid Content-Type, got nil")
	}
	if err.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status %d, got %d", http.StatusUnsupportedMediaType, err.Status)
	}
}

func TestDecodeBodyJSON_ContentTypeWithCharset(t *testing.T) {
	// "application/json; charset=utf-8" should be accepted.
	type dest struct {
		Value string `json:"value"`
	}
	body := `{"value":"ok"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	w := httptest.NewRecorder()

	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeBodyJSON_Malformed(t *testing.T) {
	tests := map[string]struct {
		body       string
		wantStatus int
	}{
		"bad syntax":        {body: `{bad json}`, wantStatus: http.StatusBadRequest},
		"unexpected EOF":    {body: `{"name":`, wantStatus: http.StatusBadRequest},
		"wrong type":        {body: `{"count": "not_an_int"}`, wantStatus: http.StatusBadRequest},
		"unknown field":     {body: `{"unknown_field":"x"}`, wantStatus: http.StatusBadRequest},
		"empty body":        {body: ``, wantStatus: http.StatusBadRequest},
		"multiple objects":  {body: `{"a":1}{"b":2}`, wantStatus: http.StatusBadRequest},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Use a struct with an int field to catch type mismatches.
			type dest struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			var d dest
			err := DecodeBodyJSON(w, r, &d)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Status != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, err.Status)
			}
		})
	}
}

func TestDecodeBodyJSON_Oversized(t *testing.T) {
	// Body exceeds 1MB limit.
	body := `{"name":"` + strings.Repeat("x", 1_048_577) + `"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	type dest struct {
		Name string `json:"name"`
	}
	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}
	if err.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, err.Status)
	}
}

func TestDecodeBodyJSON_InvalidTargetType(t *testing.T) {
	// JSON number "42" into a struct target should fail.
	body := `42`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	type dest struct {
		Name string `json:"name"`
	}
	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}
}

func TestDecodeBodyJSON_ArrayInsteadOfObject(t *testing.T) {
	body := `[{"name":"test"}]`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	type dest struct {
		Name string `json:"name"`
	}
	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err == nil {
		t.Fatal("expected error for array input, got nil")
	}
}

func TestMalformedRequest_Error(t *testing.T) {
	mr := &malformedRequest{Status: 400, Message: "bad input"}
	if mr.Error() != "bad input" {
		t.Fatalf("expected 'bad input', got %q", mr.Error())
	}
}

func TestDecodeBodyJSON_DisallowUnknownFields(t *testing.T) {
	type dest struct {
		Name string `json:"name"`
	}
	body := `{"name":"test","extra_field":"should_fail"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var d dest
	err := DecodeBodyJSON(w, r, &d)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("unknown field")) {
		t.Fatalf("error message should mention 'unknown field', got: %s", err.Error())
	}
}
