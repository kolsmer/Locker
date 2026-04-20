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
				if err := svc.CleanupExpiredSelections(context.Background()); err != nil {
					logger.Error("expired selection cleanup failed", err)
				}
			}
		}
	}()

	logger.Info("expired selection cleanup started", "interval", interval.String())
}
