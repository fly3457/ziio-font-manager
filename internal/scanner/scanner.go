package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fontManager/internal/diagnostics"
	"fontManager/internal/fontmeta"
	"fontManager/internal/models"
	"fontManager/internal/store"
)

const parseTimeout = 12 * time.Second

var errParseTimeout = errors.New("font metadata parse timed out")

type Scanner struct {
	store  *store.Store
	logDir string
}

type scanScope struct {
	kind     string
	path     string
	walkPath string
}

var parseFontFile = fontmeta.ParseFile

func New(s *store.Store, logDir ...string) *Scanner {
	scanner := &Scanner{store: s}
	if len(logDir) > 0 {
		scanner.logDir = logDir[0]
	}
	return scanner
}

func (s *Scanner) ScanRoot(root models.LibraryRoot) (models.ScanResult, error) {
	rootPath, err := filepath.Abs(root.Path)
	if err != nil {
		return models.ScanResult{RootID: root.ID, Scope: "root"}, err
	}
	root.Path = filepath.Clean(rootPath)
	return s.scan(root, scanScope{kind: "root", walkPath: root.Path})
}

func (s *Scanner) ScanFolder(root models.LibraryRoot, folderPath string) (models.ScanResult, error) {
	relPath, absPath, err := resolveFolderScope(root.Path, folderPath)
	result := models.ScanResult{RootID: root.ID, Scope: "folder", ScopePath: relPath}
	if err != nil {
		return result, err
	}
	rootPath, err := filepath.Abs(root.Path)
	if err != nil {
		return result, err
	}
	root.Path = filepath.Clean(rootPath)
	return s.scan(root, scanScope{kind: "folder", path: relPath, walkPath: absPath})
}

