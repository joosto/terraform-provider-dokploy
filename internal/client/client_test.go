package client

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDeleteApplication_UsesDeleteEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/application.stop", "/application.delete":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	if err := c.DeleteApplication("app-123"); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}

	expected := []string{"/application.stop", "/application.delete"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestDeleteApplication_FallsBackToRemove(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/application.stop":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`stop failed`))
		case "/application.delete":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`not found`))
		case "/application.remove":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	if err := c.DeleteApplication("app-123"); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}

	expected := []string{"/application.stop", "/application.delete", "/application.remove"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestDeleteApplication_ReturnsErrorWhenDeleteAndRemoveFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/application.stop":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`stop failed`))
		case "/application.delete":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`delete failed`))
		case "/application.remove":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`remove failed`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.DeleteApplication("app-123")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "application.delete failed") {
		t.Fatalf("expected delete failure in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "application.remove fallback failed") {
		t.Fatalf("expected remove fallback failure in error, got: %v", err)
	}
}
