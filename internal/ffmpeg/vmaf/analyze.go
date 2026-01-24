package vmaf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gwlsn/shrinkray/internal/logger"
)

// Analyzer orchestrates VMAF analysis for SmartShrink
type Analyzer struct {
	FFmpegPath     string
	TempDir        string
	SampleDuration int
	FastAnalysis   bool
	VMafThreshold  float64
}

// NewAnalyzer creates a new VMAF analyzer
func NewAnalyzer(ffmpegPath, tempDir string, sampleDuration int, fastAnalysis bool, threshold float64) *Analyzer {
	return &Analyzer{
		FFmpegPath:     ffmpegPath,
		TempDir:        tempDir,
		SampleDuration: sampleDuration,
		FastAnalysis:   fastAnalysis,
		VMafThreshold:  threshold,
	}
}

// Analyze performs full VMAF analysis on a video
// encodeSample is a callback that encodes a sample at the given quality
func (a *Analyzer) Analyze(ctx context.Context, inputPath string, videoDuration time.Duration,
	height int, qRange QualityRange, encodeSample EncodeSampleFunc) (*AnalysisResult, error) {

	if !IsAvailable() {
		return nil, fmt.Errorf("VMAF not available")
	}

	// Create temp directory for this analysis
	analysisDir := filepath.Join(a.TempDir, fmt.Sprintf("vmaf_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(analysisDir, 0755); err != nil {
		return nil, fmt.Errorf("creating analysis dir: %w", err)
	}
	defer os.RemoveAll(analysisDir)

	// Determine sample positions
	positions := SamplePositions(videoDuration, a.FastAnalysis)

	logger.Info("Starting VMAF analysis",
		"input", inputPath,
		"samples", len(positions),
		"threshold", a.VMafThreshold)

	// Extract reference samples
	referenceSamples, err := ExtractSamples(ctx, a.FFmpegPath, inputPath, analysisDir,
		videoDuration, a.SampleDuration, positions)
	if err != nil {
		return nil, fmt.Errorf("extracting samples: %w", err)
	}
	defer CleanupSamples(referenceSamples)

	// Wrap encodeSample to put outputs in our analysis dir
	wrappedEncode := func(ctx context.Context, samplePath string, quality int, modifier float64) (string, error) {
		return encodeSample(ctx, samplePath, quality, modifier)
	}

	// Run binary search
	result, err := BinarySearch(ctx, a.FFmpegPath, referenceSamples, qRange, a.VMafThreshold, height, wrappedEncode)
	if err != nil {
		return nil, fmt.Errorf("binary search: %w", err)
	}

	// No acceptable quality found
	if result == nil {
		return &AnalysisResult{
			ShouldSkip: true,
			SkipReason: "Already optimized",
		}, nil
	}

	// Check if we should expand from fast analysis to full
	if a.FastAnalysis && len(positions) == 1 {
		// If score is within 5 points of threshold, do full analysis
		if result.VMafScore < a.VMafThreshold+5 {
			logger.Info("Expanding to full analysis (score near threshold)",
				"score", result.VMafScore,
				"threshold", a.VMafThreshold)

			// Re-run with full positions
			fullPositions := []float64{0.25, 0.5, 0.75}
			fullSamples, err := ExtractSamples(ctx, a.FFmpegPath, inputPath, analysisDir,
				videoDuration, a.SampleDuration, fullPositions)
			if err != nil {
				return nil, fmt.Errorf("extracting full samples: %w", err)
			}
			defer CleanupSamples(fullSamples)

			result, err = BinarySearch(ctx, a.FFmpegPath, fullSamples, qRange, a.VMafThreshold, height, wrappedEncode)
			if err != nil {
				return nil, fmt.Errorf("full binary search: %w", err)
			}

			if result == nil {
				return &AnalysisResult{
					ShouldSkip: true,
					SkipReason: "Already optimized",
				}, nil
			}
		}
	}

	return &AnalysisResult{
		OptimalCRF:  result.Quality,
		QualityMod:  result.Modifier,
		VMafScore:   result.VMafScore,
		SamplesUsed: len(positions),
	}, nil
}
