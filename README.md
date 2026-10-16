# PasteCrate

A tiny, self-hosted box for sharing text and files. No accounts, no database, just memorable, user-chosen links.

Built with Go's standard library, packaged for Docker. No database, external frontend framework, or API keys required.

## Features

- Share text and/or multiple files together using your-code URLs.
- Choose a share code of 2-64 alphanumeric, dash or underscore characters.
- Files are downloaded rather than displayed inline.
- Upload progress indicator for files totaling more than 10 MiB.
- Pastes expire automatically after 72 hours.
- Up to 100 MiB per multipart request, including form overhead; 2 MiB text limit.
- Light/dark appearance based on system settings.
- Everything is stored on a mounted filesystem volume.

## Quick start

~~~sh
git clone https://github.com/LeonardNJU/pastecrate.git
cd pastecrate
docker compose up -d --build
~~~

Open http://localhost:8080. For Internet access, use your own HTTPS reverse proxy, such as Nginx. Do not expose port 8080 directly to the Internet without reviewing the security notes below.

## Architecture

- main.go — Go HTTP server, HTML template, upload handling, expiry cleanup.
- Dockerfile — multi-stage static Go binary build.
- compose.yaml — container and persistent data volume.
- paste_data — named Docker volume holding persistent user uploads; not included in Git.

## Security and limitations

**This is a deliberately small, public-by-design tool, not a private document vault.**

- No authentication or end-to-end encryption. Anyone who can guess a code can read the paste and delete it. Do not upload secrets or sensitive files.
- Codes are chosen by users, and are not guaranteed to be difficult to guess.
- No built-in upload rate limiting, per-IP quotas, global storage quota, malware scan, or moderation. If deployed publicly, configure abuse protection and monitoring at the edge, and consider adding quotas before inviting anonymous uploads.
- Uploads have a fixed 72-hour lifetime. The server performs periodic cleanup; expiry checks also happen on reads.
- The 100 MiB limit is for the full HTTP request, so a file of exactly 100 MiB is too large due to multipart overhead.
- Server-side multipart parsing uses an 8 MiB in-memory threshold and may spool larger requests to temporary files. High concurrency can still exhaust resources; limit concurrent uploads at your reverse proxy.
- Uploaded data lives in the paste_data Docker volume (or /data inside the container). Never publish it or include it in Git history.

## Development

Run with Go 1.23 or later in a writable environment with a /data directory:

~~~sh
go run .
~~~

The application uses /data as its fixed storage path, so local Docker deployment is recommended.

## License

MIT.
