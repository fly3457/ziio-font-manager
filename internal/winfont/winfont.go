package winfont

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"fontManager/internal/fontmeta"
	"fontManager/internal/models"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	fontRegistryPath = `Software\Microsoft\Windows NT\CurrentVersion\Fonts`
	hwndBroadcast    = uintptr(0xffff)
	wmFontChange     = uintptr(0x001D)
	smtoAbortIfHung  = uintptr(0x0002)
)

var (
	gdi32                  = windows.NewLazySystemDLL("gdi32.dll")
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procAddFontResourceEx  = gdi32.NewProc("AddFontResourceExW")
	procRemoveFontResource = gdi32.NewProc("RemoveFontResourceExW")
	procSendMessageTimeout = user32.NewProc("SendMessageTimeoutW")
)

type InstallOptions struct {
	Mode     string
	Scope    string
	File     models.FontDetail
	FaceName string
}

type InstallOutcome struct {
	TargetPath        string
	RegistryKey       string
	RegistryValueName string
	RegistryValueData string
}

func InstallFont(opts InstallOptions) (InstallOutcome, error) {
	mode := strings.ToLower(opts.Mode)
	scope := strings.ToLower(opts.Scope)
	if mode == "" {
		mode = "copy"
	}
	if scope == "" {
		scope = "user"
	}
	if mode != "copy" && mode != "link" {
		return InstallOutcome{}, fmt.Errorf("unsupported install mode: %s", opts.Mode)
	}
	if scope != "user" && scope != "machine" {
		return InstallOutcome{}, fmt.Errorf("unsupported install scope: %s", opts.Scope)
	}
	if !fontmeta.IsSystemInstallable(opts.File.Format) {
		return InstallOutcome{}, fmt.Errorf("%s fonts are indexed for preview/archive but are not installed to Windows by default", opts.File.Format)
	}
	if scope == "machine" && !IsElevated() {
		return InstallOutcome{}, errors.New("machine scope requires administrator privileges; restart as administrator or use current-user install")
	}

	source := opts.File.Path
	target := source
	if mode == "copy" {
		dir, err := FontDir(scope)
		if err != nil {
			return InstallOutcome{}, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return InstallOutcome{}, err
		}
		target = uniqueTargetPath(dir, source, opts.File.Hash)
		if err := copyFile(source, target); err != nil {
			return InstallOutcome{}, err
		}
	}

	valueName := RegistryValueName(opts.File.FullName, opts.File.Format)
	valueData := target
	if scope == "machine" && mode == "copy" {
		valueData = filepath.Base(target)
	}

	root := registry.CURRENT_USER
	access := uint32(registry.SET_VALUE | registry.QUERY_VALUE)
	if scope == "machine" {
		root = registry.LOCAL_MACHINE
		access = registry.SET_VALUE | registry.QUERY_VALUE | registry.WOW64_64KEY
	}
	key, _, err := registry.CreateKey(root, fontRegistryPath, access)
	if err != nil {
		return InstallOutcome{}, err
	}
	defer key.Close()
	if err := key.SetStringValue(valueName, valueData); err != nil {
		return InstallOutcome{}, err
	}

	if err := addFontResource(target); err != nil {
		return InstallOutcome{}, err
	}
	broadcastFontChange()
	return InstallOutcome{
		TargetPath:        target,
		RegistryKey:       registryPath(scope),
		RegistryValueName: valueName,
		RegistryValueData: valueData,
	}, nil
}

func UninstallFont(record models.InstallRecord, deleteCopiedFile bool) error {
	scope := strings.ToLower(record.Scope)
	if scope == "machine" && !IsElevated() {
		return errors.New("machine scope requires administrator privileges")
	}
	if IsProtectedSystemFont(record.TargetPath) {
		return fmt.Errorf("refusing to uninstall protected system font: %s", record.TargetPath)
	}

	_ = removeFontResource(record.TargetPath)
	root := registry.CURRENT_USER
	access := uint32(registry.SET_VALUE | registry.QUERY_VALUE)
	if scope == "machine" {
		root = registry.LOCAL_MACHINE
		access = registry.SET_VALUE | registry.QUERY_VALUE | registry.WOW64_64KEY
	}
	key, err := registry.OpenKey(root, fontRegistryPath, access)
	if err == nil {
		_ = key.DeleteValue(record.RegistryValueName)
		_ = key.Close()
	}
	if deleteCopiedFile && strings.EqualFold(record.Mode, "copy") && isUnderUserFontDir(record.TargetPath) {
		_ = os.Remove(record.TargetPath)
	}
	broadcastFontChange()
	return nil
}

func FontDir(scope string) (string, error) {
	if strings.EqualFold(scope, "machine") {
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		return filepath.Join(windir, "Fonts"), nil
	}
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return filepath.Join(local, "Microsoft", "Windows", "Fonts"), nil
}

func IsElevated() bool {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

func RegistryValueName(fullName, format string) string {
	name := strings.TrimSpace(fullName)
	if name == "" {
		name = "Ziio Font"
	}
	switch strings.ToUpper(format) {
	case "OTF", "OTC":
		return name + " (OpenType)"
	case "FON", "FNT", "FOT":
		return name
	case "TYPE1":
		return name + " (Type 1)"
	default:
		return name + " (TrueType)"
	}
}

func IsProtectedSystemFont(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	systemDir, err := FontDir("machine")
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(systemDir, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}

func registryPath(scope string) string {
	if strings.EqualFold(scope, "machine") {
		return `HKLM\` + fontRegistryPath
	}
	return `HKCU\` + fontRegistryPath
}

func addFontResource(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r1, _, callErr := procAddFontResourceEx.Call(uintptr(unsafe.Pointer(p)), 0, 0)
	if r1 == 0 {
		if callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return fmt.Errorf("AddFontResourceEx failed for %s", path)
	}
	return nil
}

func removeFontResource(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r1, _, callErr := procRemoveFontResource.Call(uintptr(unsafe.Pointer(p)), 0, 0)
	if r1 == 0 && callErr != windows.ERROR_SUCCESS {
		return callErr
	}
	return nil
}

func broadcastFontChange() {
	var result uintptr
	procSendMessageTimeout.Call(hwndBroadcast, wmFontChange, 0, 0, smtoAbortIfHung, 1000, uintptr(unsafe.Pointer(&result)))
}

func uniqueTargetPath(dir, source, hash string) string {
	base := filepath.Base(source)
	prefix := "font"
	if len(hash) >= 10 {
		prefix = hash[:10]
	}
	target := filepath.Join(dir, prefix+"_"+base)
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return target
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 2; ; i++ {
		target = filepath.Join(dir, fmt.Sprintf("%s_%s_%d%s", prefix, stem, i, ext))
		if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
			return target
		}
	}
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func isUnderUserFontDir(path string) bool {
	dir, err := FontDir("user")
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != "." && !strings.HasPrefix(rel, "..")
}
