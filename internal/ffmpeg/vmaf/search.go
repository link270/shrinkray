package vmaf

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gwlsn/shrinkray/internal/logger"
)

const (
	// minModRange is the minimum modifier range worth searching.
	// Below 5% bitrate difference, quality changes are imperceptible.
	minModRange = 0.05

	// maxSearchIters caps search iterations (excluding initial bound probes)
	// to prevent thrashing near the boundary due to sampling noise.
	maxSearchIters = 4

	// Tolerance settings for early termination.
	// Search stops when bounds are tight AND best score is within tolerance above threshold.
	// This prevents over-searching when we've found a "good enough" result.
	baseTolerance = 0.5
	toleranceStep = 0.5
)

// QualityRange defines the search bounds for an encoder
type QualityRange struct {
	Min         int     // Minimum quality value (best quality)
	Max         int     // Maximum quality value (most compression)
	UsesBitrate bool    // True for VideoToolbox (bitrate modifier)
	MinMod      float64 // Min bitrate modifier (most compression)
	MaxMod      float64 // Max bitrate modifier (best quality)
}

// SearchResult holds the result of the search
type SearchResult struct {
	Quality    int     // CRF/CQ/QP value
	Modifier   float64 // Bitrate modifier (for VideoToolbox)
	VMafScore  float64 // Achieved VMAF score
	Iterations int     // Number of quality levels tested (each test encodes all samples)
}

// EncodeSampleFunc is a function that encodes a sample at the given quality
// and returns the path to the encoded file
type EncodeSampleFunc func(ctx context.Context, samplePath string, quality int, modifier float64) (string, error)

// BinarySearch finds the optimal quality setting via interpolated binary search.
// Uses linear interpolation between data points to converge faster than pure binary search.
// When tonemap is provided and enabled, scoring uses HDR-aware comparison.
func BinarySearch(ctx context.Context, ffmpegPath string, referenceSamples []*Sample,
	qRange QualityRange, threshold float64, height int, tonemap *TonemapConfig, encodeSample EncodeSampleFunc) (*SearchResult, error) {

	// Validate inputs
	if len(referenceSamples) == 0 {
		return nil, fmt.Errorf("no reference samples provided")
	}

	// Create the encode+score function that both search modes share
	scorer := newSampleScorer(ctx, ffmpegPath, referenceSamples, height, tonemap, encodeSample)

	if qRange.UsesBitrate {
		if qRange.MinMod >= qRange.MaxMod {
			return nil, fmt.Errorf("invalid bitrate range: min %.3f >= max %.3f", qRange.MinMod, qRange.MaxMod)
		}
		return interpolatedSearchBitrate(scorer, qRange, threshold)
	}

	if qRange.Min >= qRange.Max {
		return nil, fmt.Errorf("invalid CRF range: min %d >= max %d", qRange.Min, qRange.Max)
	}
	return interpolatedSearchCRF(scorer, qRange, threshold)
}

// sampleScorer handles encoding and VMAF scoring
type sampleScorer struct {
	ctx              context.Context
	ffmpegPath       string
	referenceSamples []*Sample
	height           int
	tonemap          *TonemapConfig
	encodeSample     EncodeSampleFunc
	testCount        int // Number of quality levels tested
}

func newSampleScorer(ctx context.Context, ffmpegPath string, referenceSamples []*Sample,
	height int, tonemap *TonemapConfig, encodeSample EncodeSampleFunc) *sampleScorer {
	return &sampleScorer{
		ctx:              ctx,
		ffmpegPath:       ffmpegPath,
		referenceSamples: referenceSamples,
		height:           height,
		tonemap:          tonemap,
		encodeSample:     encodeSample,
	}
}

// scoreCRF encodes at a CRF value and returns the VMAF score
func (s *sampleScorer) scoreCRF(crf int) (float64, error) {
	s.testCount++
	encodeStart := time.Now()

	distortedSamples := make([]*Sample, 0, len(s.referenceSamples))
	var encodeErr error

	for i, ref := range s.referenceSamples {
		distPath, err := s.encodeSample(s.ctx, ref.Path, crf, 0)
		if err != nil {
			encodeErr = fmt.Errorf("encoding sample %d at CRF %d: %w", i, crf, err)
			break
		}
		distortedSamples = append(distortedSamples, &Sample{Path: distPath})
	}

	// Cleanup on encode failure
	if encodeErr != nil {
		CleanupSamples(distortedSamples)
		return 0, encodeErr
	}

	encodeDuration := time.Since(encodeStart)

	scoreStart := time.Now()
	score, err := ScoreSamples(s.ctx, s.ffmpegPath, s.referenceSamples, distortedSamples, s.height, s.tonemap)
	scoreDuration := time.Since(scoreStart)

	CleanupSamples(distortedSamples)

	if err != nil {
		return 0, fmt.Errorf("scoring at CRF %d: %w", crf, err)
	}

	logger.Info("VMAF search iteration",
		"crf", crf,
		"vmaf", fmt.Sprintf("%.2f", score),
		"encode_time", encodeDuration.String(),
		"score_time", scoreDuration.String())

	return score, nil
}

