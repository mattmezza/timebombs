package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/mattmezza/timebombs/internal/model"
)

// SARIF 2.1.0 minimal structure tailored for timebomb results.
// Spec: https://docs.oasis-open.org/sarif/sarif/v2.1.0/

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri,omitempty"`
	Rules          []sarifRule `json:"rules,omitempty"`
}

type sarifRule struct {
	ID               string         `json:"id"`
	ShortDescription sarifMultiText `json:"shortDescription"`
	FullDescription  sarifMultiText `json:"fullDescription"`
	HelpURI          string         `json:"helpUri,omitempty"`
}

type sarifMultiText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID     string                 `json:"ruleId"`
	Level      string                 `json:"level"`
	Message    sarifMultiText         `json:"message"`
	Locations  []sarifLocation        `json:"locations"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// WriteSARIF renders bombs as SARIF 2.1.0.
func WriteSARIF(w io.Writer, bombs []model.Timebomb, now time.Time, toolVersion string) error {
	results := make([]sarifResult, 0, len(bombs))
	for _, b := range bombs {
		state := b.State(now)
		level := "warning"
		if state == model.StateExploded {
			level = "error"
		}
		props := map[string]interface{}{
			"deadline":       b.Deadline.Format("2006-01-02"),
			"state":          string(state),
			"days_remaining": b.DaysRemaining(now),
		}
		if b.ID != "" {
			props["id"] = b.ID
		}
		results = append(results, sarifResult{
			RuleID:  "timebomb",
			Level:   level,
			Message: sarifMultiText{Text: b.Description},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: b.File},
					Region:           sarifRegion{StartLine: b.Line},
				},
			}},
			Properties: props,
		})
	}

	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "timebombs",
				Version:        toolVersion,
				InformationURI: "https://github.com/mattmezza/timebombs",
				Rules: []sarifRule{{
					ID:               "timebomb",
					ShortDescription: sarifMultiText{Text: "Timebomb annotation found."},
					FullDescription:  sarifMultiText{Text: "A TIMEBOMB annotation marks conscious tech debt with a deadline. Warning while ticking; error once the deadline has passed."},
					HelpURI:          "https://github.com/mattmezza/timebombs",
				}},
			}},
			Results: results,
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
