package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetFor(t *testing.T) {
	r := &Release{
		TagName: "v0.4.0",
		Assets: []Asset{
			{Name: "timebombs_Linux_x86_64.tar.gz"},
			{Name: "timebombs_Linux_arm64.tar.gz"},
			{Name: "timebombs_Darwin_x86_64.tar.gz"},
			{Name: "timebombs_Darwin_arm64.tar.gz"},
			{Name: "timebombs_Windows_x86_64.zip"},
			{Name: "timebombs_Windows_arm64.zip"},
			{Name: "checksums.txt"},
		},
	}
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "timebombs_Linux_x86_64.tar.gz"},
		{"linux", "arm64", "timebombs_Linux_arm64.tar.gz"},
		{"darwin", "amd64", "timebombs_Darwin_x86_64.tar.gz"},
		{"darwin", "arm64", "timebombs_Darwin_arm64.tar.gz"},
		{"windows", "amd64", "timebombs_Windows_x86_64.zip"},
	}
	for _, c := range cases {
		a, err := AssetFor(r, c.goos, c.goarch)
		if err != nil {
			t.Errorf("%s/%s: %v", c.goos, c.goarch, err)
			continue
		}
		if a.Name != c.want {
			t.Errorf("%s/%s: got %s want %s", c.goos, c.goarch, a.Name, c.want)
		}
	}
}

func TestAssetFor_Missing(t *testing.T) {
	r := &Release{TagName: "v0.4.0", Assets: []Asset{{Name: "timebombs_Linux_x86_64.tar.gz"}}}
	_, err := AssetFor(r, "darwin", "arm64")
	if err == nil {
		t.Errorf("expected error for missing asset")
	}
}

func TestAssetFor_UnsupportedPlatform(t *testing.T) {
	r := &Release{}
	if _, err := AssetFor(r, "plan9", "amd64"); err == nil {
		t.Errorf("expected error for unsupported OS")
	}
	if _, err := AssetFor(r, "linux", "mips"); err == nil {
		t.Errorf("expected error for unsupported arch")
	}
}

func TestNeedsUpgrade(t *testing.T) {
	cases := []struct {
		cur, latest string
		want        bool
	}{
		{"0.3.0", "v0.4.0", true},
		{"v0.3.0", "v0.4.0", true},
		{"0.4.0", "v0.4.0", false},
		{"v0.4.0", "v0.4.0", false},
		{"dev", "v0.4.0", true},
	}
	for _, c := range cases {
		if got := NeedsUpgrade(c.cur, c.latest); got != c.want {
			t.Errorf("NeedsUpgrade(%q, %q) = %v want %v", c.cur, c.latest, got, c.want)
		}
	}
}

func TestClient_Latest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/mattmezza/timebombs/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header")
		}
		json.NewEncoder(w).Encode(Release{
			TagName: "v0.9.9",
			Assets:  []Asset{{Name: "timebombs_Linux_x86_64.tar.gz", DownloadURL: "http://example.com/x"}},
		})
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	got, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.TagName != "v0.9.9" {
		t.Errorf("tag: %q", got.TagName)
	}
}

func TestClient_Latest_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient()
	c.BaseURL = srv.URL
	if _, err := c.Latest(context.Background()); err == nil {
		t.Errorf("expected error")
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	body := []byte("#!/bin/sh\necho 0.9.9\n")
	hdr := &tar.Header{Name: "timebombs", Mode: 0o755, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "timebombs")
	if err := ExtractBinary(&buf, "timebombs_Linux_x86_64.tar.gz", dest); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, body) {
		t.Errorf("content mismatch:\n%s", out)
	}
	fi, _ := os.Stat(dest)
	if fi.Mode()&0o111 == 0 {
		t.Errorf("dest not executable: %v", fi.Mode())
	}
}

func TestExtractBinary_MissingEntry(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "README.md", Size: 0})
	tw.Close()
	gz.Close()

	dest := filepath.Join(t.TempDir(), "timebombs")
	if err := ExtractBinary(&buf, "timebombs_Linux_x86_64.tar.gz", dest); err == nil {
		t.Errorf("expected error on missing binary entry")
	}
}

func TestReplaceSelf(t *testing.T) {
	dir := t.TempDir()
	cur := filepath.Join(dir, "timebombs")
	if err := os.WriteFile(cur, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	newBin := filepath.Join(dir, "new-timebombs")
	if err := os.WriteFile(newBin, []byte("NEW"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceSelf(cur, newBin); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cur)
	if string(data) != "NEW" {
		t.Errorf("current binary not replaced: %q", data)
	}
	// Staged file should be gone.
	if _, err := os.Stat(cur + ".new"); err == nil {
		t.Errorf("staged file not cleaned up")
	}
}

// Sanity: the embedded asset naming matches AssetFor output for the build
// host.
func TestAssetNaming_RoundTrip(t *testing.T) {
	r := &Release{Assets: []Asset{
		{Name: "timebombs_Linux_x86_64.tar.gz"},
		{Name: "timebombs_Darwin_arm64.tar.gz"},
	}}
	a, err := AssetFor(r, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a.Name, "Linux_x86_64") {
		t.Errorf("unexpected asset: %s", a.Name)
	}
}

// Silence unused import warnings in minimal builds.
var _ = io.EOF