// scoreModifier encodes at a bitrate modifier and returns the VMAF score
func (s *sampleScorer) scoreModifier(mod float64) (float64, error) {
	s.testCount++
	encodeStart := time.Now()

	distortedSamples := make([]*Sample, 0, len(s.referenceSamples))
	var encodeErr error

	for i, ref := range s.referenceSamples {
		distPath, err := s.encodeSample(s.ctx, ref.Path, 0, mod)
		if err != nil {
			encodeErr = fmt.Errorf("encoding sample %d at modifier %.3f: %w", i, mod, err)
			break
		}
		distortedSamples = append(distortedSamples, &Sample{Path: distPath})
	}

	// Cleanup on encode failure
	if encodeErr != nil {
		CleanupSamples(distortedSamples)
		return 0, encodeErr
	}

	encodeDuration := time.Since(encodeStart)

	scoreStart := time.Now()
	score, err := ScoreSamples(s.ctx, s.ffmpegPath, s.referenceSamples, distortedSamples, s.height, s.tonemap)
	scoreDuration := time.Since(scoreStart)

	CleanupSamples(distortedSamples)

	if err != nil {
		return 0, fmt.Errorf("scoring at modifier %.3f: %w", mod, err)
	}

	logger.Info("VMAF search iteration",
		"modifier", fmt.Sprintf("%.3f", mod),
		"vmaf", fmt.Sprintf("%.2f", score),
		"encode_time", encodeDuration.String(),
		"score_time", scoreDuration.String())

	return score, nil
}

// interpolatedSearchCRF finds optimal CRF using interpolated search.
// For CRF: lower value = better quality, higher value = more compression.
func interpolatedSearchCRF(s *sampleScorer, qRange QualityRange, threshold float64) (*SearchResult, error) {
	// betterCRF gives quality >= threshold, worseCRF gives quality < threshold
	betterCRF := qRange.Min
	worseCRF := qRange.Max

	// Establish bounds by testing extremes
	betterScore, err := s.scoreCRF(betterCRF)
	if err != nil {
		return nil, err
	}
	if betterScore < threshold {
		// Even best quality fails threshold
		return nil, nil
	}

	worseScore, err := s.scoreCRF(worseCRF)
	if err != nil {
		return nil, err
	}
	if worseScore >= threshold {
		// Even max compression meets threshold - use it
		return &SearchResult{
			Quality:    worseCRF,
			VMafScore:  worseScore,
			Iterations: s.testCount,
		}, nil
	}

	// Interpolated search loop
	// searchIter counts iterations within this loop (not including bound probes)
	searchIter := 0
	for searchIter < maxSearchIters {
		searchIter++

		// Check if bounds are adjacent - can't search further
		if worseCRF-betterCRF <= 1 {
			break
		}

		// Calculate next CRF to try
		var nextCRF int
		if searchIter == 1 {
			// First search iteration: bias 80% toward compression for better bracketing
			nextCRF = betterCRF + int(0.8*float64(worseCRF-betterCRF))
		} else {
			// Subsequent iterations: linear interpolation
			nextCRF = interpolateInt(betterCRF, betterScore, worseCRF, worseScore, threshold)
		}

		// Clamp to interior to guarantee progress
		nextCRF = clampInterior(nextCRF, betterCRF, worseCRF)
		if nextCRF <= betterCRF || nextCRF >= worseCRF {
			break // Can't make progress
		}

		nextScore, err := s.scoreCRF(nextCRF)
		if err != nil {
			return nil, err
		}

		// Update bounds based on result
		if nextScore >= threshold {
			betterCRF = nextCRF
			betterScore = nextScore
		} else {
			worseCRF = nextCRF
			worseScore = nextScore
		}

		// Early termination: if bounds are tight and we have a good result
		if shouldTerminateCRF(worseCRF-betterCRF, betterScore, threshold, searchIter) {
			break
		}
	}

	return &SearchResult{
		Quality:    betterCRF,
		VMafScore:  betterScore,
		Iterations: s.testCount,
	}, nil
}

