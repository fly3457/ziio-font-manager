# Build Directory

The build directory is used to house all the build files and assets for your application. 

The structure is:

* bin - Output directory
* darwin - macOS specific files
* windows - Windows specific files

## Mac

The `darwin` directory holds files specific to Mac builds.
These may be customised and used as part of the build. To return these files to the default state, simply delete them
and
build with `wails build`.

The directory contains the following files:

- `Info.plist` - the main plist file used for Mac builds. It is used when building using `wails build`.
- `Info.dev.plist` - same as the main plist file but used when building using `wails dev`.

## Windows

The `windows` directory contains the manifest and rc files used when building with `wails build`.
These may be customised for your application. To return these files to the default state, simply delete them and
build with `wails build`.

- `icon.ico` - The icon used for the application. This is used when building using `wails build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer. These are used when building using `wails build`.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.

## 发布许可文件

发布 Ziio 的 Windows 可执行文件时，应在同一发行版或分发包中附上根目录的 `LICENSE` 和 `THIRD_PARTY_NOTICES.md`。第三方声明包含随附字体和当前构建依赖的许可正文；更新依赖或发布新版本前，需要重新核对实际打包组件，不能直接假定旧声明仍然完整。

不要把用户字体库、个人数据库、诊断日志或本地环境配置放入发行包。通过 Ziio 管理的字体不受项目 MIT 许可证覆盖。
