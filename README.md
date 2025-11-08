# Code Switch

AI relay manager for Claude & Codex providers.  
Builds with [Wails 3](https://v3.wails.io).

## Prerequisites

- Go 1.24+
- Node.js 18+
- npm / pnpm / yarn (project uses npm scripts)
- Wails 3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)

## Development

```bash
wails3 task dev
```

This installs frontend deps, runs the Vite dev server and Go backend in watch mode.

## Build

Before building, ensure the desktop bundle metadata (company, product name, etc.) is synchronized:

```bash
# Update build assets (Info.plist, icons, etc.) after editing build/config.yml
wails3 task common:update:build-assets

# Produce binaries + .app bundle
wails3 task build
```

The macOS app bundle is generated at `./bin/codeswitch.app`.

### Cross-compile (macOS ➜ Windows)

1. Install mingw-w64:
   ```bash
   brew install mingw-w64
   ```
2. Update build assets (if you changed `build/config.yml`):
   ```bash
   wails3 task common:update:build-assets
   ```
3. Build Windows binaries from macOS using the Windows task:
   ```bash
   wails3 task windows:build ARCH=amd64
   ```
   - Output: `./bin/codeswitch.exe` + supporting files.
   - To produce the NSIS installer, run the `windows:package` task with the same environment variables.

### Publish a Release

Use the helper script to build and upload assets to GitHub Releases (requires the `gh` CLI):

```bash
# Build macOS .app + Windows installer and publish (defaults to v0.1.0 / RELEASE_NOTES.md)
scripts/publish_release.sh
```

The script:
- runs `wails3 task common:update:build-assets`
- builds macOS (`bin/codeswitch.app`) via `wails3 task package`
- cross-compiles + packages Windows installer (`bin/codeswitch-*-installer.exe`) via `windows:package`
- zips `codeswitch.app` into `codeswitch-macos.zip`, then uploads zip + installer to GitHub Releases

Want to run the steps manually? Execute:

```bash
wails3 task package
wails3 task windows:package ARCH=amd64
ditto -c -k --sequesterRsrc --keepParent bin/codeswitch.app bin/codeswitch-macos.zip
```

## Packaging Notes

If `codeswitch.app` fails to open because the executable is “missing”, it usually means the Info.plist was out of sync. Re-run the `common:update:build-assets` task, then rebuild as shown above.
