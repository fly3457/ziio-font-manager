package library

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"fontManager/internal/diagnostics"
	"fontManager/internal/fontmeta"
	"fontManager/internal/models"
	"fontManager/internal/scanner"
	"fontManager/internal/store"
	"fontManager/internal/winfont"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type LibraryService struct {
	ctx     context.Context
	store   *store.Store
	scanner *scanner.Scanner
	logDir  string
	mu      sync.Mutex
	running map[int64]bool
}

func NewLibraryService(s *store.Store, logDir ...string) *LibraryService {
	dir := ""
	if len(logDir) > 0 {
		dir = logDir[0]
	}
	return &LibraryService{
		store:   s,
		scanner: scanner.New(s, dir),
		logDir:  dir,
		running: map[int64]bool{},
	}
}

func (s *LibraryService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *LibraryService) ChooseAndAddRoot() (models.LibraryRoot, error) {
	if s.ctx == nil {
		return models.LibraryRoot{}, fmt.Errorf("application context is not ready")
	}
	path, err := runtime.OpenDirectoryDialog(s.ctx, runtime.OpenDialogOptions{Title: "选择字体文件夹"})
	if err != nil {
		return models.LibraryRoot{}, err
	}
	if strings.TrimSpace(path) == "" {
		return models.LibraryRoot{}, fmt.Errorf("no folder selected")
	}
	return s.AddRoot(path)
}

func (s *LibraryService) AddRoot(path string) (models.LibraryRoot, error) {
	root, err := s.store.AddRoot(path)
	if err != nil {
		diagnostics.Appendf(s.logDir, "app.log", "add-root-error path=%q error=%q", path, err.Error())
		return root, err
	}
	diagnostics.Appendf(s.logDir, "app.log", "add-root id=%d path=%q", root.ID, root.Path)
	s.startScan(root)
	return s.store.RootByID(root.ID)
}

func (s *LibraryService) ScanSystemFonts() ([]models.LibraryRoot, error) {
	var roots []models.LibraryRoot
	for _, candidate := range systemFontRoots() {
		if _, err := os.Stat(candidate.path); err != nil {
			continue
		}
		root, err := s.store.AddRootWithKind(candidate.path, "system", candidate.name)
		if err != nil {
			diagnostics.Appendf(s.logDir, "app.log", "scan-system-root-error path=%q error=%q", candidate.path, err.Error())
			return roots, err
		}
		diagnostics.Appendf(s.logDir, "app.log", "scan-system-root id=%d path=%q", root.ID, root.Path)
		s.startScan(root)
		roots = append(roots, root)
	}
	if len(roots) == 0 {
		return roots, fmt.Errorf("未找到可扫描的系统字体目录")
	}
	return roots, nil
}

func (s *LibraryService) ListRoots() ([]models.LibraryRoot, error) {
	return s.store.ListRoots()
}

func (s *LibraryService) RemoveRoot(id int64) error {
	root, err := s.store.RootByID(id)
	if err != nil {
		return err
	}
	if strings.EqualFold(root.Kind, "system") {
		return fmt.Errorf("系统字库不能删除")
	}

	s.mu.Lock()
	running := s.running[id]
	s.mu.Unlock()
	if running {
		return fmt.Errorf("字体库正在扫描，请等待扫描完成后再删除")
	}
	status, err := s.store.LatestScanStatus(id)
	if err != nil {
		return err
	}
	if status.Status == "running" {
		return fmt.Errorf("字体库正在扫描，请等待扫描完成后再删除")
	}
	return s.store.RemoveRoot(id)
}

func (s *LibraryService) RescanRoot(id int64) (models.ScanResult, error) {
	root, err := s.store.RootByID(id)
	if err != nil {
		return models.ScanResult{}, err
	}
	diagnostics.Appendf(s.logDir, "app.log", "rescan-root id=%d path=%q", root.ID, root.Path)
	s.startScan(root)
	return models.ScanResult{RootID: id}, nil
}

func (s *LibraryService) GetScanStatus(rootID int64) (models.ScanStatus, error) {
	return s.store.LatestScanStatus(rootID)
}

func (s *LibraryService) ListFolders(rootID int64) ([]models.FontFolder, error) {
	return s.store.ListFolders(rootID)
}

