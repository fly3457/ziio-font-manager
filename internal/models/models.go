package models

type AppInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	DataDir      string `json:"dataDir"`
	CacheDir     string `json:"cacheDir"`
	LogDir       string `json:"logDir"`
	DatabasePath string `json:"databasePath"`
}

type LibraryRoot struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	LastScanAt    string `json:"lastScanAt"`
	FontCount     int64  `json:"fontCount"`
	ScanStatus    string `json:"scanStatus"`
	ScanTotal     int    `json:"scanTotal"`
	ScanProcessed int    `json:"scanProcessed"`
}

type FontFolder struct {
	RootID    int64  `json:"rootId"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Depth     int    `json:"depth"`
	FontCount int64  `json:"fontCount"`
}

type FontFile struct {
	ID               int64  `json:"id"`
	RootID           int64  `json:"rootId"`
	Path             string `json:"path"`
	FileName         string `json:"fileName"`
	Format           string `json:"format"`
	Size             int64  `json:"size"`
	ModifiedAt       string `json:"modifiedAt"`
	Hash             string `json:"hash"`
	Status           string `json:"status"`
	Error            string `json:"error"`
	PreviewSupported bool   `json:"previewSupported"`
}

type FontFace struct {
	ID               int64  `json:"id"`
	FileID           int64  `json:"fileId"`
	FaceIndex        int    `json:"faceIndex"`
	Family           string `json:"family"`
	Style            string `json:"style"`
	FullName         string `json:"fullName"`
	PostScriptName   string `json:"postScriptName"`
	Weight           int    `json:"weight"`
	Italic           bool   `json:"italic"`
	GlyphCount       int    `json:"glyphCount"`
	SampleText       string `json:"sampleText"`
	Manufacturer     string `json:"manufacturer"`
	Designer         string `json:"designer"`
	License          string `json:"license"`
	Version          string `json:"version"`
	PreviewSupported bool   `json:"previewSupported"`
	Status           string `json:"status"`
	Error            string `json:"error"`
}

type FontItem struct {
	FaceID           int64  `json:"faceId"`
	FileID           int64  `json:"fileId"`
	RootID           int64  `json:"rootId"`
	RootPath         string `json:"rootPath"`
	Path             string `json:"path"`
	FileName         string `json:"fileName"`
	Format           string `json:"format"`
	Family           string `json:"family"`
	Style            string `json:"style"`
	FullName         string `json:"fullName"`
	PostScriptName   string `json:"postScriptName"`
	Weight           int    `json:"weight"`
	Italic           bool   `json:"italic"`
	IsFavorite       bool   `json:"isFavorite"`
	IsInstalled      bool   `json:"isInstalled"`
	PreviewSupported bool   `json:"previewSupported"`
	Status           string `json:"status"`
	Error            string `json:"error"`
	UpdatedAt        string `json:"updatedAt"`
}

type FontDetail struct {
	FontItem
	Size           int64           `json:"size"`
	ModifiedAt     string          `json:"modifiedAt"`
	Hash           string          `json:"hash"`
	SampleText     string          `json:"sampleText"`
	Manufacturer   string          `json:"manufacturer"`
	Designer       string          `json:"designer"`
	License        string          `json:"license"`
	Version        string          `json:"version"`
	GlyphCount     int             `json:"glyphCount"`
	InstallRecords []InstallRecord `json:"installRecords"`
}

type FontQuery struct {
	Query           string `json:"query"`
	RootID          int64  `json:"rootId"`
	FolderPath      string `json:"folderPath"`
	FolderRecursive bool   `json:"folderRecursive"`
	FavoritesOnly   bool   `json:"favoritesOnly"`
	InstalledOnly   bool   `json:"installedOnly"`
	Limit           int    `json:"limit"`
	Offset          int    `json:"offset"`
}

type FontStats struct {
	FavoriteCount  int64 `json:"favoriteCount"`
	InstalledCount int64 `json:"installedCount"`
}

type PreviewResponse struct {
	FaceID           int64   `json:"faceId"`
	FontFamily       string  `json:"fontFamily"`
	CSS              string  `json:"css"`
	FontURL          string  `json:"fontUrl"`
	SampleText       string  `json:"sampleText"`
	PreviewSupported bool    `json:"previewSupported"`
	Message          string  `json:"message"`
	CacheHit         bool    `json:"cacheHit"`
	ByteSize         int64   `json:"byteSize"`
	GlyphCount       int     `json:"glyphCount"`
	MissingRuneCount int     `json:"missingRuneCount"`
	FullBytes        int64   `json:"fullBytes"`
	SubsetBytes      int64   `json:"subsetBytes"`
	Fallback         bool    `json:"fallback"`
	FallbackReason   string  `json:"fallbackReason"`
	ReductionRatio   float64 `json:"reductionRatio"`
}

type ScanStatus struct {
	RootID     int64  `json:"rootId"`
	Status     string `json:"status"`
	Total      int    `json:"total"`
	Processed  int    `json:"processed"`
	Added      int    `json:"added"`
	Updated    int    `json:"updated"`
	Failed     int    `json:"failed"`
	Missing    int    `json:"missing"`
	Unchanged  int    `json:"unchanged"`
	Scope      string `json:"scope"`
	ScopePath  string `json:"scopePath"`
	Message    string `json:"message"`
	StartedAt  string `json:"startedAt"`
	FinishedAt string `json:"finishedAt"`
}

type ScanResult struct {
	RootID    int64  `json:"rootId"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Added     int    `json:"added"`
	Updated   int    `json:"updated"`
	Failed    int    `json:"failed"`
	Missing   int    `json:"missing"`
	Unchanged int    `json:"unchanged"`
	Scope     string `json:"scope"`
	ScopePath string `json:"scopePath"`
}

type InstallRecord struct {
	ID                int64  `json:"id"`
	FileID            int64  `json:"fileId"`
	FaceID            int64  `json:"faceId"`
	SourcePath        string `json:"sourcePath"`
	TargetPath        string `json:"targetPath"`
	Mode              string `json:"mode"`
	Scope             string `json:"scope"`
	RegistryKey       string `json:"registryKey"`
	RegistryValueName string `json:"registryValueName"`
	RegistryValueData string `json:"registryValueData"`
	InstalledAt       string `json:"installedAt"`
	UninstalledAt     string `json:"uninstalledAt"`
	Status            string `json:"status"`
	Error             string `json:"error"`
}

type OperationMessage struct {
	FaceID  int64  `json:"faceId"`
	FileID  int64  `json:"fileId"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type OperationResult struct {
	Operation string             `json:"operation"`
	Succeeded int                `json:"succeeded"`
	Failed    int                `json:"failed"`
	Messages  []OperationMessage `json:"messages"`
}

type OperationProgress struct {
	Operation string `json:"operation"`
	Mode      string `json:"mode"`
	Scope     string `json:"scope"`
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
	FaceID    int64  `json:"faceId"`
	FileID    int64  `json:"fileId"`
	FileName  string `json:"fileName"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Done      bool   `json:"done"`
}
