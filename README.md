# PasteCrate

**Move text or files between devices. Just a browser and a short code.**

No accounts. No chat apps. No file-transfer assistants. No cloud-drive sign-ins. PasteCrate is a tiny **self-hosted pastebin** for sharing a note, a long text snippet, or one or more files, even across different operating systems.

## How it works

1. **On device A:** open your PasteCrate page, paste text and/or attach files, then choose a memorable code, such as `12`.
2. **On device B:** open **`https://paste.example.com/12`** in any browser.
3. **Retrieve and move on:** copy the text or download the files. Delete the paste when done, or leave it to expire automatically after 72 hours.

That's it. Your URL is simply your server address plus the short code you chose.

**Everyday uses:** move a document from phone to laptop; copy a long command to a remote machine with no shared clipboard; share a non-confidential file with a friend; transfer text or files between Windows, macOS, Linux, and mobile devices without installing another client.

> **Privacy note:** This is for **non-sensitive content**. Anyone who knows or guesses a code (especially `12`) can read, download, **and delete** the paste. `paste.example.com` is an illustrative address, not a hosted PasteCrate instance. Your own server must be reachable by both devices.

## Features

- **No accounts or client apps:** any modern browser will do.
- **Memorable URLs:** choose a 2–64 character code using letters, numbers, hyphens, or underscores.
- **Text and files together:** share text, multiple files, or both in one paste.
- **Instant retrieval:** just open the URL and copy/download.
- **Quick cleanup:** delete manually or let it expire after **72 hours**.
- **Live large-upload progress:** a progress bar when selected files total more than 10 MiB.
- **Small and self-hosted:** one Go server; no database or JavaScript build system.
- **Upload cap:** 100 MiB per HTTP request, including form overhead; 2 MiB text limit.

## Get started

### Docker Compose (recommended)

```bash
git clone https://github.com/LeonardNJU/pastecrate.git
cd pastecrate
docker compose up -d --build
```

Open **http://localhost:8080** on the server. The provided Compose file binds to localhost. To access it across the Internet, configure your own domain, HTTPS reverse proxy, and abuse protection. Do not expose the port directly without reviewing the security notes.

Uploads persist in the Docker volume `paste_data`.

### Download a prebuilt executable

Get the archive for your OS and CPU from [GitHub Releases](https://github.com/LeonardNJU/pastecrate/releases), extract it, and run:

```bash
./pastecrate
```

On Windows PowerShell, run `.\pastecrate.exe`. Open **http://localhost:8080**.

By default, standalone binaries store pastes in `./data/` relative to the working directory. Set `PASTE_DATA_DIR` and `PASTE_LISTEN_ADDR` to change storage and listening address:

```bash
PASTE_DATA_DIR=/srv/pastecrate PASTE_LISTEN_ADDR=127.0.0.1:8080 ./pastecrate
```

The container sets `PASTE_DATA_DIR=/data`, and the included Compose file uses a persistent volume.

### Build from source

Requires Go 1.23+.

```bash
go test ./...
go build -o pastecrate .
./pastecrate
```

## Security & limitations

**PasteCrate is public by design, not a private document vault.**

- No account, access control, end-to-end encryption, or hard-to-guess generated codes. Anyone who can guess the code can **read and delete** the paste.
- Don't upload passwords, private documents, keys, or confidential information.
- No built-in upload rate limit, per-IP quota, total disk quota, antivirus scanner, or moderation. Add protections before making an instance available to anonymous Internet traffic.
- The 100 MiB limit applies to the whole multipart request, so an exactly 100 MiB file exceeds it once form overhead is counted.
- The server uses Go multipart parsing with an 8 MiB in-memory threshold; larger uploads may spool to disk. High concurrency can exhaust server resources.
- Expiration is 72 hours after creation; expired items are checked when accessed and in a periodic cleanup loop.
- Docker data is saved in the `paste_data` volume (`/data` in the container); the standalone binary defaults to `./data`. Never publish or commit uploaded data.
- HTTPS through a reverse proxy is recommended for any Internet deployment.

## Project layout

- `main.go` — Go HTTP server and embedded web interface.
- `Dockerfile` — multi-stage production image.
- `compose.yaml` — simple persistent self-hosted deployment.
- `main_test.go` — tests.
- `scripts/build-release.sh` — cross-platform release archive builder.

## License

MIT.