func (s *LibraryService) startScan(root models.LibraryRoot) {
	s.mu.Lock()
	if s.running[root.ID] {
		s.mu.Unlock()
		return
	}
	s.running[root.ID] = true
	s.mu.Unlock()

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				diagnostics.Appendf(s.logDir, "app.log", "scan-panic root=%d path=%q panic=%v", root.ID, root.Path, recovered)
				if s.ctx != nil {
					runtime.LogWarningf(s.ctx, "scan panic root=%d path=%q panic=%v", root.ID, root.Path, recovered)
				}
			}
			s.mu.Lock()
			delete(s.running, root.ID)
			s.mu.Unlock()
		}()
		diagnostics.Appendf(s.logDir, "app.log", "scan-start root=%d path=%q", root.ID, root.Path)
		_, _ = s.scanner.ScanRoot(root)
		diagnostics.Appendf(s.logDir, "app.log", "scan-stop root=%d path=%q", root.ID, root.Path)
	}()
}

type systemFontRoot struct {
	name string
	path string
}

func systemFontRoots() []systemFontRoot {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	roots := []systemFontRoot{{name: "系统字体", path: filepath.Join(windir, "Fonts")}}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		roots = append(roots, systemFontRoot{name: "当前用户已安装字体", path: filepath.Join(local, "Microsoft", "Windows", "Fonts")})
	}
	return roots
}

type FontService struct {
	ctx      context.Context
	store    *store.Store
	cacheDir string
}

func NewFontService(s *store.Store, cacheDir string) *FontService {
	return &FontService{
		store:    s,
		cacheDir: filepath.Join(cacheDir, "previews"),
	}
}

func (s *FontService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *FontService) SearchFonts(query models.FontQuery) ([]models.FontItem, error) {
	return s.store.QueryFonts(query)
}

func (s *FontService) GetFontDetail(faceID int64) (models.FontDetail, error) {
	return s.store.FontDetail(faceID)
}

func (s *FontService) SetFavorite(faceID int64, favorite bool) error {
	return s.store.SetFavorite(faceID, favorite)
}

func (s *FontService) GetPreview(faceID int64, sampleText string) (models.PreviewResponse, error) {
	start := time.Now()
	file, face, err := s.store.FileForFace(faceID)
	if err != nil {
		return models.PreviewResponse{}, err
	}
	previewText := normalizePreviewSampleText(first(sampleText, face.SampleText, fontmeta.DefaultPreviewSampleText))
	sampleHash := previewSampleHash(previewText)
	response := models.PreviewResponse{
		FaceID:           faceID,
		FontFamily:       fmt.Sprintf("yuncii-preview-%d-%s", faceID, sampleHash[:12]),
		SampleText:       first(face.SampleText, "永字八法 AaBbCc 0123456789"),
		PreviewSupported: face.PreviewSupported,
	}
	response.SampleText = previewText
	if !face.PreviewSupported {
		response.Message = first(face.Error, "This font format cannot be previewed directly.")
		logPreviewInfo(s.ctx, "unsupported", faceID, file.FileName, false, 0, fontmeta.PreviewStats{}, start, response.Message)
		return response, nil
	}

	cacheKey := previewCacheKey(file.Hash, file.Path, face.FaceIndex, sampleHash)
	cachePath := filepath.Join(s.cacheDir, cacheKey+".otf")
	cacheMetaPath := filepath.Join(s.cacheDir, cacheKey+".json")
	if info, err := os.Stat(cachePath); err == nil && info.Size() > 0 {
		stats := readPreviewStats(cacheMetaPath)
		if stats.SubsetBytes == 0 {
			stats.SubsetBytes = info.Size()
		}
		response.CacheHit = true
		response.ByteSize = info.Size()
		applyPreviewStats(&response, stats)
		response.FontURL = "/preview-fonts/" + cacheKey + ".otf"
		response.CSS = previewCSS(response.FontFamily, response.FontURL, face.Weight, face.Italic)
		logPreviewInfo(s.ctx, "cache-hit", faceID, file.FileName, true, response.ByteSize, stats, start, "")
		return response, nil
	}

	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		response.PreviewSupported = false
		response.Message = err.Error()
		logPreviewInfo(s.ctx, "cache-dir-error", faceID, file.FileName, false, 0, fontmeta.PreviewStats{}, start, err.Error())
		return response, nil
	}

	bytes, _, stats, err := fontmeta.PreviewBytes(file.Path, face.FaceIndex, previewText)
	if err != nil {
		response.PreviewSupported = false
		response.Message = err.Error()
		logPreviewInfo(s.ctx, "generate-error", faceID, file.FileName, false, 0, stats, start, err.Error())
		return response, nil
	}
	if err := os.WriteFile(cachePath, bytes, 0o644); err != nil {
		response.PreviewSupported = false
		response.Message = err.Error()
		logPreviewInfo(s.ctx, "write-error", faceID, file.FileName, false, int64(len(bytes)), stats, start, err.Error())
		return response, nil
	}
	if err := writePreviewStats(cacheMetaPath, stats); err != nil && s.ctx != nil {
		runtime.LogWarningf(s.ctx, "preview metadata write failed face=%d file=%q error=%s", faceID, file.FileName, err)
	}

	response.CacheHit = false
	response.ByteSize = int64(len(bytes))
	applyPreviewStats(&response, stats)
	response.FontURL = "/preview-fonts/" + cacheKey + ".otf"
	response.CSS = previewCSS(response.FontFamily, response.FontURL, face.Weight, face.Italic)
	logPreviewInfo(s.ctx, "generated", faceID, file.FileName, false, response.ByteSize, stats, start, "")
	return response, nil
}

