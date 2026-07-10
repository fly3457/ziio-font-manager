package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"fontManager/internal/models"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type FileUpsert struct {
	RootID           int64
	Path             string
	FileName         string
	Format           string
	Size             int64
	ModifiedAt       string
	Hash             string
	Status           string
	Error            string
	PreviewSupported bool
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func now() string {
	return time.Now().Format(time.RFC3339)
}

func (s *Store) migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS library_roots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'user',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_scan_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS font_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			root_id INTEGER NOT NULL REFERENCES library_roots(id) ON DELETE CASCADE,
			path TEXT NOT NULL UNIQUE,
			file_name TEXT NOT NULL,
			format TEXT NOT NULL,
			size INTEGER NOT NULL,
			modified_at TEXT NOT NULL,
			hash TEXT NOT NULL,
			status TEXT NOT NULL,
			error TEXT NOT NULL DEFAULT '',
			preview_supported INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS font_faces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL REFERENCES font_files(id) ON DELETE CASCADE,
			face_index INTEGER NOT NULL,
			family TEXT NOT NULL,
			style TEXT NOT NULL,
			full_name TEXT NOT NULL,
			postscript_name TEXT NOT NULL,
			weight INTEGER NOT NULL DEFAULT 400,
			italic INTEGER NOT NULL DEFAULT 0,
			glyph_count INTEGER NOT NULL DEFAULT 0,
			sample_text TEXT NOT NULL DEFAULT '',
			manufacturer TEXT NOT NULL DEFAULT '',
			designer TEXT NOT NULL DEFAULT '',
			license TEXT NOT NULL DEFAULT '',
			version TEXT NOT NULL DEFAULT '',
			preview_supported INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			error TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(file_id, face_index)
		);`,
		`CREATE TABLE IF NOT EXISTS favorites (
			face_id INTEGER PRIMARY KEY REFERENCES font_faces(id) ON DELETE CASCADE,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS install_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL REFERENCES font_files(id) ON DELETE CASCADE,
			face_id INTEGER NOT NULL REFERENCES font_faces(id) ON DELETE CASCADE,
			source_path TEXT NOT NULL,
			target_path TEXT NOT NULL,
			mode TEXT NOT NULL,
			scope TEXT NOT NULL,
			registry_key TEXT NOT NULL,
			registry_value_name TEXT NOT NULL,
			registry_value_data TEXT NOT NULL,
			installed_at TEXT NOT NULL,
			uninstalled_at TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			error TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS operation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			operation TEXT NOT NULL,
			scope TEXT NOT NULL,
			mode TEXT NOT NULL,
			face_id INTEGER NOT NULL DEFAULT 0,
			file_id INTEGER NOT NULL DEFAULT 0,
			message TEXT NOT NULL,
			level TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS scan_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			root_id INTEGER NOT NULL REFERENCES library_roots(id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			total INTEGER NOT NULL DEFAULT 0,
			processed INTEGER NOT NULL DEFAULT 0,
			added INTEGER NOT NULL DEFAULT 0,
			updated INTEGER NOT NULL DEFAULT 0,
			failed INTEGER NOT NULL DEFAULT 0,
			missing INTEGER NOT NULL DEFAULT 0,
			unchanged INTEGER NOT NULL DEFAULT 0,
			scope TEXT NOT NULL DEFAULT 'root',
			scope_path TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_font_faces_family ON font_faces(family);`,
		`CREATE INDEX IF NOT EXISTS idx_font_files_root ON font_files(root_id);`,
		`CREATE INDEX IF NOT EXISTS idx_install_active ON install_records(file_id, status, uninstalled_at);`,
	}
	for _, stmt := range schema {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := s.ensureColumn("library_roots", "kind", "TEXT NOT NULL DEFAULT 'user'"); err != nil {
		return err
	}
	if err := s.ensureColumn("scan_jobs", "missing", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("scan_jobs", "unchanged", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureColumn("scan_jobs", "scope", "TEXT NOT NULL DEFAULT 'root'"); err != nil {
		return err
	}
	if err := s.ensureColumn("scan_jobs", "scope_path", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if strings.EqualFold(name, column) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func (s *Store) AddRoot(path string) (models.LibraryRoot, error) {
	return s.AddRootWithKind(path, "user", "")
}

func (s *Store) AddRootWithKind(path, kind, displayName string) (models.LibraryRoot, error) {
	clean, err := filepath.Abs(path)
	if err != nil {
		return models.LibraryRoot{}, err
	}
	clean = filepath.Clean(clean)
	info, err := filepath.EvalSymlinks(clean)
	if err == nil {
		clean = info
	}
	t := now()
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) {
		name = clean
	}
	if strings.TrimSpace(displayName) != "" {
		name = strings.TrimSpace(displayName)
	}
	if strings.TrimSpace(kind) == "" {
		kind = "user"
	}
	if !strings.EqualFold(kind, "system") {
		if err := s.validateUserRootPath(clean); err != nil {
			return models.LibraryRoot{}, err
		}
	}
	_, err = s.db.Exec(`
		INSERT INTO library_roots(path, name, kind, enabled, created_at, updated_at)
		VALUES(?, ?, ?, 1, ?, ?)
		ON CONFLICT(path) DO UPDATE SET enabled = 1, name = excluded.name, kind = excluded.kind, updated_at = excluded.updated_at
	`, clean, name, kind, t, t)
	if err != nil {
		return models.LibraryRoot{}, err
	}
	return s.RootByPath(clean)
}

func (s *Store) validateUserRootPath(path string) error {
	rows, err := s.db.Query(`SELECT path, name FROM library_roots WHERE kind != 'system'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var existingPath, existingName string
		if err := rows.Scan(&existingPath, &existingName); err != nil {
			return err
		}
		if samePath(path, existingPath) {
			continue
		}
		if pathContains(existingPath, path) {
			return fmt.Errorf("该字体库已被已有字体库 %q 覆盖：%s", existingName, existingPath)
		}
		if pathContains(path, existingPath) {
			return fmt.Errorf("已有字体库 %q 位于该路径内：%s，请先移除已有字体库后再添加父目录", existingName, existingPath)
		}
	}
	return rows.Err()
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

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func (s *Store) RootByPath(path string) (models.LibraryRoot, error) {
	var r models.LibraryRoot
	var enabled int
	err := s.db.QueryRow(`
		SELECT r.id, r.path, r.name, r.kind, r.enabled, r.created_at, r.updated_at, r.last_scan_at,
			COUNT(DISTINCT ff.id),
			COALESCE((SELECT sj.status FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 'idle'),
			COALESCE((SELECT sj.total FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0),
			COALESCE((SELECT sj.processed FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0)
		FROM library_roots r
		LEFT JOIN font_files ff ON ff.root_id = r.id AND ff.status != 'missing'
		WHERE r.path = ?
		GROUP BY r.id
	`, path).Scan(&r.ID, &r.Path, &r.Name, &r.Kind, &enabled, &r.CreatedAt, &r.UpdatedAt, &r.LastScanAt, &r.FontCount, &r.ScanStatus, &r.ScanTotal, &r.ScanProcessed)
	r.Enabled = enabled == 1
	return r, err
}

func (s *Store) RootByID(id int64) (models.LibraryRoot, error) {
	var r models.LibraryRoot
	var enabled int
	err := s.db.QueryRow(`
		SELECT r.id, r.path, r.name, r.kind, r.enabled, r.created_at, r.updated_at, r.last_scan_at,
			COUNT(DISTINCT ff.id),
			COALESCE((SELECT sj.status FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 'idle'),
			COALESCE((SELECT sj.total FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0),
			COALESCE((SELECT sj.processed FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0)
		FROM library_roots r
		LEFT JOIN font_files ff ON ff.root_id = r.id AND ff.status != 'missing'
		WHERE r.id = ?
		GROUP BY r.id
	`, id).Scan(&r.ID, &r.Path, &r.Name, &r.Kind, &enabled, &r.CreatedAt, &r.UpdatedAt, &r.LastScanAt, &r.FontCount, &r.ScanStatus, &r.ScanTotal, &r.ScanProcessed)
	r.Enabled = enabled == 1
	return r, err
}

func (s *Store) ListRoots() ([]models.LibraryRoot, error) {
	rows, err := s.db.Query(`
		SELECT r.id, r.path, r.name, r.kind, r.enabled, r.created_at, r.updated_at, r.last_scan_at,
			COUNT(DISTINCT ff.id),
			COALESCE((SELECT sj.status FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 'idle'),
			COALESCE((SELECT sj.total FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0),
			COALESCE((SELECT sj.processed FROM scan_jobs sj WHERE sj.root_id = r.id ORDER BY sj.id DESC LIMIT 1), 0)
		FROM library_roots r
		LEFT JOIN font_files ff ON ff.root_id = r.id AND ff.status != 'missing'
		GROUP BY r.id
		ORDER BY r.kind DESC, r.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roots []models.LibraryRoot
	for rows.Next() {
		var r models.LibraryRoot
		var enabled int
		if err := rows.Scan(&r.ID, &r.Path, &r.Name, &r.Kind, &enabled, &r.CreatedAt, &r.UpdatedAt, &r.LastScanAt, &r.FontCount, &r.ScanStatus, &r.ScanTotal, &r.ScanProcessed); err != nil {
			return nil, err
		}
		r.Enabled = enabled == 1
		roots = append(roots, r)
	}
	return roots, rows.Err()
}

func (s *Store) RemoveRoot(id int64) error {
	_, err := s.db.Exec(`DELETE FROM library_roots WHERE id = ?`, id)
	return err
}

func (s *Store) InterruptRunningScans(message string) error {
	if strings.TrimSpace(message) == "" {
		message = "Scan interrupted by application restart"
	}
	_, err := s.db.Exec(`
		UPDATE scan_jobs
		SET status = 'interrupted', message = ?, finished_at = ?
		WHERE status = 'running'
	`, message, now())
	return err
}

func (s *Store) ListFolders(rootID int64) ([]models.FontFolder, error) {
	root, err := s.RootByID(rootID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
		SELECT ff.path, COUNT(face.id)
		FROM font_files ff
		JOIN font_faces face ON face.file_id = ff.id
		WHERE ff.root_id = ? AND ff.status != 'missing'
		GROUP BY ff.path
	`, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int64{"": 0}
	for rows.Next() {
		var path string
		var faceCount int64
		if err := rows.Scan(&path, &faceCount); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root.Path, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		dir := filepath.Dir(rel)
		if dir == "." {
			dir = ""
		}
		dir = filepath.ToSlash(dir)
		for {
			counts[dir] += faceCount
			if dir == "" {
				break
			}
			parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(dir)))
			if parent == "." {
				parent = ""
			}
			dir = parent
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	folders := make([]models.FontFolder, 0, len(counts))
	for path, count := range counts {
		name := filepath.Base(filepath.FromSlash(path))
		if path == "" {
			name = "全部字体"
		}
		depth := 0
		if path != "" {
			depth = strings.Count(path, "/") + 1
		}
		folders = append(folders, models.FontFolder{
			RootID:    rootID,
			Path:      path,
			Name:      name,
			Depth:     depth,
			FontCount: count,
		})
	}
	sort.Slice(folders, func(i, j int) bool {
		if folders[i].Path == "" {
			return true
		}
		if folders[j].Path == "" {
			return false
		}
		return strings.ToLower(folders[i].Path) < strings.ToLower(folders[j].Path)
	})
	return folders, nil
}

func (s *Store) FileIsCurrent(rootID int64, path string, size int64, modifiedAt string) (bool, error) {
	var currentRootID int64
	var currentSize int64
	var currentModifiedAt string
	var status string
	err := s.db.QueryRow(`SELECT root_id, size, modified_at, status FROM font_files WHERE path = ?`, path).Scan(&currentRootID, &currentSize, &currentModifiedAt, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if currentRootID != rootID {
		return false, nil
	}
	return status != "missing" && currentSize == size && currentModifiedAt == modifiedAt, nil
}

func (s *Store) BeginScan(rootID int64, scopeArgs ...string) (int64, error) {
	scope := "root"
	scopePath := ""
	if len(scopeArgs) > 0 && strings.TrimSpace(scopeArgs[0]) != "" {
		scope = strings.TrimSpace(scopeArgs[0])
	}
	if len(scopeArgs) > 1 {
		scopePath = strings.TrimSpace(scopeArgs[1])
	}
	res, err := s.db.Exec(`INSERT INTO scan_jobs(root_id, status, scope, scope_path, started_at) VALUES(?, 'running', ?, ?, ?)`, rootID, scope, scopePath, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateScan(jobID int64, status string, total, processed, added, updated, failed int, message string, extraCounts ...int) error {
	missing := 0
	unchanged := 0
	if len(extraCounts) > 0 {
		missing = extraCounts[0]
	}
	if len(extraCounts) > 1 {
		unchanged = extraCounts[1]
	}
	finished := ""
	if status != "running" {
		finished = now()
	}
	_, err := s.db.Exec(`
		UPDATE scan_jobs
		SET status = ?, total = ?, processed = ?, added = ?, updated = ?, failed = ?, missing = ?, unchanged = ?, message = ?, finished_at = ?
		WHERE id = ?
	`, status, total, processed, added, updated, failed, missing, unchanged, message, finished, jobID)
	return err
}

func (s *Store) LatestScanStatus(rootID int64) (models.ScanStatus, error) {
	var st models.ScanStatus
	err := s.db.QueryRow(`
		SELECT root_id, status, total, processed, added, updated, failed, missing, unchanged, scope, scope_path, message, started_at, finished_at
		FROM scan_jobs
		WHERE root_id = ?
		ORDER BY id DESC LIMIT 1
	`, rootID).Scan(&st.RootID, &st.Status, &st.Total, &st.Processed, &st.Added, &st.Updated, &st.Failed, &st.Missing, &st.Unchanged, &st.Scope, &st.ScopePath, &st.Message, &st.StartedAt, &st.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ScanStatus{RootID: rootID, Status: "idle", Scope: "root"}, nil
	}
	return st, err
}

func (s *Store) UpsertFontFile(file FileUpsert, faces []models.FontFace) (int64, bool, error) {
	t := now()
	var existingID int64
	err := s.db.QueryRow(`SELECT id FROM font_files WHERE path = ?`, file.Path).Scan(&existingID)
	added := false
	if errors.Is(err, sql.ErrNoRows) {
		res, err := s.db.Exec(`
			INSERT INTO font_files(root_id, path, file_name, format, size, modified_at, hash, status, error, preview_supported, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, file.RootID, file.Path, file.FileName, file.Format, file.Size, file.ModifiedAt, file.Hash, file.Status, file.Error, boolInt(file.PreviewSupported), t, t)
		if err != nil {
			return 0, false, err
		}
		existingID, err = res.LastInsertId()
		if err != nil {
			return 0, false, err
		}
		added = true
	} else if err != nil {
		return 0, false, err
	} else {
		_, err = s.db.Exec(`
			UPDATE font_files
			SET root_id = ?, file_name = ?, format = ?, size = ?, modified_at = ?, hash = ?, status = ?, error = ?, preview_supported = ?, updated_at = ?
			WHERE id = ?
		`, file.RootID, file.FileName, file.Format, file.Size, file.ModifiedAt, file.Hash, file.Status, file.Error, boolInt(file.PreviewSupported), t, existingID)
		if err != nil {
			return 0, false, err
		}
	}

	if err := s.ReplaceFaces(existingID, faces); err != nil {
		return 0, false, err
	}
	return existingID, added, nil
}

func (s *Store) ReplaceFaces(fileID int64, faces []models.FontFace) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	t := now()
	seen := make([]int, 0, len(faces))
	for _, face := range faces {
		seen = append(seen, face.FaceIndex)
		_, err = tx.Exec(`
			INSERT INTO font_faces(
				file_id, face_index, family, style, full_name, postscript_name, weight, italic,
				glyph_count, sample_text, manufacturer, designer, license, version,
				preview_supported, status, error, created_at, updated_at
			)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(file_id, face_index) DO UPDATE SET
				family = excluded.family,
				style = excluded.style,
				full_name = excluded.full_name,
				postscript_name = excluded.postscript_name,
				weight = excluded.weight,
				italic = excluded.italic,
				glyph_count = excluded.glyph_count,
				sample_text = excluded.sample_text,
				manufacturer = excluded.manufacturer,
				designer = excluded.designer,
				license = excluded.license,
				version = excluded.version,
				preview_supported = excluded.preview_supported,
				status = excluded.status,
				error = excluded.error,
				updated_at = excluded.updated_at
		`, fileID, face.FaceIndex, emptyDefault(face.Family, "Unknown Family"), emptyDefault(face.Style, "Regular"),
			emptyDefault(face.FullName, emptyDefault(face.Family, "Unknown Family")), face.PostScriptName,
			defaultWeight(face.Weight), boolInt(face.Italic), face.GlyphCount, face.SampleText,
			face.Manufacturer, face.Designer, face.License, face.Version,
			boolInt(face.PreviewSupported), face.Status, face.Error, t, t)
		if err != nil {
			return err
		}
	}

	if len(seen) == 0 {
		_, err = tx.Exec(`DELETE FROM font_faces WHERE file_id = ?`, fileID)
	} else {
		args := make([]any, 0, len(seen)+1)
		args = append(args, fileID)
		placeholders := make([]string, len(seen))
		for i, idx := range seen {
			placeholders[i] = "?"
			args = append(args, idx)
		}
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM font_faces WHERE file_id = ? AND face_index NOT IN (%s)`, strings.Join(placeholders, ",")), args...)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) MarkMissingFiles(rootID int64, seen map[string]struct{}) (int, error) {
	return s.MarkMissingFilesInScope(rootID, "", seen)
}

func (s *Store) MarkMissingFilesInScope(rootID int64, scopeAbsPath string, seen map[string]struct{}) (int, error) {
	rows, err := s.db.Query(`SELECT id, path FROM font_files WHERE root_id = ? AND status != 'missing'`, rootID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	scopeAbsPath = filepath.Clean(strings.TrimSpace(scopeAbsPath))
	hasScope := scopeAbsPath != "" && scopeAbsPath != "."
	seenKeys := make(map[string]struct{}, len(seen))
	for path := range seen {
		seenKeys[pathKey(path)] = struct{}{}
	}

	var missing []int64
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return 0, err
		}
		if hasScope && !samePath(scopeAbsPath, path) && !pathContains(scopeAbsPath, path) {
			continue
		}
		if _, ok := seenKeys[pathKey(path)]; !ok {
			missing = append(missing, id)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range missing {
		if _, err := s.db.Exec(`UPDATE font_files SET status = 'missing', updated_at = ? WHERE id = ?`, now(), id); err != nil {
			return 0, err
		}
	}
	return len(missing), nil
}

func (s *Store) FinishRootScan(rootID int64) error {
	_, err := s.db.Exec(`UPDATE library_roots SET last_scan_at = ?, updated_at = ? WHERE id = ?`, now(), now(), rootID)
	return err
}

func (s *Store) QueryFonts(q models.FontQuery) ([]models.FontItem, error) {
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 200
	}
	if q.Offset < 0 {
		q.Offset = 0
	}

	where := []string{"ff.status != 'missing'"}
	args := []any{}
	if q.RootID > 0 {
		where = append(where, "ff.root_id = ?")
		args = append(args, q.RootID)
	}
	if q.RootID > 0 && strings.TrimSpace(q.FolderPath) != "" {
		root, err := s.RootByID(q.RootID)
		if err != nil {
			return nil, err
		}
		folderAbs := filepath.Join(root.Path, filepath.Clean(filepath.FromSlash(q.FolderPath)))
		prefix := escapeLike(folderAbs)
		if q.FolderRecursive {
			where = append(where, `ff.path LIKE ? ESCAPE '\'`)
			args = append(args, prefix+`\\%`)
		} else {
			where = append(where, `ff.path LIKE ? ESCAPE '\'`)
			args = append(args, prefix+`\\%`)
		}
	}
	if q.FavoritesOnly {
		where = append(where, "fav.face_id IS NOT NULL")
	}
	if q.InstalledOnly {
		where = append(where, `EXISTS (
			SELECT 1 FROM install_records ir
			WHERE ir.file_id = ff.id AND ir.status = 'installed' AND ir.uninstalled_at = ''
		)`)
	}
	if strings.TrimSpace(q.Query) != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(q.Query)) + "%"
		where = append(where, `(LOWER(face.family) LIKE ? OR LOWER(face.full_name) LIKE ? OR LOWER(face.style) LIKE ? OR LOWER(ff.file_name) LIKE ? OR LOWER(ff.path) LIKE ?)`)
		args = append(args, like, like, like, like, like)
	}
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(`
		SELECT
			face.id, ff.id, ff.root_id, r.path, ff.path, ff.file_name, ff.format,
			face.family, face.style, face.full_name, face.postscript_name,
			face.weight, face.italic,
			CASE WHEN fav.face_id IS NULL THEN 0 ELSE 1 END,
			CASE WHEN EXISTS (
				SELECT 1 FROM install_records ir
				WHERE ir.file_id = ff.id AND ir.status = 'installed' AND ir.uninstalled_at = ''
			) THEN 1 ELSE 0 END,
			face.preview_supported, face.status, COALESCE(NULLIF(face.error, ''), ff.error), ff.updated_at
		FROM font_faces face
		JOIN font_files ff ON ff.id = face.file_id
		JOIN library_roots r ON r.id = ff.root_id
		LEFT JOIN favorites fav ON fav.face_id = face.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY face.family COLLATE NOCASE, face.style COLLATE NOCASE, ff.file_name COLLATE NOCASE
		LIMIT ? OFFSET ?
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.FontItem
	for rows.Next() {
		item, err := scanFontItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFontItem(row scanner) (models.FontItem, error) {
	var item models.FontItem
	var italic, fav, installed, preview int
	err := row.Scan(&item.FaceID, &item.FileID, &item.RootID, &item.RootPath, &item.Path,
		&item.FileName, &item.Format, &item.Family, &item.Style, &item.FullName,
		&item.PostScriptName, &item.Weight, &italic, &fav, &installed, &preview,
		&item.Status, &item.Error, &item.UpdatedAt)
	item.Italic = italic == 1
	item.IsFavorite = fav == 1
	item.IsInstalled = installed == 1
	item.PreviewSupported = preview == 1
	return item, err
}

func (s *Store) FontDetail(faceID int64) (models.FontDetail, error) {
	row := s.db.QueryRow(`
		SELECT
			face.id, ff.id, ff.root_id, r.path, ff.path, ff.file_name, ff.format,
			face.family, face.style, face.full_name, face.postscript_name,
			face.weight, face.italic,
			CASE WHEN fav.face_id IS NULL THEN 0 ELSE 1 END,
			CASE WHEN EXISTS (
				SELECT 1 FROM install_records ir
				WHERE ir.file_id = ff.id AND ir.status = 'installed' AND ir.uninstalled_at = ''
			) THEN 1 ELSE 0 END,
			face.preview_supported, face.status, COALESCE(NULLIF(face.error, ''), ff.error), ff.updated_at,
			ff.size, ff.modified_at, ff.hash, face.sample_text, face.manufacturer, face.designer,
			face.license, face.version, face.glyph_count
		FROM font_faces face
		JOIN font_files ff ON ff.id = face.file_id
		JOIN library_roots r ON r.id = ff.root_id
		LEFT JOIN favorites fav ON fav.face_id = face.id
		WHERE face.id = ?
	`, faceID)

	var d models.FontDetail
	var italic, fav, installed, preview int
	err := row.Scan(&d.FaceID, &d.FileID, &d.RootID, &d.RootPath, &d.Path, &d.FileName, &d.Format,
		&d.Family, &d.Style, &d.FullName, &d.PostScriptName, &d.Weight, &italic,
		&fav, &installed, &preview, &d.Status, &d.Error, &d.UpdatedAt, &d.Size, &d.ModifiedAt,
		&d.Hash, &d.SampleText, &d.Manufacturer, &d.Designer, &d.License, &d.Version, &d.GlyphCount)
	if err != nil {
		return d, err
	}
	d.Italic = italic == 1
	d.IsFavorite = fav == 1
	d.IsInstalled = installed == 1
	d.PreviewSupported = preview == 1
	records, err := s.InstallRecordsForFace(faceID)
	if err != nil {
		return d, err
	}
	d.InstallRecords = records
	return d, nil
}

func (s *Store) SetFavorite(faceID int64, favorite bool) error {
	if favorite {
		_, err := s.db.Exec(`INSERT OR IGNORE INTO favorites(face_id, created_at) VALUES(?, ?)`, faceID, now())
		return err
	}
	_, err := s.db.Exec(`DELETE FROM favorites WHERE face_id = ?`, faceID)
	return err
}

func (s *Store) FileForFace(faceID int64) (models.FontFile, models.FontFace, error) {
	var file models.FontFile
	var face models.FontFace
	var filePreview, facePreview, italic int
	err := s.db.QueryRow(`
		SELECT
			ff.id, ff.root_id, ff.path, ff.file_name, ff.format, ff.size, ff.modified_at,
			ff.hash, ff.status, ff.error, ff.preview_supported,
			face.id, face.file_id, face.face_index, face.family, face.style, face.full_name,
			face.postscript_name, face.weight, face.italic, face.glyph_count, face.sample_text,
			face.manufacturer, face.designer, face.license, face.version, face.preview_supported,
			face.status, face.error
		FROM font_faces face
		JOIN font_files ff ON ff.id = face.file_id
		WHERE face.id = ?
	`, faceID).Scan(&file.ID, &file.RootID, &file.Path, &file.FileName, &file.Format, &file.Size,
		&file.ModifiedAt, &file.Hash, &file.Status, &file.Error, &filePreview,
		&face.ID, &face.FileID, &face.FaceIndex, &face.Family, &face.Style, &face.FullName,
		&face.PostScriptName, &face.Weight, &italic, &face.GlyphCount, &face.SampleText,
		&face.Manufacturer, &face.Designer, &face.License, &face.Version, &facePreview,
		&face.Status, &face.Error)
	file.PreviewSupported = filePreview == 1
	face.Italic = italic == 1
	face.PreviewSupported = facePreview == 1
	return file, face, err
}

func (s *Store) FilesForFaces(faceIDs []int64) ([]models.FontDetail, error) {
	if len(faceIDs) == 0 {
		return nil, nil
	}
	details := make([]models.FontDetail, 0, len(faceIDs))
	for _, id := range faceIDs {
		d, err := s.FontDetail(id)
		if err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, nil
}

func (s *Store) AddInstallRecord(r models.InstallRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO install_records(
			file_id, face_id, source_path, target_path, mode, scope, registry_key,
			registry_value_name, registry_value_data, installed_at, uninstalled_at, status, error
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.FileID, r.FaceID, r.SourcePath, r.TargetPath, r.Mode, r.Scope, r.RegistryKey,
		r.RegistryValueName, r.RegistryValueData, now(), r.UninstalledAt, r.Status, r.Error)
	return err
}

func (s *Store) InstallRecordsForFace(faceID int64) ([]models.InstallRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, file_id, face_id, source_path, target_path, mode, scope, registry_key,
			registry_value_name, registry_value_data, installed_at, uninstalled_at, status, error
		FROM install_records
		WHERE face_id = ?
		ORDER BY id DESC
	`, faceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallRecords(rows)
}

func (s *Store) ActiveInstallRecordsForFaces(faceIDs []int64) ([]models.InstallRecord, error) {
	if len(faceIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(faceIDs))
	args := make([]any, len(faceIDs))
	for i, id := range faceIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := s.db.Query(`
		SELECT id, file_id, face_id, source_path, target_path, mode, scope, registry_key,
			registry_value_name, registry_value_data, installed_at, uninstalled_at, status, error
		FROM install_records
		WHERE status = 'installed' AND uninstalled_at = '' AND face_id IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallRecords(rows)
}

func scanInstallRecords(rows *sql.Rows) ([]models.InstallRecord, error) {
	var records []models.InstallRecord
	for rows.Next() {
		var r models.InstallRecord
		if err := rows.Scan(&r.ID, &r.FileID, &r.FaceID, &r.SourcePath, &r.TargetPath,
			&r.Mode, &r.Scope, &r.RegistryKey, &r.RegistryValueName, &r.RegistryValueData,
			&r.InstalledAt, &r.UninstalledAt, &r.Status, &r.Error); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) MarkInstallRecordUninstalled(id int64, errText string) error {
	status := "uninstalled"
	if errText != "" {
		status = "error"
	}
	_, err := s.db.Exec(`UPDATE install_records SET status = ?, error = ?, uninstalled_at = ? WHERE id = ?`, status, errText, now(), id)
	return err
}

func (s *Store) LogOperation(operation, scope, mode string, faceID, fileID int64, level, message string) error {
	_, err := s.db.Exec(`
		INSERT INTO operation_logs(operation, scope, mode, face_id, file_id, message, level, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
	`, operation, scope, mode, faceID, fileID, message, level, now())
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func emptyDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultWeight(v int) int {
	if v <= 0 {
		return 400
	}
	return v
}

func escapeLike(v string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(v)
}
