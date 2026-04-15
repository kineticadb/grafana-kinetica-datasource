# Deploying the Kinetica Grafana Plugin on Docker

This guide covers how to run Grafana locally using Docker and install the Kinetica datasource plugin.

## 1. Running Grafana Locally with Docker

If you don't have a Grafana instance running, you can quickly set one up using Docker.

### 1.1 Prerequisites

- Docker installed and running on your machine
- Verify Docker is working:

```bash
docker --version
docker ps
```

### 1.2 Run Grafana Container

Start a basic Grafana container:

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  grafana/grafana:latest
```

### 1.3 Run with Persistent Storage

To persist your dashboards, data sources, and settings across container restarts:

```bash
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -v grafana-storage:/var/lib/grafana \
  grafana/grafana:latest
```

### 1.4 Access Grafana

1. Open your browser and navigate to `http://localhost:3000`
2. Log in with the default credentials:
   - Username: `admin`
   - Password: `admin`
3. You'll be prompted to change the password on first login

### 1.5 Stop and Start the Container

```bash
# Stop Grafana
docker stop grafana

# Start Grafana
docker start grafana

# Remove the container (data persists if using a volume)
docker rm grafana
```

---

## 2. Installing the Plugin on Grafana Running in Docker

Follow these steps to install the plugin from the `kinetica-datasource` directory or ZIP file.

### 2.1 Prerequisites

- Docker installed and running
- The `kinetica-datasource` directory or `kinetica-datasource.zip` file containing the built plugin (see [grafana_deployment.md](grafana_deployment.md) for build instructions)

### 2.2 Option A: Mount the Plugin Directory (Recommended for Development)

This approach mounts the plugin directory directly into the container, making it easy to update the plugin without rebuilding the container.

```bash
# Run Grafana with the plugin directory mounted
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -v $(pwd)/kinetica-datasource:/var/lib/grafana/plugins/kinetica-datasource \
  -e "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-datasource" \
  grafana/grafana:latest
```

**Notes:**
- Replace `$(pwd)/kinetica-datasource` with the absolute path to your `kinetica-datasource` directory if not running from the parent directory
- The `-e` flag sets the environment variable to allow the unsigned plugin

### 2.3 Option B: Copy Plugin into a Named Volume (Recommended for Persistence)

This approach uses a Docker volume for persistent storage.

```bash
# 1. Create a named volume for plugins
docker volume create grafana-plugins

# 2. Start a temporary container to copy the plugin
docker run --rm \
  -v $(pwd)/kinetica-datasource:/src \
  -v grafana-plugins:/dest \
  alpine cp -r /src /dest/kinetica-datasource

# 3. Run Grafana with the plugins volume
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -v grafana-plugins:/var/lib/grafana/plugins \
  -e "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-datasource" \
  grafana/grafana:latest
```

### 2.4 Option C: Using Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    volumes:
      - ./kinetica-datasource:/var/lib/grafana/plugins/kinetica-datasource
      - grafana-storage:/var/lib/grafana
    environment:
      - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-datasource
    restart: unless-stopped

volumes:
  grafana-storage:
```

Then start Grafana:

```bash
docker-compose up -d
```

### 2.5 Option D: Install from ZIP File

If you have the `kinetica-datasource.zip` archive, you can install it into a running or new Grafana Docker container.

#### Installing into a New Container

```bash
# 1. Create a named volume for plugins
docker volume create grafana-plugins

# 2. Extract the zip file into the volume using a temporary container
docker run --rm \
  -v $(pwd)/kinetica-datasource.zip:/tmp/plugin.zip \
  -v grafana-plugins:/plugins \
  alpine sh -c "unzip /tmp/plugin.zip -d /plugins"

# 3. Run Grafana with the plugins volume
docker run -d \
  --name grafana \
  -p 3000:3000 \
  -v grafana-plugins:/var/lib/grafana/plugins \
  -e "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=kinetica-datasource" \
  grafana/grafana:latest
```

#### Installing into an Existing Running Container

```bash
# 1. Copy the zip file into the container
docker cp kinetica-datasource.zip grafana:/tmp/

# 2. Extract the zip file inside the container
docker exec grafana sh -c "unzip -o /tmp/kinetica-datasource.zip -d /var/lib/grafana/plugins/"

# 3. Clean up the zip file
docker exec grafana rm /tmp/kinetica-datasource.zip

# 4. Restart the container to load the plugin
docker restart grafana
```

**Note:** The container must have `unzip` available. If not, use this alternative approach:

```bash
# 1. Extract locally first
unzip kinetica-datasource.zip -d /tmp/

# 2. Copy the extracted directory into the container
docker cp /tmp/kinetica-datasource grafana:/var/lib/grafana/plugins/

# 3. Restart the container
docker restart grafana

# 4. Clean up local extracted files
rm -rf /tmp/kinetica-datasource
```

---

## 3. Verify the Plugin Installation

1. Open your browser and navigate to `http://localhost:3000`
2. Log in with default credentials (`admin` / `admin`)
3. Go to **Administration** → **Plugins**
4. Search for "Kinetica" — the plugin should appear in the list
5. Click on the plugin and select **Create a Kinetica data source** to configure it

---

## 4. Updating the Plugin

**For mounted volumes (Option A and Docker Compose):**

```bash
# 1. Update the kinetica-datasource directory with new build
cp -r dist/* kinetica-datasource/

# 2. Restart the Grafana container
docker restart grafana
```

**For named volumes (Option B):**

```bash
# 1. Copy updated plugin to the volume
docker run --rm \
  -v $(pwd)/kinetica-datasource:/src \
  -v grafana-plugins:/dest \
  alpine sh -c "rm -rf /dest/kinetica-datasource && cp -r /src /dest/kinetica-datasource"

# 2. Restart the Grafana container
docker restart grafana
```

---

## 5. Viewing Plugin Logs

To troubleshoot plugin issues, check the Grafana container logs:

```bash
docker logs grafana | grep -i kinetica
```

Or follow logs in real-time:

```bash
docker logs -f grafana
```
