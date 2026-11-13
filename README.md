# xbox-clips-archiver-daemon

[![Go Build](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build.yml/badge.svg)](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build.yml)
[![Go Test](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/test.yml/badge.svg)](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/test.yml)
[![Build Docker Image](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build-docker.yml/badge.svg)](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build-docker.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Brief description

`xbox-clips-archiver-daemon` is a Go daemon that downloads screenshots and game
clips for one or more Xbox accounts and archives the media in an S3 bucket. It
uses the Xbox Live API through [xbl.io](https://xbl.io/), keeps a local cache to
avoid downloading the same media repeatedly, and can send upload notifications
to Discord, email through SendGrid, or AWS SNS.

## Detailed description

For each configured Xbox API key, the daemon loads the corresponding profile
and fetches that account's paginated screenshots and game clips. It downloads
available media to a local cache, selects files that have not already been
archived in S3, uploads them, and verifies each uploaded object's checksum by
default. Upload keys can be organized by user and game, and alerts can include
HTTPS links to the uploaded objects.

The daemon can run continuously at a configurable interval or run a single
pass with `--pulse`, which is useful for scheduled jobs. Xbox API keys,
AWS credentials, and alert-provider settings are supplied through flags or
environment variables.

## Pipeline

At startup, account profiles are loaded concurrently. Each work pass then runs
the following pipeline concurrently for each distinct account. A user's upload
can begin without waiting for other users' downloads:

```
  1. Load Xbox account profiles for the configured API keys concurrently (at startup)
                         |
                         v
  2. Fetch paginated screenshots and game clips for the account
                         |
                         v
  3. Download available media to the local cache
                         |
                         v
  4. Build S3 object keys and check cached media for existing objects
                         |
                         v
  5. Upload new objects and verify S3 checksums (enabled by default)
                         |
                         v
  6. Send account-specific upload or partial-failure alerts to configured providers
                         |
                         v
  7. Exit (--pulse) or wait for --interval and start another pass
```

Media without a download URI is skipped. Existing S3 objects are checked for
cached media; files downloaded during the current pass are uploaded without a
pre-upload existence lookup. Per-user API or download failures are logged and
skipped so processing can continue for other accounts. An upload or checksum
verification error stops that account's upload, but other accounts continue;
the errors are returned after all accounts finish. Duplicate S3 object keys
within a pass are rejected, including collisions between accounts.
When multiple API keys resolve to the same account, its pipeline runs once,
trying another key if fetching or downloading media with the first key fails.

## Features

### Application features

- **Xbox DVR archival** — fetches screenshots and game clips for multiple
  accounts using a comma-separated list of API keys.
- **Paged API retrieval** — follows the continuation tokens returned by the
  Xbox API to retrieve all available media for each account.
- **Local media cache** — stores downloaded files by user ID, title ID, and
  media ID, and tracks cached downloads for 30 days.
- **Duplicate-aware S3 uploads** — checks S3 for cached files before uploading
  them again; newly downloaded media is uploaded directly.
- **Configurable object keys** — group by user ID or gamertag, group by title
  ID or name, and name files with either their capture timestamp or media ID.
- **Upload integrity verification** — compares uploaded media with S3 checksum
  metadata by default; verification can be disabled.
- **Multi-channel alerting** — notify through a Discord webhook, SendGrid email,
  or AWS SNS topic. Configured providers all receive upload notifications.
- **Upload reports** — each account has its own alerts, identified by gamertag
  and user ID, with counts by media type, uploaded bytes,
  duration, verification status, and unsigned HTTPS object URLs. Partial uploads
  are reported when a later upload or verification fails; an empty successful
  pass does not send an alert.
- **Two run modes** — a recurring daemon (default interval: five minutes) or a
  single work pass with `--pulse`.
- **Configurable operation** — application settings are available as command
  line flags and environment variables.
- **Signal handling** — the daemon handles `SIGINT`, `SIGTERM`, and `SIGABRT`,
  and flushes logs before exiting.
- **Structured logging** — uses `glog`; logs are written to standard error by
  default.

### Repository features

- **Vendored Go modules** for reproducible builds using the checked-in
  dependency tree.
- **Cross-compilation Makefile targets** — debug and release builds for Linux,
  macOS, and Windows across `x86`, `x64`, `arm`, and `arm64`.
- **Docker image build** — builds a Linux x64 release binary and removes the
  source tree from the final image.
- **GitHub Actions**:
  - [`build.yml`](.github/workflows/build.yml) runs `go mod tidy`, refreshes the
    vendor tree, runs `go vet`, builds a release binary, and creates a GitHub
    Release on pushes to `master`.
  - [`build-docker.yml`](.github/workflows/build-docker.yml) builds and pushes a
    Docker image on pushes to `master` and supports manual dispatch.
- **Apache License 2.0**.

## Requirements

- [Go 1.25+](https://go.dev/dl/) (see `src/go.mod`)
- [GNU Make](https://www.gnu.org/software/make/)
- A POSIX-compatible shell for Make recipes
- Access to xbl.io and one or more Xbox API keys
- An S3 bucket and AWS credentials
- Docker (optional, for building or running a container image)

## Building

1. Clone the repository:

   ```sh
   git clone git@github.com:nikita-petko/xbox-clips-archiver-daemon.git
   cd xbox-clips-archiver-daemon
   ```

2. Build a debug binary:

   ```sh
   make build-debug
   ```

   For a release build, use `make build-release`. The build targets use the
   checked-in vendor tree and do not update dependencies. Run `make vendor`
   explicitly after changing Go dependencies. Override the target platform with
   `GOOS` and `GOARCH`, for example:

   ```sh
   make build-release GOOS=linux GOARCH=amd64
   ```

   Useful Make targets:

   | Target | Description |
   | --- | --- |
   | `build-debug` | Debug build for the detected host OS and architecture |
   | `build-release` | Release build with stripped symbols |
   | `build-debug-<arch>` / `build-release-<arch>` | Build one of `x86`, `x64`, `arm`, or `arm64` |
   | `build-debug-all` / `build-release-all` | Build all supported architectures |
   | `build-docker` | Build a Docker image; CI also pushes the image |
   | `fmt` | Format Go source files |
   | `test` | Run `go vet` and race-enabled unit tests |
   | `vendor` | Run `go mod tidy` and update `src/vendor/` |

   Build outputs are placed under `bin/<debug|release>/<os>/<arch>/`.

### Building with Docker

```sh
make build-docker IMAGE_NAME=mfdlabs/xbox-clips-archiver-daemon
```

Or build and run the image directly:

```sh
docker build -t xbox-clips-archiver-daemon -f Dockerfile .
docker run --rm \
  -e XBL_API_KEYS \
  -e S3_BUCKET \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  xbox-clips-archiver-daemon --pulse
```

The Docker image contains a release executable and expects its runtime
configuration to be supplied through flags or environment variables.

## Usage

Run `bin/debug/<os>/<arch>/xbox-clips-archiver-daemon --help` for the generated
flag help, or run from source:

```sh
cd src
go run -mod=vendor main.go --help
```

### Example: one-shot run

The following example uses the AWS SDK's default credential chain for S3:

```sh
export XBL_API_KEYS="xbl-api-key-1,xbl-api-key-2"
export S3_BUCKET="my-xbox-archive"
export S3_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."

bin/release/linux/x64/xbox-clips-archiver-daemon \
  --pulse \
  --cache-path .cache \
  --s3-path-prefix xbox
```

For S3, credentials can instead be supplied with both `--s3-access-key` and
`--s3-secret-access-key`; supplying only one is an error. When neither is
provided, the AWS SDK default credential chain is used. The optional
`AWS_SESSION_TOKEN` is included when explicit S3 access-key flags are used.

### Application flags and environment variables

Environment variables are applied after command-line flags and therefore take
precedence when both are supplied.

| Flag | Environment variable | Description |
| --- | --- | --- |
| `--interval` | `INTERVAL` | Delay between work passes; default `5m` |
| `--pulse` | — | Run one pass and exit |
| `--cache-path` | `CACHE_PATH` | Local cache directory; default `.cache` |
| `--clear-cache` | `CLEAR_CACHE` | Remove the cache directory before startup |
| `--should-lock-cache` | `SHOULD_LOCK_CACHE` | Acquire an exclusive cache lock; default `true` |
| `--xbl-api-keys` | `XBL_API_KEYS` | Required comma-separated xbl.io API keys |
| `--s3-bucket` | `S3_BUCKET` | Required destination bucket |
| `--s3-region` | `S3_REGION` | AWS region; default `us-east-1` |
| `--s3-access-key` | `S3_ACCESS_KEY` | Optional explicit S3 access key |
| `--s3-secret-access-key` | `S3_SECRET_ACCESS_KEY` | Secret paired with the explicit S3 access key |
| `--s3-path-prefix` | `S3_PATH_PREFIX` | Optional prefix for uploaded object keys |
| `--uploader-group-by-user-mode` | `UPLOADER_GROUP_BY_USER_MODE` | `none` (default), `id`, or `gamertag` |
| `--uploader-group-by-title-mode` | `UPLOADER_GROUP_BY_TITLE_MODE` | `none`, `id`, or `name` (default) |
| `--upload-media-file-name-mode` | `UPLOAD_MEDIA_FILE_NAME_MODE` | `captured` (default) or `id` |
| `--should-verify-upload-integrity` | `SHOULD_VERIFY_UPLOAD_INTEGRITY` | Verify S3 checksum after upload; default `true` |
| `--alert-url-base` | `ALERT_URL_BASE` | Optional HTTPS base URL for alert links |
| `--sendgrid-api-key` | `SEND_GRID_API_KEY` | Optional SendGrid API key |
| `--sendgrid-from` | `SEND_GRID_FROM` | SendGrid sender name; required with an API key |
| `--sendgrid-from-email` | `SEND_GRID_FROM_EMAIL` | SendGrid sender address; required with an API key |
| `--sendgrid-mailing-list` | `SEND_GRID_MAILING_LIST` | Comma-separated recipient addresses; required with an API key |
| `--sns-topic-arn` | `SNS_TOPIC_ARN` | Optional AWS SNS topic ARN |
| `--aws-credentials-from-profile` | `AWS_CREDENTIALS_FROM_PROFILE` | Use AWS profile credentials for SNS; default `false` |
| `--discord-webhook-uri` | `DISCORD_WEBHOOK_URI` | Optional Discord webhook URL |
| `--discord-alert-role-id` | `DISCORD_ALERT_ROLE_ID` | Optional Discord role ID to mention |

The application reads the SendGrid environment variables with the `SEND_GRID_`
prefix shown above. The generated flag help currently displays `SENDGRID_`
without the underscore; use the flag or the `SEND_GRID_` variable names.

If SNS alerting is enabled and `AWS_CREDENTIALS_FROM_PROFILE` is false, the
application requires `AWS_ACCESS_KEY` and `AWS_SECRET_ACCESS_KEY` to be set.
When the flag is true, those variables are not required; in either case, make
sure the AWS SDK's default credential chain can provide credentials for the
SNS client (for example, through an AWS profile). These SNS settings are
separate from the S3 access-key flags and the SDK's default S3 credential
chain.

### S3 object keys and upload behavior

Objects are stored using this layout:

```text
<S3_PATH_PREFIX>/<optional user>/<optional title>/<filename>
```

The user grouping mode accepts `none`, `id`, or `gamertag`; title grouping
accepts `none`, `id`, or `name`. Name-based grouping requires the corresponding
gamertag or title name. Path components containing slashes or percent signs
are escaped.

The filename mode `captured` uses the UTC capture timestamp in the legacy
format `YYYY-MM-DDTHH-MM-SS+0000`; `id` uses the media ID. The cached file's
extension is retained. Fractional seconds are omitted, so captures with the
same second, extension, and grouping can map to the same key. A missing capture
date in `captured` mode or a missing extension is an error. The content type is
derived from the extension; unknown extensions use
`application/octet-stream`.

Changing the grouping or filename mode changes object keys, and media stored
under a previous layout is not automatically detected. After upload, checksum
verification requires S3 checksum metadata. For SSE-KMS objects, verification
may also require additional KMS permissions. A mismatch or verification error
stops that user's uploads for the pass; the uploaded object is not deleted. Existing objects
are not reverified on later passes.

Alert links are unsigned regional HTTPS URLs by default and do not grant access
to private objects. Set `--alert-url-base` or `ALERT_URL_BASE` to use a CDN or
other HTTPS origin instead. The full object key, including the S3 prefix, is
appended to this base while preserving an optional base path. The value must
be an absolute HTTPS URL without credentials, query parameters, or a fragment.
This only changes alert links, not S3 uploads or the S3 URIs returned by the
upload package; configure the CDN origin and access permissions separately.

### Cache layout

- `<cache-path>/<user ID>/<title ID>/<media ID>/<filename>` — downloaded
  screenshot and game clip files.
- `<cache-path>/download_cache.json` — download-cache dates; entries expire
  after 30 days.
- `<cache-path>/user_profile_cache.json` — cached Xbox user IDs and gamertags;
  API keys are represented by SHA-256 hashes.
- `<cache-path>/cache.lock` — exclusive lock file preventing concurrent
  instances from using the same cache, unless locking is disabled.

`--clear-cache` removes all cached files and state before startup.

## Project layout

| Package | Responsibility |
| --- | --- |
| `flags` | Command-line options, environment-variable configuration, and validation |
| `http` | Shared retryable HTTP client |
| `xbl` | Xbox API requests |
| `users` | Resolve and cache Xbox user profiles |
| `dvr` | Fetch paginated screenshots and game clips |
| `cache` | Downloaded media, cache metadata, user profiles, and cache locking |
| `dl` | Download and group media for configured users |
| `s3` | S3 client, uploads, object lookups, URLs, and checksum verification |
| `upload` | S3 key construction, upload selection, and upload reports |
| `alerting` | Alert dispatch to Discord, SendGrid, and AWS SNS |
| `daemon` | Work-pass loop, upload notifications, and signal handling |

## License

```txt
Copyright 2023 Nikita Petko <petko@vmminfra.net>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
