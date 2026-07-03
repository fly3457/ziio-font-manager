package fontmeta

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"fontManager/internal/models"

	tfont "github.com/tdewolff/font"
)

var modernExts = map[string]string{
	".ttf":   "TTF",
	".otf":   "OTF",
	".ttc":   "TTC",
	".otc":   "OTC",
	".woff":  "WOFF",
	".woff2": "WOFF2",
	".eot":   "EOT",
}

var legacyExts = map[string]string{
	".pfm": "TYPE1",
	".pfb": "TYPE1",
	".mmm": "TYPE1",
	".afm": "TYPE1",
	".fon": "FON",
	".fnt": "FNT",
	".fot": "FOT",
}

const DefaultPreviewSampleText = "永字八法 AaBbCc 0123456789"

type ParsedFile struct {
	File  models.FontFile
	Faces []models.FontFace
}

type PreviewStats struct {
	GlyphCount       int
	MissingRuneCount int
	FullBytes        int64
	SubsetBytes      int64
	Fallback         bool
	FallbackReason   string
}

func IsSupportedPath(path string) bool {
	_, ok := FormatFromPath(path)
	return ok
}

func FormatFromPath(path string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if f, ok := modernExts[ext]; ok {
		return f, true
	}
	if f, ok := legacyExts[ext]; ok {
		return f, true
	}
	return "", false
}

func IsSystemInstallable(format string) bool {
	switch strings.ToUpper(format) {
	case "TTF", "OTF", "TTC", "OTC", "FON", "FNT", "FOT", "TYPE1":
		return true
	default:
		return false
	}
}

func IsPreviewable(format string) bool {
	switch strings.ToUpper(format) {
	case "TTF", "OTF", "TTC", "OTC", "WOFF", "WOFF2", "EOT":
		return true
	default:
		return false
	}
}

func ParseFile(path string) (ParsedFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ParsedFile{}, err
	}
	format, ok := FormatFromPath(path)
	if !ok {
		return ParsedFile{}, fmt.Errorf("unsupported font format: %s", filepath.Ext(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ParsedFile{}, err
	}
	hash := sha256.Sum256(data)
	file := models.FontFile{
		Path:             path,
		FileName:         filepath.Base(path),
		Format:           format,
		Size:             info.Size(),
		ModifiedAt:       info.ModTime().Format("2006-01-02T15:04:05-07:00"),
		Hash:             hex.EncodeToString(hash[:]),
		Status:           "ok",
		PreviewSupported: IsPreviewable(format),
	}

	if _, ok := modernExts[strings.ToLower(filepath.Ext(path))]; ok {
		faces, parseErr := parseModern(path, data, format)
		if parseErr != nil {
			file.Status = "error"
			file.Error = parseErr.Error()
			return ParsedFile{File: file, Faces: []models.FontFace{fallbackFace(path, format, parseErr)}}, nil
		}
		return ParsedFile{File: file, Faces: faces}, nil
	}

	face := fallbackFace(path, format, errors.New("legacy format: preview and metadata are limited"))
	face.Status = "limited"
	face.PreviewSupported = false
	file.Status = "limited"
	file.Error = "Legacy font format. Install/uninstall management is available where Windows supports it; preview may be unavailable."
	file.PreviewSupported = false
	return ParsedFile{File: file, Faces: []models.FontFace{face}}, nil
}

func ErrorFile(path string, parseErr error) (ParsedFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ParsedFile{}, err
	}
	format, ok := FormatFromPath(path)
	if !ok {
		return ParsedFile{}, fmt.Errorf("unsupported font format: %s", filepath.Ext(path))
	}
	message := ""
	if parseErr != nil {
		message = parseErr.Error()
	}
	file := models.FontFile{
		Path:             path,
		FileName:         filepath.Base(path),
		Format:           format,
		Size:             info.Size(),
		ModifiedAt:       info.ModTime().Format("2006-01-02T15:04:05-07:00"),
		Hash:             "",
		Status:           "error",
		Error:            message,
		PreviewSupported: false,
	}
	face := fallbackFace(path, format, parseErr)
	face.Status = "error"
	face.PreviewSupported = false
	return ParsedFile{File: file, Faces: []models.FontFace{face}}, nil
}

