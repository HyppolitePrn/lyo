# ADR 010 — Track Upload: Presigned S3 URLs Instead of Proxying Through the API

**Status:** Accepted
**Date:** 2026-08-05

## Context

Broadcasters publish recorded tracks: audio files ranging from a few to a few hundred megabytes, uploaded
from phones on connections that are frequently slow and occasionally drop.

The straightforward implementation is a multipart upload to the API, which then forwards the bytes to
object storage. On a single-VPS deployment that shares one process with the live streaming Hub, this is
a poor idea for a specific reason: the audio bytes would be buffered and copied by the same process that
must keep fanning out real-time chunks with low latency. A handful of concurrent uploads on slow
connections would occupy request handlers and memory for minutes at a time, and the visible symptom
would be degraded live audio — a failure in an unrelated feature, which is the hardest kind to diagnose.

## Decision

The API issues an **authorisation**; it never carries the payload.

1. `POST /tracks/upload-url` — checks the `track_uploads` feature flag and the broadcaster role, then
   returns a short-lived **presigned PUT URL** scoped to a single object, plus the future public URL.
2. The phone `PUT`s the file **directly to the object store** (MinIO in development, any S3-compatible
   service in production).
3. `POST /tracks` — registers the metadata (title, artist, duration, public URL) once the transfer has
   succeeded.

`internal/storage/s3.go` also derives the object key back from a public URL, so deleting a track deletes
its object.

## Consequences

**Positive**

- The backend's memory and bandwidth are decoupled from upload traffic. Live streaming latency cannot be
  degraded by someone uploading a large file — the two workloads no longer share a resource.
- The transfer is client-to-storage, so it benefits from the storage provider's edge and retry behaviour
  rather than from a single VPS's uplink.
- The presigned URL is short-lived and object-scoped: no long-term credential ever reaches the client,
  and the grant cannot be reused for another object.
- Authorisation stays where it belongs — role and feature-flag checks happen at URL issuance, in the API.

**Negative, and accepted**

- **The upload is a three-step flow, and step 2 can succeed while step 3 never happens** (the app is
  killed, the network drops). The result is an orphaned object in the bucket with no database row. This
  is a real cost of the design; the remedy is a periodic reconciliation job listing objects without a
  matching track, which is not yet implemented.
- The API cannot inspect or validate the file content, since it never sees it. Content-type and size
  constraints must be expressed in the presigned URL itself.
- `S3_PUBLIC_ENDPOINT` must be reachable **from the device**, not from the backend — a distinction that
  is easy to get wrong across emulator, physical device and production, and which is documented at length
  in `pkg/config/config.go`.
- Object storage becomes a client-visible dependency: if it is down, publishing and recorded playback
  fail. Live streaming keeps working, which makes the degradation partial rather than total — an
  acceptable outcome that was a factor in accepting this design.
