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

func TestCreateDatabase_MySQLDirectResponseUsesMysqlIDAsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mysql.create":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"mysqlId":"mysql-123","name":"test-db","appName":"test-db"}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	db, err := c.CreateDatabase("project-1", "env-1", "test-db", "mysql", "secret", "mysql:8")
	if err != nil {
		t.Fatalf("CreateDatabase returned error: %v", err)
	}
	if db.ID != "mysql-123" {
		t.Fatalf("unexpected database ID: got %q want %q", db.ID, "mysql-123")
	}
	if db.Type != "mysql" {
		t.Fatalf("unexpected database type: got %q want %q", db.Type, "mysql")
	}
}

func TestCreateDatabase_MySQLWrappedResponseUsesMysqlIDAsID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mysql.create":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"database":{"mysqlId":"mysql-456","name":"test-db","appName":"test-db"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	db, err := c.CreateDatabase("project-1", "env-1", "test-db", "mysql", "secret", "mysql:8")
	if err != nil {
		t.Fatalf("CreateDatabase returned error: %v", err)
	}
	if db.ID != "mysql-456" {
		t.Fatalf("unexpected database ID: got %q want %q", db.ID, "mysql-456")
	}
	if db.Type != "mysql" {
		t.Fatalf("unexpected database type: got %q want %q", db.Type, "mysql")
	}
}

func TestCreateVolumeBackup_UsesComposeEndpointAndPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/volumeBackups.create":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			if payload["serviceType"] != "compose" {
				t.Fatalf("unexpected serviceType: %#v", payload["serviceType"])
			}
			if payload["composeId"] != "compose-123" {
				t.Fatalf("unexpected composeId: %#v", payload["composeId"])
			}
			if payload["destinationId"] != "dest-123" {
				t.Fatalf("unexpected destinationId: %#v", payload["destinationId"])
			}
			if payload["appName"] != "fca-ghost-kqlble" {
				t.Fatalf("unexpected appName: %#v", payload["appName"])
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"volumeBackup":{"volumeBackupId":"vb-123","name":"ghost-content","composeId":"compose-123","serviceName":"ghost","volumeName":"ghost-content-data","destinationId":"dest-123","cronExpression":"0 3 * * *","enabled":true,"keepLatestCount":14}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	backup, err := c.CreateVolumeBackup(VolumeBackup{
		Name:            "ghost-content",
		ComposeID:       "compose-123",
		AppName:         "fca-ghost-kqlble",
		ServiceName:     "ghost",
		VolumeName:      "ghost-content-data",
		DestinationID:   "dest-123",
		CronExpression:  "0 3 * * *",
		Enabled:         true,
		TurnOff:         false,
		KeepLatestCount: 14,
	})
	if err != nil {
		t.Fatalf("CreateVolumeBackup returned error: %v", err)
	}
	if backup.ID != "vb-123" {
		t.Fatalf("unexpected backup ID: got %q want %q", backup.ID, "vb-123")
	}
}

