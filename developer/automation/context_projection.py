from __future__ import annotations

from ptsip.context_plane import (
    JSONL_PATH,
    JSON_PATH,
    MANDATORY_FORMATS,
    SCHEMA_PATH,
    SOURCE_PATH,
    ContextProjectionError,
    ContextProjectionSet,
    check_context,
    decode_jsonl_records,
    main,
    render_projections,
    sync_context,
    write_context,
)

__all__ = [
    "JSONL_PATH",
    "JSON_PATH",
    "MANDATORY_FORMATS",
    "SCHEMA_PATH",
    "SOURCE_PATH",
    "ContextProjectionError",
    "ContextProjectionSet",
    "check_context",
    "decode_jsonl_records",
    "render_projections",
    "sync_context",
    "write_context",
]


if __name__ == "__main__":
    raise SystemExit(main())
