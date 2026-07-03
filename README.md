# Ziio Font Manager

**Ziio Font Manager** is a local-first desktop font manager for Windows 11. It is built with Wails, Go, React, and TypeScript, and is currently in **Beta / v0.3 planning** after completing the MVP stage.

**中文简介：** Ziio Font Manager 是一个面向 Windows 11 的本地字体管理工具。项目已经完成 MVP 验证，当前进入 Beta 阶段，重点转向稳定性、权限体验、发布工程和可维护性。

Ziio focuses on practical local font-library workflows: indexing font folders, scanning installed system fonts, previewing typefaces, searching metadata, collecting favorites, and managing Windows font installation records.

**中文补充：** Ziio 主要解决本地字体库索引、预览、检索、收藏、安装和卸载管理问题。

## Features / 功能

- **Library management:** add user font folders and scan Windows system font locations, shown separately as user libraries and system libraries.
- **Background indexing:** adding, rescanning, and system-font scanning run as background jobs so large folders do not block the UI.
- **Search and filters:** search by family, style, file name, or path; filter by favorites and installed status.
- **Folder navigation:** browse indexed fonts through an embedded folder tree with expandable levels and persisted UI state.
- **Paged result loading:** font results load in pages of 20 and continue automatically when the list is scrolled near the bottom.
- **Font previews:** previews are generated lazily for visible fonts, with custom sample text and font-size controls.
- **Font details:** inspect format, file size, glyph count, version, manufacturer, PostScript name, install records, and error status.
- **Windows font operations:** install by copy, install by link, uninstall, and choose between current-user and all-users scopes.
- **Operation feedback:** installation progress and failures are reported through Wails events and shown in the UI.

中文概览：Ziio 支持用户字体库和系统字库扫描、后台索引、搜索筛选、收藏、文件夹树、触底分页加载、字体预览、详情查看、当前用户 / 所有用户安装范围和操作进度反馈。

## Tech Stack / 技术栈

- Desktop shell: Wails v2.12
- Backend: Go 1.25
- Frontend: React 18, TypeScript 4.6, Vite 3
- Icons: lucide-react
- Database: SQLite via the pure-Go driver `modernc.org/sqlite`
- Font parsing and preview generation: `github.com/tdewolff/font`
- Windows integration: Registry, GDI `AddFontResourceEx` / `RemoveFontResourceEx`, `WM_FONTCHANGE`

## Project Structure / 项目结构

```text
.
├── app.go                       # Wails app initialization, service binding, preview font handler
├── main.go                      # Wails entry point
├── go.mod / go.sum              # Go dependencies
├── wails.json                   # Wails project configuration
├── internal/
│   ├── appdirs/                 # App data/cache directory resolution
│   ├── fontmeta/                # Font format detection, metadata parsing, preview subsetting
│   ├── library/                 # Wails-exposed library, font, and install services
│   ├── models/                  # Shared backend/frontend data models
│   ├── scanner/                 # Folder scanning, incremental indexing, parse timeout protection
│   ├── store/                   # SQLite schema, queries, favorites, install records, scan jobs
│   └── winfont/                 # Windows font install/uninstall, registry, permission boundary
├── frontend/
│   ├── src/                     # React app source
│   ├── wailsjs/                 # Wails-generated frontend bindings
│   └── package.json             # Frontend dependencies and scripts
└── build/                       # Wails build assets and local build output
```

`frontend/node_modules`, `frontend/dist`, and `build/bin` are dependencies or generated build outputs, not core source directories.

中文说明：`frontend/node_modules`、`frontend/dist` 和 `build/bin` 不应作为核心源码提交或维护。

## Data Locations / 数据位置

To avoid losing existing indexed libraries after the brand rename, the current build intentionally keeps the historical data namespace:

- Persistent data: `%APPDATA%\Yuncii\FontManager`
- SQLite database: `%APPDATA%\Yuncii\FontManager\fontmanager.db`
- Preview cache: `%LOCALAPPDATA%\Yuncii\FontManager\cache`
- Preview font files: `%LOCALAPPDATA%\Yuncii\FontManager\cache\previews`

中文说明：为了避免品牌重命名后丢失已有索引，当前版本暂时沿用历史 `Yuncii\FontManager` 数据目录。后续如果迁移到 Ziio 命名空间，需要提供自动迁移或显式导入流程。

## Development / 开发

Run the app in live development mode:

```powershell
wails dev
```

Run backend tests:

```powershell
go test ./...
```

Run the frontend production build check:

```powershell
Set-Location frontend
npm run build
```

Build the Windows executable:

```powershell
wails build -webview2 embed
```

The portable executable is usually generated at:

```text
build\bin\ZiioFontManager.exe
```

中文说明：日常开发使用 `wails dev`；改后端运行 `go test ./...`；改前端运行 `npm run build`；发布前运行 `wails build -webview2 embed`。

## Beta Roadmap / Beta 规划

Ziio is moving from MVP completeness toward Beta reliability. The current priorities are:

- Scanning stability for very large font libraries, including cancellation, retry, and clearer error states.
- Better all-users install/uninstall experience, including UAC elevation flow and safer uninstall policies.
- Preview-cache cleanup, preview-job observability, and clearer fallback messages for legacy or damaged fonts.
- UI polish for empty states, loading states, long paths, long family names, batch selection, and responsive layout.
- Release engineering: version synchronization, build artifact policy, portable/installer packaging, and release checklist.

中文概览：Beta 阶段重点是大字体库稳定性、所有用户安装权限体验、预览缓存治理、UI 细节打磨和发布工程。

## Current Limitations / 当前限制

- All-users install/uninstall currently requires running the app as administrator; an automatic UAC helper is not implemented yet.
- WOFF and WOFF2 are indexed and previewed, but are not installed into Windows by default.
- Type1, FON, FNT, FOT, and other legacy formats are managed on a best-effort basis; metadata and preview support may be limited.
- First-time indexing of very large font libraries can still take time, although it no longer blocks the add-library action.
- A single font metadata parse is capped at 12 seconds; timed-out fonts are marked as errors and skipped so the scan can continue.

中文说明：当前限制主要集中在管理员权限、旧字体格式兼容、超大库首次扫描耗时和坏字体降级处理。
