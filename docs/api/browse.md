# Browse API

Discover and browse video files in your media library.

## Browse directory

```
GET /api/browse?path=/media/Movies
```

List contents of a directory with video metadata.

**Query parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `path` | Config `media_path` | Directory to browse |

**Response:**

```json
{
  "path": "/media/Movies",
  "parent": "/media",
  "directories": [
    {
      "name": "Action",
      "path": "/media/Movies/Action"
    }
  ],
  "files": [
    {
      "name": "Movie.mkv",
      "path": "/media/Movies/Movie.mkv",
      "size": 4294967296,
      "duration": 7200000,
      "bitrate": 5000000,
      "width": 1920,
      "height": 1080,
      "frame_rate": 23.976,
      "video_codec": "h264",
      "profile": "High",
      "bit_depth": 8,
      "is_hdr": false,
      "is_hevc": false,
      "is_av1": false
    }
  ]
}
```

### File metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Filename |
| `path` | string | Absolute file path |
| `size` | int64 | File size in bytes |
| `duration` | int64 | Duration in milliseconds |
| `bitrate` | int64 | Video bitrate in bits/second |
| `width` | int | Video width in pixels |
| `height` | int | Video height in pixels |
| `frame_rate` | float64 | Frames per second |
| `video_codec` | string | Codec name (h264, hevc, av1, etc.) |
| `profile` | string | Codec profile (High, Main, Main 10) |
| `bit_depth` | int | Color bit depth (8, 10, 12) |
| `is_hdr` | bool | True if HDR content (HDR10, HLG) |
| `is_hevc` | bool | True if already HEVC encoded |
| `is_av1` | bool | True if already AV1 encoded |

**Errors:**
- `500` - Directory not found or inaccessible

## Clear cache

```
POST /api/cache/clear
```

Clear the file metadata cache. Useful after adding new files or if metadata appears stale.

**Response:**

```json
{
  "status": "cache cleared"
}
```

## Notes

- Metadata is cached to speed up browsing large directories
- Only video files are included in the `files` array
- Hidden files and directories (starting with `.`) are excluded
- Symlinks are followed