func (s *FontService) RevealInExplorer(faceID int64) error {
	file, _, err := s.store.FileForFace(faceID)
	if err != nil {
		return err
	}
	return exec.Command("explorer.exe", "/select,"+file.Path).Start()
}

type InstallService struct {
	ctx   context.Context
	store *store.Store
}

func NewInstallService(s *store.Store) *InstallService {
	return &InstallService{store: s}
}

func (s *InstallService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *InstallService) emitProgress(progress models.OperationProgress) {
	if s.ctx == nil {
		return
	}
	runtime.EventsEmit(s.ctx, "font-operation-progress", progress)
	if progress.Status == "error" {
		runtime.LogWarningf(s.ctx, "install progress status=%s current=%d total=%d face=%d file=%q message=%s",
			progress.Status, progress.Current, progress.Total, progress.FaceID, progress.FileName, progress.Message)
		return
	}
	if progress.Status == "start" || progress.Status == "done" {
		runtime.LogInfof(s.ctx, "install progress status=%s current=%d total=%d succeeded=%d failed=%d",
			progress.Status, progress.Current, progress.Total, progress.Succeeded, progress.Failed)
	}
}

func (s *InstallService) InstallFonts(faceIDs []int64, mode string, scope string) (models.OperationResult, error) {
	result := models.OperationResult{Operation: "install"}
	details, err := s.store.FilesForFaces(faceIDs)
	if err != nil {
		return result, err
	}
	s.emitProgress(models.OperationProgress{
		Operation: "install",
		Mode:      defaultString(mode, "copy"),
		Scope:     defaultString(scope, "user"),
		Total:     len(details),
		Status:    "start",
		Message:   "install started",
	})
	installedFiles := map[int64]winfont.InstallOutcome{}
	installedErrs := map[int64]error{}

	for index, detail := range details {
		s.emitProgress(models.OperationProgress{
			Operation: "install",
			Mode:      defaultString(mode, "copy"),
			Scope:     defaultString(scope, "user"),
			Current:   index + 1,
			Total:     len(details),
			Succeeded: result.Succeeded,
			Failed:    result.Failed,
			FaceID:    detail.FaceID,
			FileID:    detail.FileID,
			FileName:  detail.FileName,
			Status:    "running",
			Message:   "installing",
		})
		outcome, ok := installedFiles[detail.FileID]
		if !ok {
			if err, failed := installedErrs[detail.FileID]; failed {
				addFailure(&result, detail.FaceID, detail.FileID, err.Error())
				s.emitProgress(models.OperationProgress{
					Operation: "install",
					Mode:      defaultString(mode, "copy"),
					Scope:     defaultString(scope, "user"),
					Current:   index + 1,
					Total:     len(details),
					Succeeded: result.Succeeded,
					Failed:    result.Failed,
					FaceID:    detail.FaceID,
					FileID:    detail.FileID,
					FileName:  detail.FileName,
					Status:    "error",
					Message:   err.Error(),
				})
				continue
			}
			outcome, err = winfont.InstallFont(winfont.InstallOptions{
				Mode:     mode,
				Scope:    scope,
				File:     detail,
				FaceName: detail.FullName,
			})
			if err != nil {
				installedErrs[detail.FileID] = err
				addFailure(&result, detail.FaceID, detail.FileID, err.Error())
				_ = s.store.LogOperation("install", scope, mode, detail.FaceID, detail.FileID, "error", err.Error())
				s.emitProgress(models.OperationProgress{
					Operation: "install",
					Mode:      defaultString(mode, "copy"),
					Scope:     defaultString(scope, "user"),
					Current:   index + 1,
					Total:     len(details),
					Succeeded: result.Succeeded,
					Failed:    result.Failed,
					FaceID:    detail.FaceID,
					FileID:    detail.FileID,
					FileName:  detail.FileName,
					Status:    "error",
					Message:   err.Error(),
				})
				continue
			}
			installedFiles[detail.FileID] = outcome
		}
		record := models.InstallRecord{
			FileID:            detail.FileID,
			FaceID:            detail.FaceID,
			SourcePath:        detail.Path,
			TargetPath:        outcome.TargetPath,
			Mode:              defaultString(mode, "copy"),
			Scope:             defaultString(scope, "user"),
			RegistryKey:       outcome.RegistryKey,
			RegistryValueName: outcome.RegistryValueName,
			RegistryValueData: outcome.RegistryValueData,
			Status:            "installed",
		}
		if err := s.store.AddInstallRecord(record); err != nil {
			addFailure(&result, detail.FaceID, detail.FileID, err.Error())
			_ = s.store.LogOperation("install", scope, mode, detail.FaceID, detail.FileID, "error", err.Error())
			s.emitProgress(models.OperationProgress{
				Operation: "install",
				Mode:      defaultString(mode, "copy"),
				Scope:     defaultString(scope, "user"),
				Current:   index + 1,
				Total:     len(details),
				Succeeded: result.Succeeded,
				Failed:    result.Failed,
				FaceID:    detail.FaceID,
				FileID:    detail.FileID,
				FileName:  detail.FileName,
				Status:    "error",
				Message:   err.Error(),
			})
			continue
		}
		addSuccess(&result, detail.FaceID, detail.FileID, fmt.Sprintf("Installed %s", detail.FullName))
		_ = s.store.LogOperation("install", scope, mode, detail.FaceID, detail.FileID, "info", "installed")
		s.emitProgress(models.OperationProgress{
			Operation: "install",
			Mode:      defaultString(mode, "copy"),
			Scope:     defaultString(scope, "user"),
			Current:   index + 1,
			Total:     len(details),
			Succeeded: result.Succeeded,
			Failed:    result.Failed,
			FaceID:    detail.FaceID,
			FileID:    detail.FileID,
			FileName:  detail.FileName,
			Status:    "installed",
			Message:   "installed",
		})
	}
	s.emitProgress(models.OperationProgress{
		Operation: "install",
		Mode:      defaultString(mode, "copy"),
		Scope:     defaultString(scope, "user"),
		Current:   len(details),
		Total:     len(details),
		Succeeded: result.Succeeded,
		Failed:    result.Failed,
		Status:    "done",
		Message:   "install completed",
		Done:      true,
	})
	return result, nil
}

