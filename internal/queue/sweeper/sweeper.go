package sweeper

import (
	"context"
	"log"
	"time"

	"github.com/aridsondez/AWS-SQS-LITE/internal/queue/store"
	"golang.org/x/telemetry/counter"
)

type Sweeper struct {
	store store.Store
	interval time.Duration
	stopCh chan struct{}
}


func New(store store.Store, interval time.Duration) *Sweeper {

	return &Sweeper{
		store: store,
		interval: interval,
		stopCh: make(chan struct{}),
	}
}

func (s *Sweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("Sweeper started, interval: %s", s.interval)

	for {

		select{
		case <-ctx.Done():
			log.Printf("Sweeper Stopped (Context Cancelled)")
			return

		case <-s.stopCh:
			log.Printf("Sweeper Stopped(stop signal)")
			return 
		
		case <-ticker.C:
			count, err := s.store.Sweeper(ctx)
			if err != nil {
				log.Printf("Sweeper error: %v", err)
			} else if count > 0 {
				log.Printf("Sweeper processed %d messages", count)
			}
			// If count == 0, silently continue (no messages to process)
		}

	}
}

func (s *Sweeper) Stop() {
	close(s.stopCh)
}