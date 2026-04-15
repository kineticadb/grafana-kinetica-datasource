# Creating a Deployment Archive for the Grafana Plugin


To create a deployment archive for your Grafana plugin, you need to compile the source code, gather the artifacts, and zip them up. This guide provides a step-by-step process and a breakdown of what the archive will contain.

## 1. Build the Plugin

You cannot zip the source code (`src/`) directly. Grafana requires the compiled assets located in the `dist/` folder.

### 1.1 Install Dependencies

```bash
npm install
# or
yarn install
```

### 1.2 Build Frontend & Backend

Since your plugin has a Go backend, you need to build both parts. You can use either npm or mage directly:

```bash
# Using npm (triggers webpack for frontend)
npm run build

# Using mage (builds both frontend and all backend binaries)
mage
```

For more granular control over the build process, see the "How to Use Mage" section below.

## Installing Mage

Mage is required to build the Go backend. Install it using one of the following methods:

### Using Go (if Go is installed)

```bash
go install github.com/magefile/mage@latest
```

Ensure `$GOPATH/bin` is in your `PATH`.

### Using Homebrew (macOS)

```bash
brew install mage
```

### Using Pre-built Binaries (No Go Required)

For systems without Go installed, download pre-built binaries from the [Mage releases page](https://github.com/magefile/mage/releases):

**Linux (amd64):**

```bash
curl -L https://github.com/magefile/mage/releases/download/v1.15.0/mage_1.15.0_Linux-64bit.tar.gz | tar xz
sudo mv mage /usr/local/bin/
```

**Linux (arm64):**

```bash
curl -L https://github.com/magefile/mage/releases/download/v1.15.0/mage_1.15.0_Linux-ARM64.tar.gz | tar xz
sudo mv mage /usr/local/bin/
```

**macOS (Intel):**

```bash
curl -L https://github.com/magefile/mage/releases/download/v1.15.0/mage_1.15.0_macOS-64bit.tar.gz | tar xz
sudo mv mage /usr/local/bin/
```

**macOS (Apple Silicon):**

```bash
curl -L https://github.com/magefile/mage/releases/download/v1.15.0/mage_1.15.0_macOS-ARM64.tar.gz | tar xz
sudo mv mage /usr/local/bin/
```

**Windows:**

1. Download `mage_1.15.0_Windows-64bit.zip` from the [releases page](https://github.com/magefile/mage/releases)
2. Extract `mage.exe` to a directory in your `PATH` (e.g., `C:\Windows\System32`)

### Verify Installation

```bash
mage -version
```

## How to Use Mage

You have granular control over the build process. Available targets:

### Build Everything (Default)

Builds frontend and backend for all platforms (Linux, Windows, macOS):

```bash
mage
# or
mage All
```

### Build Frontend Only

Runs `npm install` and `npm run build`:

```bash
mage BuildFrontend
```

### Build Backend Only

Builds Go binaries for all platforms:

```bash
mage BuildBackend
```

### Build for Specific Platforms

```bash
# Linux (amd64 and arm64)
mage BuildLinux

# Windows (amd64)
mage BuildWindows

# macOS (amd64 and arm64)
mage BuildDarwin
```

### Clean Build Artifacts

Removes the `dist/` directory:

```bash
mage Clean
```

### List All Available Targets

```bash
mage -l
```

---

## 2. Create the Archive (ZIP)

Once the build finishes, everything you need is in the `dist/` directory. You can run `mage Clean` before building to ensure a fresh build.

### Command Line (Linux/Mac)

```bash
# 1. Create a directory named after your plugin ID
mkdir kinetica-datasource

# 2. Copy the contents of dist/ into it
cp -r dist/* kinetica-datasource/

# 3. Zip that directory
zip -r kinetica-datasource.zip kinetica-datasource/
```

**Why this structure?** When an administrator unzips this file in their Grafana `plugins` folder, it should extract into its own subdirectory, not scatter files into the root plugins folder.

## 3. Archive Contents

The final `kinetica-datasource.zip` will strictly contain compiled/production-ready files. It will **not** contain `src/`, `node_modules/`, or `package.json`.

| File / Folder | Description |
|---------------|-------------|
| `plugin.json` | Critical. Contains metadata (ID, version, executable names). Grafana reads this first. |
| `module.js` | The compiled frontend JavaScript bundle (React components, Query Editor, etc.). |
| `module.js.map` | Source maps for debugging the minified code. |
| `gpx_*` binaries | Backend executables for each platform: `gpx_*_linux_amd64`, `gpx_*_linux_arm64`, `gpx_*_darwin_amd64`, `gpx_*_darwin_arm64`, `gpx_*_windows_amd64.exe`. |
| `img/` | Contains plugin logos (`logo.svg`, `large.png`) displayed in Grafana. |
| `README.md` | The text displayed in the "Config" tab of the plugin in Grafana. |
| `LICENSE` | The license file. |

## 4. Installation on Existing Grafana

### 4.1 Copy the Archive

Transfer the `.zip` file to the Grafana server.

### 4.2 Locate the Plugins Directory

Usually `/var/lib/grafana/plugins`.

### 4.3 Extract the Archive

Unzip the file into the plugins directory.

### 4.4 Set Permissions

Ensure the Grafana user has read/execute permissions, especially for the `gpx_*` binaries:

```bash
chmod +x /var/lib/grafana/plugins/kinetica-datasource/gpx_*
```

### 4.5 Allow Unsigned Plugin

If you haven't signed the plugin with Grafana Labs, edit `grafana.ini`:

```ini
[plugins]
allow_loading_unsigned_plugins = kinetica-datasource
```

### 4.6 Restart Grafana

```bash
sudo systemctl restart grafana-server
```

---

## 5. Docker Deployment

For instructions on running Grafana locally with Docker and installing the plugin in a Docker environment, see the dedicated guide:

**[Docker Deployment Guide](grafana_docker_deployment.md)**

This guide covers:
- Running Grafana locally with Docker
- Installing the plugin from a directory or ZIP file
- Using Docker Compose
- Updating the plugin
- Troubleshooting
