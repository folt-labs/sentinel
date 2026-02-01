package daemon

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/collectors/auth"
	"github.com/folt-labs/sentinel/agent/internal/collectors/files"
	"github.com/folt-labs/sentinel/agent/internal/collectors/system"
	"github.com/folt-labs/sentinel/agent/internal/config"
	"github.com/folt-labs/sentinel/agent/internal/transport"
	"github.com/folt-labs/sentinel/agent/internal/types"
)

// Collector interface for all data collectors
type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]types.Event, error)
}

// StreamingCollector interface for real-time collectors
type StreamingCollector interface {
	Collector
	StartStreaming(ctx context.Context) error
	Stop()
}

// Daemon manages the agent lifecycle
type Daemon struct {
	cfg                 *config.Config
	collectors          []Collector
	streamingCollectors []StreamingCollector
	client              *transport.Client
	events              chan types.Event
	wg                  sync.WaitGroup
}

// New creates a new daemon instance
func New(cfg *config.Config) (*Daemon, error) {
	client, err := transport.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		cfg:    cfg,
		client: client,
		events: make(chan types.Event, 10000), // Larger buffer for high volume
	}

	// Initialize collectors based on config
	d.initCollectors()

	return d, nil
}

func (d *Daemon) initCollectors() {
	if d.cfg.Collectors.SSH.Enabled {
		sshCollector := auth.NewSSHCollector(d.cfg.Collectors.SSH.LogFiles)
		d.collectors = append(d.collectors, sshCollector)
		d.streamingCollectors = append(d.streamingCollectors, sshCollector)
	}

	if d.cfg.Collectors.FileIntegrity.Enabled {
		d.collectors = append(d.collectors, files.NewIntegrityCollector(d.cfg.Collectors.FileIntegrity.Paths))
	}

	if d.cfg.Collectors.Ports.Enabled {
		d.collectors = append(d.collectors, system.NewPortsCollector())
	}

	if d.cfg.Collectors.Resources.Enabled {
		d.collectors = append(d.collectors, system.NewResourcesCollector())
	}

	log.Printf("Initialized %d collectors (%d streaming)", len(d.collectors), len(d.streamingCollectors))
}

// Run starts the daemon
func (d *Daemon) Run(ctx context.Context) error {
	// Start streaming collectors for real-time events
	for _, sc := range d.streamingCollectors {
		if err := sc.StartStreaming(ctx); err != nil {
			log.Printf("Failed to start streaming for %s: %v", sc.Name(), err)
		} else {
			log.Printf("Started real-time streaming for %s", sc.Name())
		}
	}

	// Start real-time event sender (sends immediately, no batching delay)
	d.wg.Add(1)
	go d.realtimeEventSender(ctx)

	// Start periodic collection for non-streaming collectors
	d.wg.Add(1)
	go d.periodicCollectionLoop(ctx)

	<-ctx.Done()
	return nil
}

// periodicCollectionLoop handles periodic collectors (file integrity, ports, resources)
func (d *Daemon) periodicCollectionLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.cfg.Agent.CollectEvery)
	defer ticker.Stop()

	// Run initial collection
	d.runCollection(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.runCollection(ctx)
		}
	}
}

func (d *Daemon) runCollection(ctx context.Context) {
	for _, collector := range d.collectors {
		events, err := collector.Collect(ctx)
		if err != nil {
			log.Printf("Collector %s error: %v", collector.Name(), err)
			continue
		}

		for _, event := range events {
			select {
			case d.events <- event:
			default:
				log.Printf("Event queue full, dropping event from %s", collector.Name())
			}
		}
	}
}

// realtimeEventSender sends events immediately as they arrive
// Uses micro-batching: waits up to 100ms to collect a small batch, then sends
func (d *Daemon) realtimeEventSender(ctx context.Context) {
	defer d.wg.Done()

	var batch []types.Event
	flushTimer := time.NewTimer(100 * time.Millisecond)
	flushTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Send remaining events before shutdown
			if len(batch) > 0 {
				d.sendBatch(batch)
			}
			return

		case event := <-d.events:
			batch = append(batch, event)

			// Start flush timer if this is the first event in batch
			if len(batch) == 1 {
				flushTimer.Reset(100 * time.Millisecond)
			}

			// Send immediately if batch is large enough (for high volume)
			if len(batch) >= 50 {
				flushTimer.Stop()
				d.sendBatch(batch)
				batch = nil
			}

		case <-flushTimer.C:
			// Flush after 100ms even if batch is small
			if len(batch) > 0 {
				d.sendBatch(batch)
				batch = nil
			}
		}
	}
}

func (d *Daemon) sendBatch(events []types.Event) {
	if err := d.client.SendEvents(events); err != nil {
		log.Printf("Failed to send %d events: %v", len(events), err)
		// Events will be queued for retry by transport layer
	} else {
		if len(events) > 0 {
			log.Printf("Sent %d events", len(events))
		}
	}
}

// Shutdown gracefully stops the daemon
func (d *Daemon) Shutdown() error {
	// Stop streaming collectors
	for _, sc := range d.streamingCollectors {
		sc.Stop()
	}

	d.wg.Wait()
	return d.client.Close()
}
