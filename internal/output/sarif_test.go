package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/mattmezza/timebombs/internal/model"
)

func TestWriteSARIF_Structure(t *testing.T) {
	now := mustDate("2026-04-13")
	bombs := []model.Timebomb{
		{File: "a.py", Line: 42, Deadline: mustDate("2025-05-22"), ID: "JIRA-123", Description: "old."},
		{File: "b.go", Line: 10, Deadline: mustDate("2099-01-01"), Description: "future."},
	}
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, bombs, now, "0.1.0"); err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}
			}
			Results []struct {
				RuleID  string `json:"ruleId"`
				Level   string `json:"level"`
				Message struct {
					Text string `json:"text"`
				}
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						}
						Region struct {
							StartLine int `json:"startLine"`
						}
					} `json:"physicalLocation"`
				}
				Properties map[string]interface{} `json:"properties"`
			}
		}
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid SARIF: %v\n%s", err, buf.String())
	}
	if parsed.Version != "2.1.0" {
		t.Errorf("version: %q", parsed.Version)
	}
	if len(parsed.Runs) != 1 || len(parsed.Runs[0].Results) != 2 {
		t.Fatalf("bad runs/results: %+v", parsed.Runs)
	}
	r0 := parsed.Runs[0].Results[0]
	if r0.Level != "error" {
		t.Errorf("exploded should be error, got %q", r0.Level)
	}
	if r0.Locations[0].PhysicalLocation.ArtifactLocation.URI != "a.py" {
		t.Errorf("uri: %+v", r0.Locations)
	}
	if r0.Locations[0].PhysicalLocation.Region.StartLine != 42 {
		t.Errorf("startLine: %+v", r0.Locations[0].PhysicalLocation.Region)
	}
	if r0.Properties["deadline"] != "2025-05-22" {
		t.Errorf("properties: %+v", r0.Properties)
	}
	if r0.Properties["state"] != "exploded" {
		t.Errorf("state: %+v", r0.Properties)
	}
	r1 := parsed.Runs[0].Results[1]
	if r1.Level != "warning" {
		t.Errorf("ticking should be warning, got %q", r1.Level)
	}
	if parsed.Runs[0].Tool.Driver.Name != "timebombs" {
		t.Errorf("driver name: %q", parsed.Runs[0].Tool.Driver.Name)
	}
	if parsed.Runs[0].Tool.Driver.Version != "0.1.0" {
		t.Errorf("driver version: %q", parsed.Runs[0].Tool.Driver.Version)
	}
}
