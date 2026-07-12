# Test fixture: fsharp-elm-empty

This directory contains a synthetic, frozen target tree representing
an empty future Circus repository, used as a golden fixture for the
`factory-core-v1` doctrine compiler.

It is not The Circus repository and must not contain any product
implementation. Only Factory wiring is allowed.

## Layout

```
fsharp-elm-empty/
├── README.md          # this file
├── expected/          # golden compiled tree
│   ├── Makefile
│   ├── docs/
│   │   └── factory/
│   │       └── README.md
│   └── .factory/
│       ├── doctrine.lock.json
│       ├── project.json
│       └── generated/
│           ├── factory.mk
│           └── doctrine-inventory.md
```

## Maintenance

Golden files are reviewed content, not opaque serialized state. When
the canonical `factory-core-v1` pack changes, regenerate the expected
tree with:

```bash
rm -rf /tmp/circus-fixture && mkdir /tmp/circus-fixture
go run ./cmd/leamas factory doctrine compile \
  --profile fsharp-elm-service-v1 \
  --target /tmp/circus-fixture
cp -R /tmp/circus-fixture/. \
  internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/
```

Review the diff manually. The fixture must remain deterministic and
must not contain user-specific paths, timestamps, or secrets.
