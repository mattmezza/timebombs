package scanner

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	ignore "github.com/sabhiram/go-gitignore"

	"github.com/mattmezza/timebombs/internal/model"
)

// Options controls scanner behavior.
type Options struct {
	// Exclude is a list of user-provided glob patterns (doublestar syntax)
	// evaluated against the path relative to each scan root.
	Exclude []string
	// Include, when non-empty, restricts the scan to files matching any of
	// these globs. Directories are always traversed (subject to Exclude and
	// gitignore); the allowlist only gates leaf files.
	Include []string
	// UseGitignore enables .gitignore-aware filtering.
	UseGitignore bool
	// MaxFileSize skips files larger than this many bytes (0 = no limit).
	MaxFileSize int64
	// Workers controls the number of parallel file-parse workers. 0 means
	// runtime.NumCPU(). 1 forces serial processing.
	Workers int
}

// DefaultMaxFileSize is the default upper bound on a scanned file's size.
const DefaultMaxFileSize int64 = 4 * 1024 * 1024 // 4 MiB

// alwaysSkipDirs are directories never entered regardless of gitignore.
var alwaysSkipDirs = map[string]struct{}{
	".git": {}, ".hg": {}, ".svn": {},
	"node_modules": {}, "vendor": {}, ".venv": {}, "venv": {},
	"__pycache__": {}, "dist": {}, "build": {}, "target": {},
}

// Scan walks the given roots and returns all timebombs found.
func Scan(roots []string, opts Options) ([]model.Timebomb, error) {
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	var all []model.Timebomb
	for _, root := range roots {
		bombs, err := scanRoot(root, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, bombs...)
	}
	return all, nil
}

func scanRoot(root string, opts Options) ([]model.Timebomb, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	// Single file.
	if !info.IsDir() {
		return scanFile(root, root, opts)
	}

	var gi *ignore.GitIgnore
	if opts.UseGitignore {
		giPath := filepath.Join(root, ".gitignore")
		if _, err := os.Stat(giPath); err == nil {
			gi, _ = ignore.CompileIgnoreFile(giPath)
		}
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	jobs := make(chan string, workers*4)
	var (
		mu  sync.Mutex
		out []model.Timebomb
		wg  sync.WaitGroup
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				bombs, ferr := scanFile(path, path, opts)
				if ferr != nil || len(bombs) == 0 {
					continue
				}
				mu.Lock()
				out = append(out, bombs...)
				mu.Unlock()
			}
		}()
	}

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		name := d.Name()

		if d.IsDir() {
			if _, skip := alwaysSkipDirs[name]; skip {
				return filepath.SkipDir
			}
			if gi != nil && gi.MatchesPath(rel+string(filepath.Separator)) {
				return filepath.SkipDir
			}
			if matchAnyGlob(opts.Exclude, rel) || matchAnyGlob(opts.Exclude, rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if gi != nil && gi.MatchesPath(rel) {
			return nil
		}
		if matchAnyGlob(opts.Exclude, rel) {
			return nil
		}
		if len(opts.Include) > 0 && !matchAnyGlob(opts.Include, rel) {
			return nil
		}

		jobs <- path
		return nil
	})
	close(jobs)
	wg.Wait()
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

func matchAnyGlob(patterns []string, path string) bool {
	return MatchAny(patterns, path)
}

// MatchAny reports whether path matches any of the doublestar patterns.
// Exposed so callers outside the scanner can apply the same matching rules.
func MatchAny(patterns []string, path string) bool {
	path = filepath.ToSlash(path)
	for _, p := range patterns {
		ok, err := doublestar.PathMatch(p, path)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func scanFile(displayPath, actualPath string, opts Options) ([]model.Timebomb, error) {
	info, err := os.Stat(actualPath)
	if err != nil {
		return nil, err
	}
	if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
		return nil, nil
	}
	data, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, err
	}
	if isBinary(data) {
		return nil, nil
	}
	bombs := Parse(data)
	for i := range bombs {
		bombs[i].File = filepath.ToSlash(displayPath)
	}
	return bombs, nil
}

// isBinary returns true if the file contents look non-text.
func isBinary(data []byte) bool {
	n := len(data)
	if n > 512 {
		n = 512
	}
	if bytes.IndexByte(data[:n], 0) >= 0 {
		return true
	}
	return false
}

// ParseDuration parses a human-friendly duration like "30d", "2w", "3m", "1y".
// Also accepts plain Go duration syntax ("72h") as a fallback.
func ParseDuration(s string) (days int, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty duration")
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	var mult int
	switch unit {
	case 'd':
		mult = 1
	case 'w':
		mult = 7
	case 'm':
		mult = 30
	case 'y':
		mult = 365
	default:
		return 0, errors.New("unsupported duration unit; use d/w/m/y")
	}
	n := 0
	if numStr == "" {
		return 0, errors.New("missing number in duration")
	}
	for _, r := range numStr {
		if r < '0' || r > '9' {
			return 0, errors.New("invalid duration")
		}
		n = n*10 + int(r-'0')
	}
	return n * mult, nil
}