func PreviewBytes(path string, index int, sampleText string) ([]byte, string, PreviewStats, error) {
	stats := PreviewStats{}
	format, ok := FormatFromPath(path)
	if !ok || !IsPreviewable(format) {
		return nil, "", stats, fmt.Errorf("preview is not supported for %s", filepath.Ext(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", stats, err
	}
	sfnt, err := tfont.ParseSFNT(data, index)
	if err != nil {
		return nil, "", stats, err
	}
	full := sfnt.Write()
	if len(full) == 0 {
		return nil, "", stats, errors.New("font parser produced empty preview data")
	}
	stats.FullBytes = int64(len(full))

	glyphIDs, missing := previewGlyphIDs(sfnt, sampleText)
	stats.GlyphCount = len(glyphIDs)
	stats.MissingRuneCount = missing
	if len(glyphIDs) <= 1 {
		stats.Fallback = true
		stats.FallbackReason = "sample text has no supported glyphs"
		stats.SubsetBytes = int64(len(full))
		return full, "font/otf", stats, nil
	}

	subset, err := sfnt.Subset(glyphIDs, tfont.SubsetOptions{Tables: tfont.KeepMinTables})
	if err != nil {
		stats.Fallback = true
		stats.FallbackReason = err.Error()
		stats.SubsetBytes = int64(len(full))
		return full, "font/otf", stats, nil
	}
	out := subset.Write()
	if len(out) == 0 {
		stats.Fallback = true
		stats.FallbackReason = "subset writer produced empty data"
		stats.SubsetBytes = int64(len(full))
		return full, "font/otf", stats, nil
	}
	if _, err := tfont.ParseSFNT(out, 0); err != nil {
		stats.Fallback = true
		stats.FallbackReason = "subset verification failed: " + err.Error()
		stats.SubsetBytes = int64(len(full))
		return full, "font/otf", stats, nil
	}

	stats.SubsetBytes = int64(len(out))
	return out, "font/otf", stats, nil
}

func previewGlyphIDs(sfnt *tfont.SFNT, sampleText string) ([]uint16, int) {
	seenGlyphs := map[uint16]bool{0: true}
	seenRunes := map[rune]bool{}
	glyphIDs := []uint16{0}
	missing := 0
	for _, r := range previewSampleText(sampleText) {
		if r < 0x20 {
			continue
		}
		if seenRunes[r] {
			continue
		}
		seenRunes[r] = true
		glyphID := sfnt.GlyphIndex(r)
		if glyphID == 0 {
			missing++
			continue
		}
		if seenGlyphs[glyphID] {
			continue
		}
		seenGlyphs[glyphID] = true
		glyphIDs = append(glyphIDs, glyphID)
	}
	return glyphIDs, missing
}

func previewSampleText(sampleText string) string {
	text := strings.Join(strings.Fields(sampleText), " ")
	if text == "" {
		return DefaultPreviewSampleText
	}
	return text
}

func parseModern(path string, data []byte, format string) ([]models.FontFace, error) {
	maxFaces := 1
	if isCollection(data, format) {
		maxFaces = 128
	}

	var faces []models.FontFace
	for i := 0; i < maxFaces; i++ {
		sfnt, err := tfont.ParseSFNT(data, i)
		if err != nil {
			if i == 0 {
				return nil, err
			}
			break
		}
		names := parseNameTable(sfnt.Tables["name"])
		family := first(names[tfont.NamePreferredFamily], names[tfont.NameWWSFamily], names[tfont.NameFontFamily], trimExt(filepath.Base(path)))
		style := first(names[tfont.NamePreferredSubfamily], names[tfont.NameWWSSubfamily], names[tfont.NameFontSubfamily], "Regular")
		fullName := first(names[tfont.NameFull], names[tfont.NameCompatibleFull], strings.TrimSpace(family+" "+style), family)
		sample := first(names[tfont.NameSampleText], DefaultPreviewSampleText)
		weight := inferWeight(style + " " + fullName)
		italic := strings.Contains(strings.ToLower(style+" "+fullName), "italic") || strings.Contains(strings.ToLower(style+" "+fullName), "oblique")

		faces = append(faces, models.FontFace{
			FaceIndex:        i,
			Family:           family,
			Style:            style,
			FullName:         fullName,
			PostScriptName:   first(names[tfont.NamePostScript], sanitizePostScript(fullName)),
			Weight:           weight,
			Italic:           italic,
			GlyphCount:       int(sfnt.NumGlyphs()),
			SampleText:       sample,
			Manufacturer:     names[tfont.NameManufacturer],
			Designer:         names[tfont.NameDesigner],
			License:          names[tfont.NameLicense],
			Version:          names[tfont.NameVersion],
			PreviewSupported: true,
			Status:           "ok",
		})
	}
	if len(faces) == 0 {
		return nil, errors.New("no faces found in font file")
	}
	return faces, nil
}

func fallbackFace(path, format string, err error) models.FontFace {
	name := trimExt(filepath.Base(path))
	message := ""
	if err != nil {
		message = err.Error()
	}
	return models.FontFace{
		FaceIndex:        0,
		Family:           name,
		Style:            "Regular",
		FullName:         name,
		PostScriptName:   sanitizePostScript(name),
		Weight:           400,
		SampleText:       DefaultPreviewSampleText,
		PreviewSupported: IsPreviewable(format),
		Status:           "limited",
		Error:            message,
	}
}

func parseNameTable(table []byte) map[tfont.NameID]string {
	names := map[tfont.NameID]string{}
	scores := map[tfont.NameID]int{}
	if len(table) < 6 {
		return names
	}
	count := int(binary.BigEndian.Uint16(table[2:4]))
	stringOffset := int(binary.BigEndian.Uint16(table[4:6]))
	for i := 0; i < count; i++ {
		off := 6 + i*12
		if off+12 > len(table) {
			break
		}
		platform := binary.BigEndian.Uint16(table[off : off+2])
		encoding := binary.BigEndian.Uint16(table[off+2 : off+4])
		language := binary.BigEndian.Uint16(table[off+4 : off+6])
		nameID := tfont.NameID(binary.BigEndian.Uint16(table[off+6 : off+8]))
		length := int(binary.BigEndian.Uint16(table[off+8 : off+10]))
		valueOffset := int(binary.BigEndian.Uint16(table[off+10 : off+12]))
		start := stringOffset + valueOffset
		end := start + length
		if start < 0 || end > len(table) || length == 0 {
			continue
		}
		value := decodeNameString(platform, encoding, table[start:end])
		value = strings.TrimSpace(strings.ReplaceAll(value, "\x00", ""))
		if value == "" {
			continue
		}
		score := nameScore(platform, language)
		if existing, ok := scores[nameID]; !ok || score > existing {
			scores[nameID] = score
			names[nameID] = value
		}
	}
	return names
}

func decodeNameString(platform, encoding uint16, b []byte) string {
	if platform == 0 || platform == 3 || (platform == 2 && encoding == 1) {
		if len(b)%2 == 1 {
			b = b[:len(b)-1]
		}
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(b[i*2 : i*2+2])
		}
		return string(utf16.Decode(u))
	}
	return string(b)
}

func nameScore(platform, language uint16) int {
	score := 1
	switch platform {
	case 3:
		score = 100
	case 0:
		score = 90
	case 1:
		score = 50
	}
	if language == 0x0409 {
		score += 5
	}
	return score
}

func isCollection(data []byte, format string) bool {
	if strings.EqualFold(format, "TTC") || strings.EqualFold(format, "OTC") {
		return true
	}
	return len(data) >= 4 && string(data[:4]) == "ttcf"
}

func inferWeight(text string) int {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "black") || strings.Contains(lower, "heavy"):
		return 900
	case strings.Contains(lower, "extra bold") || strings.Contains(lower, "extrabold") || strings.Contains(lower, "ultra bold") || strings.Contains(lower, "ultrabold"):
		return 800
	case strings.Contains(lower, "bold"):
		return 700
	case strings.Contains(lower, "semibold") || strings.Contains(lower, "semi bold") || strings.Contains(lower, "demibold") || strings.Contains(lower, "demi bold"):
		return 600
	case strings.Contains(lower, "medium"):
		return 500
	case strings.Contains(lower, "light"):
		return 300
	case strings.Contains(lower, "thin") || strings.Contains(lower, "hairline"):
		return 100
	default:
		return 400
	}
}

func first(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func trimExt(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func sanitizePostScript(v string) string {
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-", ":", "-", ";", "-")
	return replacer.Replace(strings.TrimSpace(v))
}
