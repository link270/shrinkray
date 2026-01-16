# Presets and encoders

Query available transcoding presets and detected hardware encoders.

## List presets

```
GET /api/presets
```

Returns all available presets with their assigned encoder.

**Response:**

```json
[
  {
    "id": "compress-hevc",
    "name": "Compress (HEVC)",
    "description": "Reduce size with HEVC encoding",
    "encoder": "nvenc",
    "codec": "hevc",
    "max_height": 0
  },
  {
    "id": "compress-av1",
    "name": "Compress (AV1)",
    "description": "Maximum compression with AV1 encoding",
    "encoder": "nvenc",
    "codec": "av1",
    "max_height": 0
  },
  {
    "id": "1080p",
    "name": "Downscale to 1080p",
    "description": "Downscale to 1080p max (HEVC)",
    "encoder": "nvenc",
    "codec": "hevc",
    "max_height": 1080
  },
  {
    "id": "720p",
    "name": "Downscale to 720p",
    "description": "Downscale to 720p (big savings)",
    "encoder": "nvenc",
    "codec": "hevc",
    "max_height": 720
  }
]
```

### Preset fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Preset identifier for API calls |
| `name` | string | Human-readable name |
| `description` | string | Brief description |
| `encoder` | string | Assigned hardware encoder |
| `codec` | string | Target codec: `hevc` or `av1` |
| `max_height` | int | Max output height (0 = no scaling) |

## List encoders

```
GET /api/encoders
```

Returns all detected hardware encoders and which one is selected as best.

**Response:**

```json
{
  "encoders": [
    {
      "name": "NVIDIA NVENC",
      "accel": "nvenc",
      "available": true,
      "hevc": true,
      "av1": true
    },
    {
      "name": "Intel Quick Sync",
      "accel": "qsv",
      "available": false,
      "hevc": false,
      "av1": false
    },
    {
      "name": "VAAPI (Linux)",
      "accel": "vaapi",
      "available": false,
      "hevc": false,
      "av1": false
    },
    {
      "name": "Apple VideoToolbox",
      "accel": "videotoolbox",
      "available": false,
      "hevc": false,
      "av1": false
    },
    {
      "name": "Software (CPU)",
      "accel": "none",
      "available": true,
      "hevc": true,
      "av1": true
    }
  ],
  "best": {
    "name": "NVIDIA NVENC",
    "accel": "nvenc",
    "available": true,
    "hevc": true,
    "av1": true
  }
}
```

### Encoder fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Human-readable encoder name |
| `accel` | string | Encoder type: `nvenc`, `qsv`, `vaapi`, `videotoolbox`, `none` |
| `available` | bool | Whether encoder was successfully detected |
| `hevc` | bool | HEVC encoding supported |
| `av1` | bool | AV1 encoding supported |

## Hardware encoder priority

Shrinkray automatically selects the best available encoder in this order:

1. **NVIDIA NVENC** - Best quality/speed balance, wide GPU support
2. **Intel Quick Sync** - Good for Intel CPUs with integrated graphics
3. **AMD VAAPI** - Linux AMD GPU encoding
4. **Apple VideoToolbox** - macOS hardware encoding
5. **Software** - CPU fallback (libx265/libsvtav1)

## Codec support by hardware

| Platform | HEVC | AV1 |
|----------|------|-----|
| NVIDIA NVENC | GTX 1050+ | RTX 40 series+ |
| Intel QSV | 6th gen+ | Arc GPUs, 14th gen+ |
| AMD VAAPI | Polaris+ | RX 7000+ |
| Apple VideoToolbox | Any Mac | M3+ |
| Software | Always | Always |

If hardware AV1 encoding is not available, AV1 presets fall back to software encoding (significantly slower).