func (s *InstallService) UninstallFonts(faceIDs []int64, deleteCopiedFiles bool) (models.OperationResult, error) {
	result := models.OperationResult{Operation: "uninstall"}
	records, err := s.store.ActiveInstallRecordsForFaces(faceIDs)
	if err != nil {
		return result, err
	}
	if len(records) == 0 {
		for _, id := range faceIDs {
			addFailure(&result, id, 0, "No active install record found for this font")
		}
		return result, nil
	}

	processedTargets := map[string]error{}
	for _, record := range records {
		targetKey := strings.ToLower(filepath.Clean(record.TargetPath)) + "|" + record.RegistryValueName
		err, seen := processedTargets[targetKey]
		if !seen {
			err = winfont.UninstallFont(record, deleteCopiedFiles)
			processedTargets[targetKey] = err
		}
		errText := ""
		if err != nil {
			errText = err.Error()
			addFailure(&result, record.FaceID, record.FileID, errText)
			_ = s.store.LogOperation("uninstall", record.Scope, record.Mode, record.FaceID, record.FileID, "error", errText)
		} else {
			addSuccess(&result, record.FaceID, record.FileID, "Uninstalled")
			_ = s.store.LogOperation("uninstall", record.Scope, record.Mode, record.FaceID, record.FileID, "info", "uninstalled")
		}
		_ = s.store.MarkInstallRecordUninstalled(record.ID, errText)
	}
	return result, nil
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizePreviewSampleText(value string) string {
	text := strings.Join(strings.Fields(value), " ")
	if text == "" {
		return fontmeta.DefaultPreviewSampleText
	}
	return text
}

func previewSampleHash(value string) string {
	sum := sha1.Sum([]byte(normalizePreviewSampleText(value)))
	return hex.EncodeToString(sum[:])
}

func previewCacheKey(hash, path string, faceIndex int, sampleHash string) string {
	source := strings.TrimSpace(strings.ToLower(hash))
	if source == "" {
		sum := sha1.Sum([]byte(strings.ToLower(filepath.Clean(path))))
		source = hex.EncodeToString(sum[:])
	}
	source = sanitizeCacheKey(source)
	if len(source) > 40 {
		source = source[:40]
	}
	sample := sanitizeCacheKey(sampleHash)
	if len(sample) > 20 {
		sample = sample[:20]
	}
	return fmt.Sprintf("v2-%s-%d-%s", source, faceIndex, sample)
}

func sanitizeCacheKey(value string) string {
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func previewCSS(fontFamily, fontURL string, weight int, italic bool) string {
	return fmt.Sprintf("@font-face{font-family:'%s';src:url('%s') format('opentype');font-weight:%d;font-style:%s;font-display:block;}",
		fontFamily, fontURL, weight, cssStyle(italic))
}

func readPreviewStats(path string) fontmeta.PreviewStats {
	data, err := os.ReadFile(path)
	if err != nil {
		return fontmeta.PreviewStats{}
	}
	var stats fontmeta.PreviewStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return fontmeta.PreviewStats{}
	}
	return stats
}

func writePreviewStats(path string, stats fontmeta.PreviewStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func applyPreviewStats(response *models.PreviewResponse, stats fontmeta.PreviewStats) {
	response.GlyphCount = stats.GlyphCount
	response.MissingRuneCount = stats.MissingRuneCount
	response.FullBytes = stats.FullBytes
	response.SubsetBytes = stats.SubsetBytes
	response.Fallback = stats.Fallback
	response.FallbackReason = stats.FallbackReason
	response.ReductionRatio = previewReductionRatio(stats.FullBytes, stats.SubsetBytes)
}

func previewReductionRatio(fullBytes, subsetBytes int64) float64 {
	if fullBytes <= 0 || subsetBytes <= 0 {
		return 0
	}
	return 1 - float64(subsetBytes)/float64(fullBytes)
}

func logPreviewInfo(ctx context.Context, stage string, faceID int64, fileName string, cacheHit bool, byteSize int64, stats fontmeta.PreviewStats, start time.Time, message string) {
	if ctx == nil {
		return
	}
	elapsed := time.Since(start)
	text := fmt.Sprintf("preview %s face=%d file=%q cacheHit=%t bytes=%d elapsed=%s", stage, faceID, fileName, cacheHit, byteSize, elapsed.Round(time.Millisecond))
	if stats.GlyphCount > 0 || stats.MissingRuneCount > 0 || stats.SubsetBytes > 0 || stats.FullBytes > 0 {
		text += fmt.Sprintf(" glyphs=%d missingRunes=%d subsetBytes=%d fullBytes=%d reduction=%.2f fallback=%t",
			stats.GlyphCount, stats.MissingRuneCount, stats.SubsetBytes, stats.FullBytes, previewReductionRatio(stats.FullBytes, stats.SubsetBytes), stats.Fallback)
	}
	if stats.FallbackReason != "" {
		text += " fallbackReason=" + stats.FallbackReason
	}
	if message != "" {
		text += " message=" + message
	}
	if elapsed > 300*time.Millisecond || strings.Contains(stage, "error") {
		runtime.LogWarning(ctx, text)
		return
	}
	runtime.LogInfo(ctx, text)
}

func cssStyle(italic bool) string {
	if italic {
		return "italic"
	}
	return "normal"
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func addSuccess(r *models.OperationResult, faceID, fileID int64, message string) {
	r.Succeeded++
	r.Messages = append(r.Messages, models.OperationMessage{FaceID: faceID, FileID: fileID, Level: "info", Message: message})
}

func addFailure(r *models.OperationResult, faceID, fileID int64, message string) {
	r.Failed++
	r.Messages = append(r.Messages, models.OperationMessage{FaceID: faceID, FileID: fileID, Level: "error", Message: message})
}