func TestCreateVolumeBackup_FallbackLookupOnBooleanResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/volumeBackups.create":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/volumeBackups.list":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`not found`))
		case "/volumeBackups.all":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"volumeBackupId":"vb-lookup","name":"ghost-content","composeId":"compose-123","serviceName":"ghost","volumeName":"ghost-content-data","destinationId":"dest-123"}]`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	backup, err := c.CreateVolumeBackup(VolumeBackup{
		Name:          "ghost-content",
		ComposeID:     "compose-123",
		ServiceName:   "ghost",
		VolumeName:    "ghost-content-data",
		DestinationID: "dest-123",
	})
	if err != nil {
		t.Fatalf("CreateVolumeBackup returned error: %v", err)
	}
	if backup.ID != "vb-lookup" {
		t.Fatalf("unexpected backup ID: got %q want %q", backup.ID, "vb-lookup")
	}
}

func TestDeleteVolumeBackup_UsesDeleteEndpoint(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/volumeBackups.delete":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")
	if err := c.DeleteVolumeBackup("vb-123"); err != nil {
		t.Fatalf("DeleteVolumeBackup returned error: %v", err)
	}

	expected := []string{"/volumeBackups.delete"}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("unexpected call order: got %v want %v", calls, expected)
	}
}

func TestFindBackupDestinationByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.all":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destinations":[{"destinationId":"dest-hetzner","name":"Hetzner backup s3 bucket","type":"s3"}]}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.FindBackupDestinationByName("hetzner backup s3 bucket")
	if err != nil {
		t.Fatalf("FindBackupDestinationByName returned error: %v", err)
	}
	if destination.ID != "dest-hetzner" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-hetzner")
	}
}

func TestCreateBackupDestination_UsesDestinationCreateEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.create":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}

			if payload["name"] != "Hetzner backup s3 bucket" {
				t.Fatalf("unexpected name: %#v", payload["name"])
			}
			if payload["provider"] != "s3" {
				t.Fatalf("unexpected provider: %#v", payload["provider"])
			}
			if payload["bucket"] != "backups-2f6fe75d" {
				t.Fatalf("unexpected bucket: %#v", payload["bucket"])
			}
			if payload["accessKey"] != "access-key" {
				t.Fatalf("unexpected accessKey: %#v", payload["accessKey"])
			}
			if payload["secretAccessKey"] != "secret-key" {
				t.Fatalf("unexpected secretAccessKey: %#v", payload["secretAccessKey"])
			}
			if _, ok := payload["accessKeyId"]; ok {
				t.Fatalf("unexpected alias field accessKeyId in payload")
			}
			if _, ok := payload["secretKey"]; ok {
				t.Fatalf("unexpected alias field secretKey in payload")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destination":{"destinationId":"dest-123","name":"Hetzner backup s3 bucket","type":"s3","bucket":"backups-2f6fe75d","region":"nbg1","endpoint":"https://nbg1.your-objectstorage.com"}}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.CreateBackupDestination(BackupDestination{
		Name:            "Hetzner backup s3 bucket",
		Type:            "s3",
		Bucket:          "backups-2f6fe75d",
		Region:          "nbg1",
		Endpoint:        "https://nbg1.your-objectstorage.com",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("CreateBackupDestination returned error: %v", err)
	}
	if destination.ID != "dest-123" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-123")
	}
}

func TestUpdateBackupDestination_FallsBackToGetDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/destination.update":
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode payload: %v", err)
			}
			if payload["provider"] != "s3" {
				t.Fatalf("unexpected provider: %#v", payload["provider"])
			}
			if payload["accessKey"] != "access-key" {
				t.Fatalf("unexpected accessKey: %#v", payload["accessKey"])
			}
			if payload["secretAccessKey"] != "secret-key" {
				t.Fatalf("unexpected secretAccessKey: %#v", payload["secretAccessKey"])
			}
			if _, ok := payload["accessKeyId"]; ok {
				t.Fatalf("unexpected alias field accessKeyId in payload")
			}
			if _, ok := payload["secretKey"]; ok {
				t.Fatalf("unexpected alias field secretKey in payload")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`true`))
		case "/destination.one":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"destinationId":"dest-123","name":"Hetzner backup s3 bucket","type":"s3","bucket":"backups-2f6fe75d","region":"nbg1","endpoint":"https://nbg1.your-objectstorage.com"}`))
		default:
			t.Fatalf("unexpected endpoint called: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := NewDokployClient(server.URL, "test-key")

	destination, err := c.UpdateBackupDestination(BackupDestination{
		ID:              "dest-123",
		Name:            "Hetzner backup s3 bucket",
		Type:            "s3",
		Bucket:          "backups-2f6fe75d",
		Region:          "nbg1",
		Endpoint:        "https://nbg1.your-objectstorage.com",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
	})
	if err != nil {
		t.Fatalf("UpdateBackupDestination returned error: %v", err)
	}
	if destination.ID != "dest-123" {
		t.Fatalf("unexpected destination ID: got %q want %q", destination.ID, "dest-123")
	}
}
