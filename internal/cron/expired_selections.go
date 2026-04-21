package cron

import (
	"context"
	"time"

	"locker/internal/observability"
	"locker/internal/service"
)

func StartExpiredSelectionCleanup(ctx context.Context, svc *service.RentalFlowService, logger *observability.Logger, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("expired selection cleanup stopped")
				return
			case <-ticker.C:
				ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
				if err := svc.CleanupExpiredSelections(ctxWithTimeout); err != nil {
					logger.Error("expired selection cleanup failed", err)
				}
				cancel()
			}
		}
	}()

	logger.Info("expired selection cleanup started", "interval", interval.String())
}
