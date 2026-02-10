package client

import (
	"encoding/json"
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

func TestUpdateProjectEnv_UpdatesProjectEnvironment(t *testing.T) {
	projectEnv := "A=1\nB=2"
	updateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"projectId":"proj-1","name":"Project One","description":"Test project","env":"` + strings.ReplaceAll(projectEnv, "\n", `\n`) + `"}`))
		case "/project.update":
			updateCalls++

			var payload struct {
				ProjectID   string `json:"projectId"`
				Name        string `json:"name"`
				Description string `json:"description"`
				Env         string `json:"env"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode project.update payload: %v", err)
			}

			if payload.ProjectID != "proj-1" {
				t.Fatalf("unexpected project ID: got %q want %q", payload.ProjectID, "proj-1")
			}
			if payload.Name != "Project One" {
				t.Fatalf("unexpected project name: got %q want %q", payload.Name, "Project One")
			}
			if payload.Description != "Test project" {
				t.Fatalf("unexpected project description: got %q want %q", payload.Description, "Test project")
			}

			projectEnv = payload.Env
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.UpdateProjectEnv("proj-1", func(envMap map[string]string) {
		envMap["C"] = "3"
	})
	if err != nil {
		t.Fatalf("UpdateProjectEnv returned error: %v", err)
	}

	if updateCalls == 0 {
		t.Fatal("expected project.update to be called at least once")
	}

	finalEnv := ParseEnv(projectEnv)
	if finalEnv["A"] != "1" || finalEnv["B"] != "2" || finalEnv["C"] != "3" {
		t.Fatalf("unexpected final env map: %#v", finalEnv)
	}
}

func TestUpdateProjectEnv_NoChangesSkipsUpdate(t *testing.T) {
	updateCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"projectId":"proj-1","name":"Project One","description":"Test project","env":"A=1"}`))
		case "/project.update":
			updateCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	err := c.UpdateProjectEnv("proj-1", func(envMap map[string]string) {
		envMap["A"] = "1"
	})
	if err != nil {
		t.Fatalf("UpdateProjectEnv returned error: %v", err)
	}

	if updateCalls != 0 {
		t.Fatalf("expected no project.update call, got %d", updateCalls)
	}
}