// interpolatedSearchBitrate finds optimal bitrate modifier using interpolated search.
// For bitrate modifier: higher value = better quality, lower value = more compression.
func interpolatedSearchBitrate(s *sampleScorer, qRange QualityRange, threshold float64) (*SearchResult, error) {
	// betterMod gives quality >= threshold, worseMod gives quality < threshold
	// Note: higher modifier = better quality (opposite of CRF)
	betterMod := qRange.MaxMod
	worseMod := qRange.MinMod

	// Establish bounds by testing extremes
	betterScore, err := s.scoreModifier(betterMod)
	if err != nil {
		return nil, err
	}
	if betterScore < threshold {
		// Even best quality fails threshold
		return nil, nil
	}

	worseScore, err := s.scoreModifier(worseMod)
	if err != nil {
		return nil, err
	}
	if worseScore >= threshold {
		// Even max compression meets threshold - use it
		return &SearchResult{
			Modifier:   worseMod,
			VMafScore:  worseScore,
			Iterations: s.testCount,
		}, nil
	}

	// Interpolated search loop
	searchIter := 0
	for searchIter < maxSearchIters {
		searchIter++

		// Check if bounds are too close - can't search further
		if betterMod-worseMod <= minModRange {
			break
		}

		// Calculate next modifier to try
		var nextMod float64
		if searchIter == 1 {
			// First search iteration: bias 80% toward compression for better bracketing
			nextMod = betterMod - 0.8*(betterMod-worseMod)
		} else {
			// Subsequent iterations: linear interpolation
			nextMod = interpolateFloat(betterMod, betterScore, worseMod, worseScore, threshold)
		}

		// Clamp to interior to guarantee progress
		nextMod = clampInteriorFloat(nextMod, worseMod, betterMod)
		if nextMod <= worseMod+minModRange/2 || nextMod >= betterMod-minModRange/2 {
			break // Can't make progress
		}

		nextScore, err := s.scoreModifier(nextMod)
		if err != nil {
			return nil, err
		}

		// Update bounds based on result
		if nextScore >= threshold {
			betterMod = nextMod
			betterScore = nextScore
		} else {
			worseMod = nextMod
			worseScore = nextScore
		}

		// Early termination: if bounds are tight and we have a good result
		if shouldTerminateBitrate(betterMod-worseMod, betterScore, threshold, searchIter) {
			break
		}
	}

	return &SearchResult{
		Modifier:   betterMod,
		VMafScore:  betterScore,
		Iterations: s.testCount,
	}, nil
}

// shouldTerminateCRF checks if CRF search should stop early.
// Only terminates if bounds are tight AND best score is just above threshold (within tolerance).
// This prevents wasting iterations when we've found a result that's "good enough".
func shouldTerminateCRF(rangeSize int, bestScore, threshold float64, searchIter int) bool {
	// Don't terminate early on first iteration - need better bracketing
	if searchIter < 2 {
		return false
	}

	// Growing tolerance: more lenient in later iterations
	tolerance := baseTolerance + float64(searchIter-1)*toleranceStep

	// Terminate if:
	// 1. Bounds are close (within 3 CRF points)
	// 2. Best score meets threshold
	// 3. Best score is not excessively above threshold (within tolerance)
	//    This ensures we've pushed compression reasonably close to the limit
	return rangeSize <= 3 && bestScore >= threshold && bestScore-threshold <= tolerance
}

// shouldTerminateBitrate checks if bitrate search should stop early.
// Same logic as CRF: tight bounds AND score just above threshold.
func shouldTerminateBitrate(rangeSize float64, bestScore, threshold float64, searchIter int) bool {
	// Don't terminate early on first iteration
	if searchIter < 2 {
		return false
	}

	tolerance := baseTolerance + float64(searchIter-1)*toleranceStep

	// Terminate if bounds are close and score is within tolerance above threshold
	return rangeSize <= minModRange*3 && bestScore >= threshold && bestScore-threshold <= tolerance
}

// interpolateInt calculates next CRF using linear interpolation.
// Returns midpoint if interpolation would fail (non-monotonic data).
func interpolateInt(betterVal int, betterScore float64, worseVal int, worseScore, threshold float64) int {
	denom := betterScore - worseScore
	if denom <= 0 {
		// Guard against division by zero / non-monotonic noise
		return (betterVal + worseVal) / 2
	}

	// Linear interpolation: find where threshold falls proportionally
	f := (threshold - worseScore) / denom
	result := float64(worseVal) + f*float64(betterVal-worseVal)
	return int(math.Round(result))
}

// interpolateFloat calculates next modifier using linear interpolation.
// Returns midpoint if interpolation would fail.
func interpolateFloat(betterVal, betterScore, worseVal, worseScore, threshold float64) float64 {
	denom := betterScore - worseScore
	if denom <= 0 {
		return (betterVal + worseVal) / 2
	}

	f := (threshold - worseScore) / denom
	return worseVal + f*(betterVal-worseVal)
}

// clampInterior clamps an int value to strictly inside (low, high)
func clampInterior(val, low, high int) int {
	if val <= low {
		val = low + 1
	}
	if val >= high {
		val = high - 1
	}
	return val
}

// clampInteriorFloat clamps a float value to strictly inside (low, high)
func clampInteriorFloat(val, low, high float64) float64 {
	margin := (high - low) * 0.01 // 1% margin
	if margin < minModRange/2 {
		margin = minModRange / 2
	}
	if val <= low+margin {
		val = low + margin
	}
	if val >= high-margin {
		val = high - margin
	}
	return val
}
