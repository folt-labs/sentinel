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

// Daemon manages the agent lifecycle
type Daemon struct {
	cfg        *config.Config
	collectors []Collector
	client     *transport.Client
	events     chan types.Event
	wg         sync.WaitGroup
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
		events: make(chan types.Event, 1000),
	}

	// Initialize collectors based on config
	d.initCollectors()

	return d, nil
}

func (d *Daemon) initCollectors() {
	if d.cfg.Collectors.SSH.Enabled {
		d.collectors = append(d.collectors, auth.NewSSHCollector(d.cfg.Collectors.SSH.LogFiles))
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

	log.Printf("Initialized %d collectors", len(d.collectors))
}

// Run starts the daemon
func (d *Daemon) Run(ctx context.Context) error {
	// Start event sender
	d.wg.Add(1)
	go d.eventSender(ctx)

	// Start collection scheduler
	d.wg.Add(1)
	go d.collectionLoop(ctx)

	<-ctx.Done()
	return nil
}

func (d *Daemon) collectionLoop(ctx context.Context) {
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

func (d *Daemon) eventSender(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.cfg.Agent.SendEvery)
	defer ticker.Stop()

	var batch []types.Event

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
			// Send if batch is large enough
			if len(batch) >= 100 {
				d.sendBatch(batch)
				batch = nil
			}
		case <-ticker.C:
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
		log.Printf("Sent %d events", len(events))
	}
}

// Shutdown gracefully stops the daemon
func (d *Daemon) Shutdown() error {
	d.wg.Wait()
	return d.client.Close()
}
