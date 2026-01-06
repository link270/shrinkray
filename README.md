<div align="center">
  <img src="web/templates/logo.png" alt="Shrinkray" width="128" height="128">
  <h1>Shrinkray</h1>
  <p><strong>Intentional video transcoding for Unraid</strong></p>
  <p>Select a folder. Pick a preset. Shrink your media library.</p>

  ![Version](https://img.shields.io/badge/version-1.3.7-blue?style=flat-square)
  ![Go](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat-square&logo=go)
  ![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)
  ![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?style=flat-square&logo=docker)
</div>

---

## What is Shrinkray?

Shrinkray is a user-friendly video transcoding tool designed for Unraid to be simple from the ground up.

### Features

- **4 Smart Presets** — HEVC compress, AV1 compress, 1080p downscale, 720p downscale
- **Full GPU Pipeline** — Hardware decoding AND encoding (frames stay on GPU)
- **Batch Selection** — Select entire folders to transcode whole seasons or libraries at once
- **Scheduling** — Restrict transcoding to specific hours (e.g., overnight only)
- **Quality Control** — Adjustable CRF for fine-tuned compression
- **Push Notifications** — Pushover alerts when your queue completes
- **Smart Skipping** — Automatically skips files already in target codec/resolution

---

## Screenshot

<div align="center">
  <img src="docs/screenshot.png" alt="Shrinkray UI" width="800">
  <p><em>Clean, focused interface — browse, select, transcode</em></p>
</div>

---

## Quick Start

### Unraid (Community Applications)

1. Search **"Shrinkray"** in Community Applications
2. Install and configure paths:
   - `/config` → Your appdata location
   - `/media` → Your media library
3. For GPU acceleration, pass through your GPU device (see [Hardware Acceleration](#hardware-acceleration))
4. Open the WebUI at port **8080**

### Docker Compose

```yaml
services:
  shrinkray:
    image: ghcr.io/gwlsn/shrinkray:latest
    container_name: shrinkray
    ports:
      - 8080:8080
    volumes:
      - /path/to/config:/config
      - /path/to/media:/media
      - /path/to/fast/storage:/temp  # Optional: SSD for temp files
    restart: unless-stopped
```

### Docker CLI

```bash
docker run -d \
  --name shrinkray \
  -p 8080:8080 \
  -v /path/to/config:/config \
  -v /path/to/media:/media \
  ghcr.io/gwlsn/shrinkray:latest
```

---

## Presets

| Preset | Codec | Description | Theoretical Savings |
|--------|-------|-------------|-----------------|
| **Compress (HEVC)** | H.265 | Re-encode to HEVC | 40–60% smaller |
| **Compress (AV1)** | AV1 | Re-encode to HEVC | 50–70% smaller |
| **1080p** | HEVC | Downscale 4K → 1080p | 60–80% smaller |
| **720p** | HEVC | Downscale to 720p | 70–85% smaller |

All presets copy audio and subtitles unchanged (stream copy).

---

## FFmpeg Options Reference

Shrinkray automatically selects the best available encoder for your hardware. Below are the exact FFmpeg settings used.

### HEVC Encoders

| Encoder | FFmpeg Encoder | Quality Flag | Default Value | Additional Args |
|---------|----------------|--------------|---------------|-----------------|
| **Software** | `libx265` | `-crf` | 26 | `-preset medium` |
| **VideoToolbox** | `hevc_videotoolbox` | `-b:v` | 35% of source bitrate | `-allow_sw 1` |
| **NVENC** | `hevc_nvenc` | `-cq` | 28 | `-preset p4 -tune hq -rc vbr` |
| **Quick Sync** | `hevc_qsv` | `-global_quality` | 27 | `-preset medium` |
| **VAAPI** | `hevc_vaapi` | `-qp` | 27 | — |

### AV1 Encoders

| Encoder | FFmpeg Encoder | Quality Flag | Default Value | Additional Args |
|---------|----------------|--------------|---------------|-----------------|
| **Software** | `libsvtav1` | `-crf` | 35 | `-preset 6` |
| **VideoToolbox** | `av1_videotoolbox` | `-b:v` | 25% of source bitrate | `-allow_sw 1` |
| **NVENC** | `av1_nvenc` | `-cq` | 32 | `-preset p4 -tune hq -rc vbr` |
| **Quick Sync** | `av1_qsv` | `-global_quality` | 32 | `-preset medium` |
| **VAAPI** | `av1_vaapi` | `-qp` | 32 | — |

> **Note:** VideoToolbox uses dynamic bitrate calculation instead of CRF. Target bitrate is calculated as a percentage of source bitrate, clamped between 500 kbps and 15,000 kbps.

### Hardware Decoding

When hardware encoding is active, Shrinkray enables hardware decoding for a full GPU pipeline:

| Platform | Input Args |
|----------|------------|
| **VideoToolbox** | `-hwaccel videotoolbox` |
| **NVENC** | `-hwaccel cuda -hwaccel_output_format cuda` |
| **Quick Sync** | `-hwaccel qsv -hwaccel_output_format qsv` |
| **VAAPI** | `-vaapi_device /dev/dri/renderD128 -hwaccel vaapi -hwaccel_output_format vaapi` |

### Scaling Filters (1080p/720p Presets)

| Platform | Scale Filter |
|----------|--------------|
| **Software** | `scale=-2:'min(ih,1080)'` |
| **NVENC** | `scale_cuda=-2:'min(ih,1080)'` |
| **Quick Sync** | `scale_qsv=-2:'min(ih,1080)'` |
| **VAAPI** | `scale_vaapi=w=-2:h='min(ih,1080)'` |
| **VideoToolbox** | `scale=-2:'min(ih,1080)'` (CPU) |

---

## Hardware Acceleration

Shrinkray automatically detects and uses the best available hardware encoder. No configuration required—just pass through your GPU.

### Supported Hardware

| Platform | Requirements | Docker Flags |
|----------|--------------|--------------|
| **NVIDIA (NVENC)** | GTX 1050+ / RTX series | `--runtime=nvidia --gpus all` |
| **Intel (Quick Sync)** | 6th gen+ CPU or Arc GPU | `--device /dev/dri:/dev/dri` |
| **AMD (VAAPI)** | Polaris+ GPU on Linux | `--device /dev/dri:/dev/dri` |
| **Apple (VideoToolbox)** | Any Mac (M1/M2/M3 or Intel) | Native (no Docker needed) |

### Unraid GPU Passthrough

**NVIDIA:**
1. Install the **Nvidia-Driver** plugin from Community Applications
2. Add to container Extra Parameters: `--runtime=nvidia --gpus all`

**Intel / AMD:**
1. Add to container Extra Parameters: `--device /dev/dri:/dev/dri`

### AV1 Hardware Requirements

AV1 hardware encoding requires newer GPUs:

| Platform | Minimum Hardware |
|----------|------------------|
| **NVIDIA** | RTX 40 series (Ada Lovelace) |
| **Intel** | Arc GPUs, Intel Gen 14+ iGPUs |
| **Apple** | M3 chip or newer |
| **AMD** | RX 7000 series (RDNA 3) |

### Verifying Detection

Open logs for Shrinkray, all the detected encoders are shown and the currently selected encoders have an asterisk.
A "HW" or "SW" badge will appear on jobs in your queue to let you know if they are being software or hardware transcoded.

---

## Scheduling

Restrict transcoding to specific hours to reduce system load during the day.

1. Open **Settings** (gear icon)
2. Enable **Schedule transcoding**
3. Set start and end hours (e.g., 22:00 – 06:00 for overnight)

**Behavior:**
- Jobs can always be added to the queue
- Transcoding only runs during the allowed window
- Running jobs complete even if the window closes
- Jobs automatically resume when the window reopens

---

## Configuration

Configuration is stored in `/config/shrinkray.yaml`. Most settings are available in the WebUI.

| Setting | Default | Description |
|---------|---------|-------------|
| `media_path` | `/media` | Root directory to browse |
| `temp_path` | *(empty)* | Fast storage for temp files (SSD recommended) |
| `original_handling` | `replace` | `replace` = delete original, `keep` = rename to `.old` |
| `workers` | `1` | Concurrent transcode jobs (1–6) |
| `quality_hevc` | `0` | CRF override for HEVC (0 = default, range: 15–40) |
| `quality_av1` | `0` | CRF override for AV1 (0 = default, range: 20–50) |
| `schedule_enabled` | `false` | Enable time-based scheduling |
| `schedule_start_hour` | `22` | Hour transcoding may start (0–23) |
| `schedule_end_hour` | `6` | Hour transcoding must stop (0–23) |
| `pushover_user_key` | *(empty)* | Pushover user key for notifications |
| `pushover_app_token` | *(empty)* | Pushover app token for notifications |

### Example Configuration

```yaml
media_path: /media
temp_path: /tmp/shrinkray
original_handling: replace
workers: 2
quality_hevc: 24
schedule_enabled: true
schedule_start_hour: 22
schedule_end_hour: 6
```

---

## Pushover Notifications

Get push notifications when your transcode queue completes.

1. Create an application at [pushover.net](https://pushover.net)
2. Copy your **User Key** and **API Token**
3. Enter both in Shrinkray → Settings
4. Enable **"Notify when done"** before starting jobs

Notifications include: completed/failed job counts, total space saved.

---

## Building from Source

```bash
# Clone the repository
git clone https://github.com/gwlsn/shrinkray.git
cd shrinkray

# Build
go build -o shrinkray ./cmd/shrinkray

# Run locally
./shrinkray -media /path/to/media

# Run tests
go test ./...
```

**Requirements:**
- Go 1.22+
- FFmpeg with HEVC/AV1 encoder support

---

## FAQ

### Why is CPU usage 10–40% with hardware encoding?

This is normal. The GPU handles video encoding/decoding, but the CPU still handles:
- Demuxing (parsing input container)
- Muxing (writing output container)
- Audio/subtitle stream copying
- FFmpeg process overhead

### Why did Shrinkray skip some files?

Files are automatically skipped if:
- Already encoded in the target codec (HEVC/AV1)
- Already at or below the target resolution (1080p/720p presets)


### What happens if the transcoded file is larger?

Shrinkray rejects files where the output is larger than the original. The original file is kept unchanged.

---

## License

MIT License — see [LICENSE](LICENSE) for details.
