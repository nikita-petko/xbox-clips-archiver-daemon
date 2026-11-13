# xbox-clips-archiver-daemon

[![Go Build](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build.yml/badge.svg)](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build.yml)
[![Build Docker Image](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build-docker.yml/badge.svg)](https://github.com/nikita-petko/xbox-clips-archiver-daemon/actions/workflows/build-docker.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## Media uploads

Downloaded media is uploaded to the configured S3 bucket under
`<S3_PATH_PREFIX>/<optional user>/<optional title>/<filename>`.
`--uploader-group-by-user-mode` (or `UPLOADER_GROUP_BY_USER_MODE`) accepts
`none` (default), `id`, or `gamertag`. `--uploader-group-by-title-mode` (or
`UPLOADER_GROUP_BY_TITLE_MODE`) accepts `none`, `id`, or `name` (default).
Name-based grouping requires the corresponding gamertag or title name.
Path components containing slashes or percent signs are escaped.
`--upload-media-file-name-mode` (or `UPLOAD_MEDIA_FILE_NAME_MODE`) accepts
`captured` (default), which uses the UTC capture timestamp in the legacy format
`YYYY-MM-DDTHH-MM-SS+0000`, or `id`, which uses the media ID. Fractional
seconds are omitted. Both retain the
cached file's extension. A missing capture date in `captured` mode or missing
extension is an error. The S3 content type is derived from the extension;
unknown extensions use `application/octet-stream`.

Changing grouping or filename modes changes object keys; media stored under a
different layout is not detected automatically.
Captures in the same second with the same extension and grouping produce the
same key; use `id` mode to distinguish them.

Cached media
is checked for an existing object first; newly downloaded media is uploaded
without a prior existence check. Media with no download URI is skipped.
The upload package returns S3 URIs for newly uploaded media; existing objects
are not included. An upload or S3 lookup error is returned to the caller.

After each upload the S3 checksum is verified against the cached file by
default. Set `--should-verify-upload-integrity=false` or
`SHOULD_VERIFY_UPLOAD_INTEGRITY=false` to disable verification. Verification
requires S3 checksum metadata and, for SSE-KMS objects, may require additional
KMS permissions. A mismatch or verification error stops the upload pass and is
reported as an error; the object is not deleted. Existing objects are not
reverified on later passes.

When media is uploaded, configured alerting services receive the number of
screenshots and game clips, total uploaded bytes, upload-pass duration,
verification status, and unsigned regional HTTPS URLs for the uploaded
objects. The links do not grant access to private objects. Partial uploads are
also reported when an upload or verification fails. No alert is sent for an
empty successful pass. Spaces in object keys are encoded as `%20` and literal
plus signs as `%2B` in HTTPS links, so the timestamp's `+0000` is not
interpreted as a space by S3.

To use a CloudFront distribution or another HTTPS origin for alert links, set
`--alert-url-base=https://share.cdn.vmminfra.dev` or
`ALERT_URL_BASE=https://share.cdn.vmminfra.dev`. The complete S3 object key,
including `S3_PATH_PREFIX`, is appended to this base; an optional base path is
preserved. For example, the key `xbox/game/video.mp4` becomes
`https://share.cdn.vmminfra.dev/xbox/game/video.mp4`. The base must be an
absolute HTTPS URL without credentials, query parameters, or a fragment.
This only changes alert links, not S3 uploads or the S3 URIs returned by the
upload package. Configure the CDN origin and access permissions separately.

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
