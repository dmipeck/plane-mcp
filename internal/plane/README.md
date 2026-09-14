# Plane OpenAPI source of truth

`openapi.yaml` is the full spectacular schema for Plane **v1.4.2**, pinned to
makeplane/plane commit `5f7d92784c403f76284f0f16718f320221dc7fec`.

It lives **beside** (not inside) `internal/oas/`, so `go generate` with ogen
`--clean` cannot wipe the vendored schema.

## Regenerate the YAML

Prefer the tagged backend image so deps match the release:

```bash
docker run --rm \
  -e ENABLE_DRF_SPECTACULAR=1 \
  -e SECRET_KEY='regen-not-a-secret' \
  -e DATABASE_URL='sqlite:////tmp/plane.db' \
  -e REDIS_URL='redis://127.0.0.1:6379/0' \
  -v "$PWD/internal/plane:/out" \
  makeplane/plane-backend:v1.4.2 \
  python manage.py spectacular --format openapi --file /out/openapi.yaml
```

Do not use Cloud `https://api.plane.so/api/schema/` as SoT.

## Regenerate the Go client

From the repo root (after `go get -tool` has pinned ogen):

```bash
go generate ./...
```

`.ogen.yml` filters to the six v0.1 path templates only.