func (s *Scanner) scan(root models.LibraryRoot, scope scanScope) (models.ScanResult, error) {
	result := models.ScanResult{RootID: root.ID, Scope: scope.kind, ScopePath: scope.path}
	s.logScan("scan-start root=%d scope=%s scopePath=%q path=%q", root.ID, scope.kind, scope.path, scope.walkPath)
	jobID, err := s.store.BeginScan(root.ID, scope.kind, scope.path)
	if err != nil {
		s.logScan("scan-begin-error root=%d scope=%s scopePath=%q path=%q error=%q", root.ID, scope.kind, scope.path, scope.walkPath, err.Error())
		return result, err
	}

	if info, err := os.Stat(scope.walkPath); err != nil {
		if scope.kind == "folder" && errors.Is(err, os.ErrNotExist) {
			missing, missingErr := s.store.MarkMissingFilesInScope(root.ID, scope.walkPath, map[string]struct{}{})
			result.Missing = missing
			if missingErr != nil {
				result.Failed++
				_ = s.updateScan(jobID, "error", result, missingErr.Error())
				s.logScan("mark-missing-error root=%d scope=%s scopePath=%q missing=%d error=%q", root.ID, scope.kind, scope.path, result.Missing, missingErr.Error())
				return result, missingErr
			}
			if err := s.store.FinishRootScan(root.ID); err != nil {
				result.Failed++
				_ = s.updateScan(jobID, "error", result, err.Error())
				s.logScan("finish-root-error root=%d scope=%s scopePath=%q missing=%d error=%q", root.ID, scope.kind, scope.path, result.Missing, err.Error())
				return result, err
			}
			err = s.updateScan(jobID, "complete", result, "Folder no longer exists; indexed files were removed from Ziio")
			s.logScan("scan-finish root=%d scope=%s scopePath=%q status=complete total=0 processed=0 added=0 updated=0 failed=%d missing=%d unchanged=%d error=%q", root.ID, scope.kind, scope.path, result.Failed, result.Missing, result.Unchanged, errorText(err))
			return result, err
		}
		result.Failed++
		_ = s.updateScan(jobID, "error", result, err.Error())
		s.logScan("scan-stat-error root=%d scope=%s scopePath=%q path=%q error=%q", root.ID, scope.kind, scope.path, scope.walkPath, err.Error())
		return result, err
	} else if !info.IsDir() {
		err := fmt.Errorf("scan scope is not a folder: %s", scope.walkPath)
		result.Failed++
		_ = s.updateScan(jobID, "error", result, err.Error())
		s.logScan("scan-scope-error root=%d scope=%s scopePath=%q path=%q error=%q", root.ID, scope.kind, scope.path, scope.walkPath, err.Error())
		return result, err
	}

	var candidates []string
	walkErr := filepath.WalkDir(scope.walkPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Failed++
			s.logScan("walk-error root=%d path=%q error=%q", root.ID, path, err.Error())
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if fontmeta.IsSupportedPath(path) {
			candidates = append(candidates, path)
		}
		return nil
	})
	if walkErr != nil {
		result.Total = len(candidates)
		_ = s.updateScan(jobID, "error", result, walkErr.Error())
		s.logScan("scan-walk-fatal root=%d scope=%s scopePath=%q path=%q candidates=%d error=%q", root.ID, scope.kind, scope.path, scope.walkPath, len(candidates), walkErr.Error())
		return result, walkErr
	}

	result.Total = len(candidates)
	s.logScan("scan-candidates root=%d scope=%s scopePath=%q path=%q total=%d", root.ID, scope.kind, scope.path, scope.walkPath, result.Total)
	seen := make(map[string]struct{}, len(candidates))
	_ = s.updateScan(jobID, "running", result, "Scanning font files")

	for _, path := range candidates {
		result.Processed++
		clean, err := filepath.Abs(path)
		if err == nil {
			path = filepath.Clean(clean)
		}
		seen[path] = struct{}{}
		s.logScan("current root=%d candidate=%d/%d path=%q", root.ID, result.Processed, result.Total, path)

		if info, err := os.Stat(path); err == nil {
			modifiedAt := info.ModTime().Format("2006-01-02T15:04:05-07:00")
			current, err := s.store.FileIsCurrent(root.ID, path, info.Size(), modifiedAt)
			if err == nil && current {
				result.Unchanged++
				if result.Processed%100 == 0 || result.Processed == result.Total {
					s.logScan("progress root=%d scope=%s processed=%d/%d added=%d updated=%d failed=%d missing=%d unchanged=%d message=%q", root.ID, scope.kind, result.Processed, result.Total, result.Added, result.Updated, result.Failed, result.Missing, result.Unchanged, "skipping unchanged files")
				}
				if result.Processed%50 == 0 || result.Processed == result.Total {
					_ = s.updateScan(jobID, "running", result, "Skipping unchanged files")
				}
				continue
			}
		}

		parsed, err := parseFileWithTimeout(path, parseTimeout)
		if err != nil {
			err = scanFileError(path, result.Processed, result.Total, err)
			result.Failed++
			if fallback, fallbackErr := fontmeta.ErrorFile(path, err); fallbackErr == nil {
				_, _, _ = s.store.UpsertFontFile(store.FileUpsert{
					RootID:           root.ID,
					Path:             path,
					FileName:         fallback.File.FileName,
					Format:           fallback.File.Format,
					Size:             fallback.File.Size,
					ModifiedAt:       fallback.File.ModifiedAt,
					Hash:             fallback.File.Hash,
					Status:           fallback.File.Status,
					Error:            fallback.File.Error,
					PreviewSupported: fallback.File.PreviewSupported,
				}, fallback.Faces)
			}
			s.logScan("file-error root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, err.Error())
			_ = s.updateScan(jobID, "running", result, filepath.Base(path)+": "+err.Error())
			continue
		}
		parsed.File.RootID = root.ID
		fileID, added, err := s.store.UpsertFontFile(store.FileUpsert{
			RootID:           root.ID,
			Path:             path,
			FileName:         parsed.File.FileName,
			Format:           parsed.File.Format,
			Size:             parsed.File.Size,
			ModifiedAt:       parsed.File.ModifiedAt,
			Hash:             parsed.File.Hash,
			Status:           parsed.File.Status,
			Error:            parsed.File.Error,
			PreviewSupported: parsed.File.PreviewSupported,
		}, parsed.Faces)
		_ = fileID
		if err != nil {
			result.Failed++
			s.logScan("upsert-error root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, err.Error())
		} else if added {
			result.Added++
		} else {
			result.Updated++
		}
		if err == nil && parsed.File.Status == "error" {
			result.Failed++
			s.logScan("file-error-status root=%d candidate=%d/%d path=%q error=%q", root.ID, result.Processed, result.Total, path, parsed.File.Error)
		}
		if result.Processed%100 == 0 || result.Processed == result.Total {
			s.logScan("progress root=%d scope=%s processed=%d/%d added=%d updated=%d failed=%d missing=%d unchanged=%d message=%q", root.ID, scope.kind, result.Processed, result.Total, result.Added, result.Updated, result.Failed, result.Missing, result.Unchanged, parsed.File.FileName)
		}
		if result.Processed%25 == 0 || result.Processed == result.Total {
			_ = s.updateScan(jobID, "running", result, parsed.File.FileName)
		}
	}

	var missing int
	if scope.kind == "folder" {
		missing, err = s.store.MarkMissingFilesInScope(root.ID, scope.walkPath, seen)
	} else {
		missing, err = s.store.MarkMissingFiles(root.ID, seen)
	}
	result.Missing = missing
	if err != nil {
		result.Failed++
		_ = s.updateScan(jobID, "error", result, err.Error())
		s.logScan("mark-missing-error root=%d scope=%s scopePath=%q processed=%d/%d added=%d updated=%d failed=%d missing=%d unchanged=%d error=%q", root.ID, scope.kind, scope.path, result.Processed, result.Total, result.Added, result.Updated, result.Failed, result.Missing, result.Unchanged, err.Error())
		return result, err
	}
	if err := s.store.FinishRootScan(root.ID); err != nil {
		result.Failed++
		_ = s.updateScan(jobID, "error", result, err.Error())
		s.logScan("finish-root-error root=%d scope=%s scopePath=%q processed=%d/%d added=%d updated=%d failed=%d missing=%d unchanged=%d error=%q", root.ID, scope.kind, scope.path, result.Processed, result.Total, result.Added, result.Updated, result.Failed, result.Missing, result.Unchanged, err.Error())
		return result, err
	}

	status := "complete"
	if result.Failed > 0 {
		status = "complete_with_errors"
	}
	err = s.updateScan(jobID, status, result, "Scan finished")
	s.logScan("scan-finish root=%d scope=%s scopePath=%q status=%s total=%d processed=%d added=%d updated=%d failed=%d missing=%d unchanged=%d error=%q", root.ID, scope.kind, scope.path, status, result.Total, result.Processed, result.Added, result.Updated, result.Failed, result.Missing, result.Unchanged, errorText(err))
	return result, err
}

func (s *Scanner) updateScan(jobID int64, status string, result models.ScanResult, message string) error {
	return s.store.UpdateScan(jobID, status, result.Total, result.Processed, result.Added, result.Updated, result.Failed, message, result.Missing, result.Unchanged)
}

func resolveFolderScope(rootPath, folderPath string) (string, string, error) {
	trimmed := strings.TrimSpace(folderPath)
	if trimmed == "" {
		return "", "", fmt.Errorf("no folder selected")
	}
	rel := filepath.Clean(filepath.FromSlash(trimmed))
	if rel == "." || rel == string(filepath.Separator) {
		return "", "", fmt.Errorf("no folder selected")
	}
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("folder path is outside the font library")
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", "", err
	}
	rootAbs = filepath.Clean(rootAbs)
	absPath := filepath.Clean(filepath.Join(rootAbs, rel))
	if !samePath(rootAbs, absPath) && !pathContains(rootAbs, absPath) {
		return "", "", fmt.Errorf("folder path is outside the font library")
	}
	return filepath.ToSlash(rel), absPath, nil
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func pathContains(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if runtime.GOOS == "windows" {
		parent = strings.ToLower(parent)
		child = strings.ToLower(child)
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func parseFileWithTimeout(path string, timeout time.Duration) (fontmeta.ParsedFile, error) {
	type parseResult struct {
		parsed fontmeta.ParsedFile
		err    error
	}
	done := make(chan parseResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- parseResult{err: fmt.Errorf("font metadata parser panic: %v", r)}
			}
		}()
		parsed, err := parseFontFile(path)
		done <- parseResult{parsed: parsed, err: err}
	}()
	select {
	case result := <-done:
		return result.parsed, result.err
	case <-time.After(timeout):
		return fontmeta.ParsedFile{}, errParseTimeout
	}
}

func scanFileError(path string, index, total int, err error) error {
	return fmt.Errorf("candidate %d/%d path=%q: %w", index, total, path, err)
}

func (s *Scanner) logScan(format string, args ...any) {
	diagnostics.Appendf(s.logDir, "scan.log", format, args...)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
