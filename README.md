# Shoko Anime Sync

A lightweight, high-performance media bridging utility written in Go. It listens to Jellyfin media playback webhook events, queries your local Shoko Server to resolve AniDB metadata, and synchronizes watch history directly to your Simkl account.

It also supports multi-user setups and sends scrobble failure notifications to a `ntfy` server.

---

## Features

- **Jellyfin Integration**: Listens to Jellyfin webhooks for playback stop events.
- **Shoko Metadata Resolution**: Automatically matches Jellyfin items with AniDB metadata via local Shoko Server endpoints.
- **Simkl Scrobbling**: Syncs both TV show episodes (with Specials support mapped to Season 0) and Movies.
- **Multi-User Mapping**: Supports mapping Jellyfin usernames to respective Simkl Bearer Tokens case-insensitively.
- **Ntfy Notifications**: Sends failure notifications (with clickable AniDB links) directly to your `ntfy` topic if a scrobble fails.
- **Docker-Ready**: Highly optimized multi-stage build container with default volume mapping configurations designed for Unraid and Docker Compose.

---

## Prerequisites

The following components and versions have been confirmed working:

- **[Jellyfin](https://jellyfin.org/) 10.11.X**
  - **[Shokofin](https://github.com/ShokoAnime/Shokofin) plugin 6.0.5.X** (for linking Jellyfin items to Shoko metadata)
  - **Webhook plugin 21.0.0.0** (for dispatching playback stop events)
- **[Shoko Server](https://shokoanime.com/downloads/shoko-server) 5.3.3**
  - Requires a Shoko API key (can be generated in the Shoko Admin Web UI)
- **[Simkl](https://simkl.com/) account**
  - **Client ID**: Register an application in the [Simkl Settings / Developer Console](https://api.simkl.org/api-reference/introduction) to obtain a Client ID.
  - **User Access Tokens**: Each user mapped in the application requires a Simkl Bearer Token (refer to the [Simkl OAuth Reference](https://api.simkl.org/api-reference/oauth)).
- **[ntfy](https://ntfy.sh/) server** *(Optional)*
  - Requires a `ntfy` server endpoint and optionally an API Token if access is protected.
  - Requires a `ntfy_topic` to target.

---

## Configuration

The application is configured using a `config.yaml` file. By default, it looks for the file in the following order:
1. `/config/config.yaml` (standard path for Docker/Unraid mounts)
2. `config.yaml` (local project directory)
3. A custom path specified in the `CONFIG_PATH` environment variable.

### `config.yaml` Structure

Create a `config.yaml` file using the following template:

```yaml
# Your Simkl Application Client ID (from Simkl developer settings)
simkl_client_id: "your_simkl_client_id_here"

# The Application Name used for API requests and User-Agent headers
app_name: "myapp"

# The API Token for your local Shoko Server
shoko_token: "your_shoko_token_here"

# The Base URL of your local Shoko Server
shoko_url: "http://10.0.0.132:8111"

# [Optional] ntfy server settings for sending notifications
# If ntfy_url is omitted or empty, ntfy notifications are skipped.
# If ntfy_url is set, ntfy_topic is required.
ntfy_url: "https://ntfy.sh/"
ntfy_token: ""
ntfy_topic: "anime-sync"

# Map of Jellyfin Usernames (case-insensitive) to their respective Simkl Bearer Tokens
users:
  userA: "SIMKL_BEARER_TOKEN_FOR_USER_A"
  userB: "SIMKL_BEARER_TOKEN_FOR_USER_B"
```

---

## Deployment

### Docker

Build the Docker image locally:
```bash
docker build -t shoko-anime-sync .
```

Run the container, mounting your configuration directory to `/config`:
```bash
docker run -d \
  -p 8282:3000 \
  -v /path/to/your/appdata/shoko-anime-sync:/config \
  --name shoko-anime-sync \
  shoko-anime-sync
```

### Unraid

When setting up your Unraid container template, configure the volume mapping and port mapping as follows:

1. **Volume Mapping**:
   * **Container Path**: `/config`
   * **Host Path**: `/mnt/user/appdata/shoko-anime-sync`
   * **Access Mode**: `Read/Write`

2. **Port Mapping**:
   * **Container Port**: `3000`
   * **Host Port**: `8282`
   * **Protocol**: `TCP`

Place your `config.yaml` inside `/mnt/user/appdata/shoko-anime-sync/` and start the container. No environment variables are necessary.

---

## Development & Testing

To run the application locally (make sure a `config.yaml` is present in the directory):
```bash
go run .
```

To run all unit and integration tests:
```bash
go test -v ./...
```

To build the binary:
```bash
go build -o shoko-anime-sync .
```

Then run with:
```bash
./shoko-anime-sync
```

---

## License

This project is licensed under the [MIT License](LICENSE).
