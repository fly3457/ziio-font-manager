# Ziio Font Manager

## English

**Ziio Font Manager** is a local-first desktop font manager for Windows 11. It is built with Wails, Go, React, and TypeScript, and is currently in **Beta / v0.3 planning**.

Ziio focuses on practical local font-library workflows: indexing font folders, scanning installed system fonts, previewing typefaces, searching metadata, collecting favorites, and managing Windows font installation records.

### Features

- **Library management:** add user font folders and scan Windows system font locations, shown separately as user libraries and system libraries.
- **Background indexing:** adding, rescanning, and system-font scanning run as background jobs so large folders do not block the UI.
- **Search and filters:** search by family, style, file name, or path; filter by favorites and installed status.
- **Folder navigation:** browse indexed fonts through an embedded folder tree with expandable levels and persisted UI state.
- **Paged result loading:** font results load in pages of 20 and continue automatically when the list is scrolled near the bottom.
- **Font previews:** previews are generated lazily for visible fonts, with custom sample text and font-size controls.
- **Font details:** inspect format, file size, glyph count, version, manufacturer, PostScript name, install records, and error status.
- **Windows font operations:** install by copy, install by link, uninstall, and choose between current-user and all-users scopes.
- **Operation feedback:** installation progress and failures are reported through Wails events and shown in the UI.

### Tech Stack

- Desktop shell: Wails v2.12
- Backend: Go 1.25
- Frontend: React 18, TypeScript 4.6, Vite 3
- Icons: lucide-react
- Database: SQLite via the pure-Go driver `modernc.org/sqlite`
- Font parsing and preview generation: `github.com/tdewolff/font`
- Windows integration: Registry, GDI `AddFontResourceEx` / `RemoveFontResourceEx`, `WM_FONTCHANGE`

### Project Structure

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

### Development

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

### Beta Roadmap

Ziio is moving from MVP completeness toward Beta reliability. The current priorities are:

- Scanning stability for very large font libraries, including cancellation, retry, and clearer error states.
- Better all-users install/uninstall experience, including UAC elevation flow and safer uninstall policies.
- Preview-cache cleanup, preview-job observability, and clearer fallback messages for legacy or damaged fonts.
- UI polish for empty states, loading states, long paths, long family names, batch selection, and responsive layout.
- Release engineering: version synchronization, build artifact policy, portable/installer packaging, and release checklist.

### Current Limitations

- All-users install/uninstall currently requires running the app as administrator; an automatic UAC helper is not implemented yet.
- WOFF and WOFF2 are indexed and previewed, but are not installed into Windows by default.
- Type1, FON, FNT, FOT, and other legacy formats are managed on a best-effort basis; metadata and preview support may be limited.
- First-time indexing of very large font libraries can still take time, although it no longer blocks the add-library action.
- A single font metadata parse is capped at 12 seconds; timed-out fonts are marked as errors and skipped so the scan can continue.

## 中文

**Ziio Font Manager** 是一个面向 Windows 11 的本地优先字体管理桌面应用。项目使用 Wails、Go、React 和 TypeScript 构建，当前处于 **Beta / v0.3 planning** 阶段。

Ziio 面向真实的本地字体库管理流程：索引字体文件夹、扫描系统已安装字体、预览字体、检索字体元数据、收藏常用字体，并管理 Windows 字体安装记录。

### 功能

- **字体库管理：** 添加用户字体目录，扫描 Windows 系统字体目录，并按用户字体库和系统字库分开展示。
- **后台索引：** 添加字体库、重新扫描和扫描系统字库都会启动后台任务，避免大目录阻塞界面。
- **搜索和筛选：** 支持按 family、样式、文件名或路径搜索，并支持收藏和已安装状态筛选。
- **文件夹导航：** 通过内嵌文件夹树浏览已索引字体，支持按层级展开，并持久化界面展开状态。
- **分页加载：** 字体结果每页加载 20 条，滚动接近底部时自动追加下一页。
- **字体预览：** 只为可见字体懒生成预览资源，支持自定义样本文字和字号。
- **字体详情：** 展示格式、文件大小、字形数量、版本、厂商、PostScript 名称、安装记录和错误状态。
- **Windows 字体操作：** 支持复制安装、快捷安装、卸载，并可选择当前用户或所有用户安装范围。
- **操作反馈：** 安装进度和失败信息通过 Wails 事件回传，并在界面中展示。

### 技术栈

- 桌面壳：Wails v2.12
- 后端：Go 1.25
- 前端：React 18、TypeScript 4.6、Vite 3
- 图标：lucide-react
- 数据库：SQLite，使用纯 Go 驱动 `modernc.org/sqlite`
- 字体解析与预览生成：`github.com/tdewolff/font`
- Windows 集成：Registry、GDI `AddFontResourceEx` / `RemoveFontResourceEx`、`WM_FONTCHANGE`

### 项目结构

```text
.
├── app.go                       # Wails 应用初始化、服务绑定和预览字体处理
├── main.go                      # Wails 启动入口
├── go.mod / go.sum              # Go 依赖
├── wails.json                   # Wails 项目配置
├── internal/
│   ├── appdirs/                 # 应用数据目录和缓存目录解析
│   ├── fontmeta/                # 字体格式识别、元数据解析、预览子集生成
│   ├── library/                 # 暴露给 Wails 前端的字体库、字体和安装服务
│   ├── models/                  # 前后端共享数据模型
│   ├── scanner/                 # 文件夹扫描、增量索引、解析超时保护
│   ├── store/                   # SQLite schema、查询、收藏、安装记录、扫描任务
│   └── winfont/                 # Windows 字体安装/卸载、注册表和权限边界
├── frontend/
│   ├── src/                     # React 应用源码
│   ├── wailsjs/                 # Wails 生成的前端绑定
│   └── package.json             # 前端依赖和脚本
└── build/                       # Wails 构建资源和本地构建产物
```

`frontend/node_modules`、`frontend/dist` 和 `build/bin` 是依赖或生成产物，不属于核心源码目录。

### 开发

以实时开发模式运行应用：

```powershell
wails dev
```

运行后端测试：

```powershell
go test ./...
```

运行前端生产构建检查：

```powershell
Set-Location frontend
npm run build
```

构建 Windows 可执行文件：

```powershell
wails build -webview2 embed
```

便携版可执行文件通常生成在：

```text
build\bin\ZiioFontManager.exe
```

### Beta 规划

Ziio 当前正在从 MVP 完整性转向 Beta 可靠性。当前优先事项包括：

- 提升超大字体库扫描稳定性，包括取消、重试和更清晰的错误状态。
- 改善所有用户安装/卸载体验，包括 UAC 提权流程和更安全的卸载策略。
- 增加预览缓存清理、预览任务可观测性，并为旧格式或损坏字体提供更清晰的降级说明。
- 打磨空状态、加载状态、长路径、长 family 名称、批量选择和响应式布局。
- 完善发布工程，包括版本同步、构建产物策略、便携包/安装包和发布检查清单。

### 当前限制

- 所有用户范围的安装/卸载当前需要以管理员身份运行应用，自动 UAC helper 尚未实现。
- WOFF 和 WOFF2 会参与索引和预览，但默认不会安装到 Windows 系统字体中。
- Type1、FON、FNT、FOT 等旧格式按尽力管理处理，元数据和预览支持可能受限。
- 首次索引超大字体库仍可能耗时较长，但不会再阻塞添加字体库操作。
- 单个字体元数据解析限制为 12 秒；超时字体会被标记为错误并跳过，避免阻塞整个扫描任务。
