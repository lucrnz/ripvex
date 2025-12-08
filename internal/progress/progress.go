package progress

import (
	"log/slog"
	"time"

	"github.com/lucrnz/ripvex/internal/util"
)

// Bar emits structured progress logs for known and unknown sizes.
type Bar struct {
	Total          int64
	MilestoneStep  int           // percentage step for known sizes
	ByteStep       int64         // byte step for unknown sizes
	RenderInterval time.Duration // interval for interval-based logs
	Logger         *slog.Logger
	Quiet          bool

	downloaded        int64
	nextMilestone     int
	nextByteLog       int64
	done              chan struct{} // signals completion
	lastIntervalBytes int64
	lastIntervalTime  time.Time
}

// New creates a progress bar instance with sane defaults.
func New(total int64, step int, byteStep int64, interval time.Duration, logger *slog.Logger, quiet bool) *Bar {
	if step <= 0 {
		step = 5
	}
	if step > 50 {
		step = 50
	}
	if byteStep <= 0 {
		byteStep = 25 * 1024 * 1024 // 25MB default fallback
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	if logger == nil {
		logger = slog.Default()
	}

	next := step
	var nextBytes int64 = byteStep

	return &Bar{
		Total:          total,
		MilestoneStep:  step,
		ByteStep:       byteStep,
		RenderInterval: interval,
		Logger:         logger,
		Quiet:          quiet,
		nextMilestone:  next,
		nextByteLog:    nextBytes,
		done:           make(chan struct{}),
	}
}

// Update records progress and handles rendering/logging when thresholds are reached.
func (b *Bar) Update(n int64) {
	if n <= 0 {
		return
	}
	b.downloaded += n

	if !b.Quiet {
		if b.Total > 0 {
			b.maybeLogMilestone()
		} else {
			b.maybeLogBytes()
		}
	}
}

// Start begins interval-based logging in a goroutine
func (b *Bar) Start() {
	if b.Quiet || b.Logger == nil || b.RenderInterval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(b.RenderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.logCurrentProgress()
			case <-b.done:
				// Log final progress before stopping
				b.logCurrentProgress()
				return
			}
		}
	}()
}

// Stop ends interval-based logging
func (b *Bar) Stop() {
	if b.done != nil {
		close(b.done)
	}
}

func (b *Bar) logCurrentProgress() {
	// Throttle: only log if bytes changed since last interval
	if b.downloaded == b.lastIntervalBytes {
		return
	}

	now := time.Now()
	var speedBytesPerSec int64
	if !b.lastIntervalTime.IsZero() {
		elapsed := now.Sub(b.lastIntervalTime).Seconds()
		if elapsed > 0 {
			speedBytesPerSec = int64(float64(b.downloaded-b.lastIntervalBytes) / elapsed)
			if speedBytesPerSec < 0 {
				speedBytesPerSec = 0
			}
		}
	}
	speedHuman := util.HumanReadableBytes(speedBytesPerSec) + "/s"

	if b.Total > 0 {
		b.Logger.Info("download_progress",
			"percent", int(b.percent()),
			"downloaded_bytes", b.downloaded,
			"downloaded", util.HumanReadableBytes(b.downloaded),
			"total_bytes", b.Total,
			"total", util.HumanReadableBytes(b.Total),
			"speed_bytes_per_sec", speedBytesPerSec,
			"speed", speedHuman,
		)
	} else {
		b.Logger.Info("download_progress",
			"downloaded_bytes", b.downloaded,
			"downloaded", util.HumanReadableBytes(b.downloaded),
			"speed_bytes_per_sec", speedBytesPerSec,
			"speed", speedHuman,
		)
	}
	b.lastIntervalTime = now
	b.lastIntervalBytes = b.downloaded
}

func (b *Bar) maybeLogMilestone() {
	if b.Logger == nil || b.Total <= 0 {
		return
	}
	pct := int(b.percent())
	for pct >= b.nextMilestone && b.nextMilestone <= 100 {
		b.Logger.Info("download_progress",
			"percent", b.nextMilestone,
			"downloaded_bytes", b.downloaded,
			"downloaded", util.HumanReadableBytes(b.downloaded),
			"total_bytes", b.Total,
			"total", util.HumanReadableBytes(b.Total),
		)
		b.nextMilestone += b.MilestoneStep
	}
}

func (b *Bar) maybeLogBytes() {
	if b.Logger == nil || b.nextByteLog <= 0 {
		return
	}
	for b.downloaded >= b.nextByteLog {
		b.Logger.Info("download_progress",
			"downloaded_bytes", b.nextByteLog,
			"downloaded", util.HumanReadableBytes(b.nextByteLog),
		)
		b.nextByteLog += b.ByteStep
	}
}

func (b *Bar) percent() float64 {
	if b.Total <= 0 {
		return 0
	}
	p := (float64(b.downloaded) / float64(b.Total)) * 100
	if p > 100 {
		return 100
	}
	return p
}
