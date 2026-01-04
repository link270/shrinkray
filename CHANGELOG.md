# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [1.3.1] - 2026-01-04

### Fixed
- Quality slider labels were backwards - now correctly show "Higher quality" on left and "Smaller file" on right (#24)

## [1.3.0] - 2026-01-01

### Added
- Scheduling feature to restrict transcoding to specific hours (e.g., overnight only)
- Schedule settings in Settings panel with enable toggle and time selection
- Schedule status display near transcode button when enabled
- Advanced settings section for quality control (CRF values for HEVC and AV1)
- Collapsible advanced settings panel (closed by default)

### Fixed
- Quality input fields now validate on change instead of every keystroke

## [1.2.0] - 2025-12-29

### Added
- Dark mode toggle
- Infinite scroll for queue display (handles thousands of jobs smoothly)
- Real-time progress feedback for batch file processing
- Allow selecting folders with nested videos for batch transcoding
- Queue sorted by status (running jobs first, then pending, then completed)

### Fixed
- UI no longer hangs when adding thousands of files to queue

## [1.1.0] - 2025-12-28

### Added
- Skip files already encoded in target codec (HEVC/AV1) to prevent unnecessary transcoding
- Skip files already at target resolution when using downscale presets (1080p/720p)
- Version number displayed in Settings panel

## [1.0.0] - 2025-12-25

### Added
- Initial public release
- Hardware-accelerated transcoding (VideoToolbox, NVENC, QSV, VAAPI)
- HEVC and AV1 compression presets
- 1080p and 720p downscale presets
- Batch folder selection for entire TV series
- Async job creation to prevent UI freezes
- Pushover notifications when queue completes
- Retry button for failed jobs
- Mobile-responsive stats bar
- Queue persistence across restarts
