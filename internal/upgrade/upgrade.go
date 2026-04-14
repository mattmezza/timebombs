// Package upgrade implements self-replacement of the running binary with a
// matching asset from a GitHub release.
package upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultRepo is the GitHub slug used when callers don't override it.
const DefaultRepo = "mattmezza/timebombs"

// Asset is one downloadable file attached to a release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// Release is the subset of a GitHub release we care about.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Client fetches release metadata from GitHub.
type Client struct {
	HTTP    *http.Client
	Repo    string // "owner/name"
	BaseURL string // override for tests
}

// NewClient returns a client with sane defaults.
func NewClient() *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Repo:    DefaultRepo,
		BaseURL: "https://api.github.com",
	}
}

// Latest returns the release tagged as latest.
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	return c.fetch(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", c.BaseURL, c.Repo))
}

// ByTag returns the release for a specific tag (e.g. "v0.3.0").
func (c *Client) ByTag(ctx context.Context, tag string) (*Release, error) {
	return c.fetch(ctx, fmt.Sprintf("%s/repos/%s/releases/tags/%s", c.BaseURL, c.Repo, url.PathEscape(tag)))
}

func (c *Client) fetch(ctx context.Context, u string) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "timebombs-upgrade")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("release not found (%s)", u)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("github %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &r, nil
}

// AssetFor returns the asset matching the given goos/goarch, using the naming
// convention produced by this project's .goreleaser.yml.
func AssetFor(r *Release, goos, goarch string) (*Asset, error) {
	os, arch, ext, err := platformTriple(goos, goarch)
	if err != nil {
		return nil, err
	}
	want := fmt.Sprintf("timebombs_%s_%s%s", os, arch, ext)
	for i := range r.Assets {
		if r.Assets[i].Name == want {
			return &r.Assets[i], nil
		}
	}
	names := make([]string, 0, len(r.Assets))
	for _, a := range r.Assets {
		names = append(names, a.Name)
	}
	return nil, fmt.Errorf("no asset %q in release %s (have: %v)", want, r.TagName, names)
}

func platformTriple(goos, goarch string) (os, arch, ext string, err error) {
	switch goos {
	case "linux":
		os = "Linux"
		ext = ".tar.gz"
	case "darwin":
		os = "Darwin"
		ext = ".tar.gz"
	case "windows":
		os = "Windows"
		ext = ".zip"
	default:
		return "", "", "", fmt.Errorf("unsupported OS %q", goos)
	}
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	default:
		return "", "", "", fmt.Errorf("unsupported arch %q", goarch)
	}
	return os, arch, ext, nil
}

// NeedsUpgrade reports whether current (e.g. "0.3.0" or "v0.3.0") is
// different from latest (typically "vX.Y.Z"). It does a string comparison
// after stripping a leading "v" from both sides — intentionally simple.
func NeedsUpgrade(current, latest string) bool {
	return strings.TrimPrefix(current, "v") != strings.TrimPrefix(latest, "v")
}

// Download fetches the asset's body. Caller closes.
func (c *Client) Download(ctx context.Context, a *Asset) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "timebombs-upgrade")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("download %s: %s", a.Name, resp.Status)
	}
	return resp.Body, nil
}

// ExtractBinary reads either a gzip-tar or a zip stream and writes the
// `timebombs` executable inside it to destPath. Returns the path on success.
func ExtractBinary(src io.Reader, assetName, destPath string) error {
	wantName := "timebombs"
	if strings.HasSuffix(assetName, ".exe.zip") || strings.Contains(assetName, "Windows") {
		wantName = "timebombs.exe"
	}
	if strings.HasSuffix(assetName, ".zip") {
		return extractZipTo(src, wantName, destPath)
	}
	return extractTarGzTo(src, wantName, destPath)
}

func extractTarGzTo(src io.Reader, want, destPath string) error {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(h.Name) != want {
			continue
		}
		return writeExe(tr, destPath)
	}
	return fmt.Errorf("no %s entry found in tarball", want)
}

func extractZipTo(src io.Reader, want, destPath string) error {
	// archive/zip needs a ReadSeeker; stage to a temp file first.
	tmp, err := os.CreateTemp("", "timebombs-zip-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	zr, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != want {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return writeExe(rc, destPath)
	}
	return fmt.Errorf("no %s entry found in zip", want)
}

func writeExe(r io.Reader, destPath string) error {
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Verify runs the new binary with `version` and checks the output is
// non-empty. Catches corrupt downloads or binaries that won't execute here.
func Verify(ctx context.Context, binPath string) error {
	cmd := exec.CommandContext(ctx, binPath, "version")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("run %s version: %w", binPath, err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return errors.New("new binary printed empty version")
	}
	return nil
}

// ReplaceSelf atomically swaps the running binary at exePath with the file
// at newBin. Both must live on the same filesystem (rename requirement).
func ReplaceSelf(exePath, newBin string) error {
	dir := filepath.Dir(exePath)
	staged := filepath.Join(dir, filepath.Base(exePath)+".new")
	// Copy the new binary next to the current one. Same-dir keeps rename
	// atomic across filesystems.
	in, err := os.Open(newBin)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(staged, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("stage %s: %w (try running with sudo if the install dir is system-owned)", staged, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(staged)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(staged)
		return err
	}
	if err := os.Rename(staged, exePath); err != nil {
		os.Remove(staged)
		return fmt.Errorf("replace %s: %w (on Windows, re-run via a helper; on Unix, check permissions)", exePath, err)
	}
	return nil
}

// ExePath returns the absolute path to the running executable, resolving
// any symlink.
func ExePath() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	return p, nil
}

// GOOS and GOARCH let callers override the detected platform in tests.
var (
	GOOS   = runtime.GOOS
	GOARCH = runtime.GOARCH
)
