# Shrinkray

Video transcoding tool for reducing media library file sizes.

## Installation

### From Source

```bash
go build -o shrinkray ./cmd/shrinkray
./shrinkray -media /path/to/media
```

### Docker

```bash
docker run -d \
  --name shrinkray \
  -p 8080:8080 \
  -v /path/to/config:/config \
  -v /path/to/media:/media \
  ghcr.io/graysonwilson/shrinkray
```

Access the web interface at `http://localhost:8080`.

## Usage

1. Browse to a folder containing video files
2. Select files
3. Choose a preset
4. Click "Transcode"

Transcoding progress is displayed in the queue panel. Completed jobs show space saved and transcode duration.

If a transcoded file is larger than the original, the job fails and the original is preserved.

## Presets

| Preset | Codec | Description |
|--------|-------|-------------|
| Compress (HEVC) | H.265 | 50-60% size savings compared to H.264|
| Compress (AV1) | AV1 | 60-75% size savings compared to H.264|
| Downscale to 1080p | H.265 | Scales to 1080p max height |
| Downscale to 720p | H.265 | Scales to 720p max height |

All presets copy audio and subtitle streams unchanged.

## Hardware Acceleration

Shrinkray automatically detects and uses available hardware encoders:

- **macOS**: VideoToolbox (HEVC, AV1 on M3+)
- **NVIDIA**: NVENC
- **Intel**: Quick Sync Video (QSV)
- **AMD/Intel (Linux)**: VAAPI

Falls back to software encoding (libx265, libsvtav1) if no hardware encoder is available.

## Configuration

Settings are available in the web interface. Configuration is stored in `config/shrinkray.yaml`:

```yaml
media_path: /media
original_handling: replace  # or "keep" to rename originals to .old
workers: 1
ffmpeg_path: ffmpeg
ffprobe_path: ffprobe
```

## Requirements

- FFmpeg with HEVC and/or AV1 encoder support
- Go 1.22+ (if building from source)
