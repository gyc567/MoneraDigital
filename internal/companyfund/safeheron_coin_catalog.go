package companyfund

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"monera-digital/internal/adaptiveschedule"
	"monera-digital/internal/safeheron"
)

const (
	defaultSafeheronCoinCatalogRefreshInterval   = 6 * time.Hour
	defaultSafeheronCoinCatalogColdRetryInterval = time.Minute
)

// ErrSafeheronCoinNotFound is returned when an exact coin-key lookup misses.
var ErrSafeheronCoinNotFound = errors.New("safeheron coin not found")

// SafeheronCoinCatalogColdMissError distinguishes a lookup made before the
// first successful complete catalog refresh from an ordinary loaded miss.
type SafeheronCoinCatalogColdMissError struct {
	CoinKey string
}

func (e *SafeheronCoinCatalogColdMissError) Error() string {
	return fmt.Sprintf("%v: catalog is not loaded for %q", ErrSafeheronCoinNotFound, e.CoinKey)
}

func (e *SafeheronCoinCatalogColdMissError) Is(target error) bool {
	return target == ErrSafeheronCoinNotFound
}

// SafeheronCoinLister is the provider surface required by the catalog.
type SafeheronCoinLister interface {
	ListCoin(context.Context) ([]safeheron.Coin, error)
}

type SafeheronCoinLookup interface {
	Lookup(coinKey string) (safeheron.Coin, error)
}

// SafeheronCoinCatalogConfig controls the periodic complete refresh cadence.
type SafeheronCoinCatalogConfig struct {
	RefreshInterval   time.Duration
	ColdRetryInterval time.Duration
}

type safeheronCoinCatalogSnapshot struct {
	coins   map[string]safeheron.Coin
	ordered []safeheron.Coin
}

// SafeheronCoinCatalog publishes only complete, validated provider snapshots.
type SafeheronCoinCatalog struct {
	client            SafeheronCoinLister
	interval          time.Duration
	coldRetryInterval time.Duration
	snapshot          atomic.Pointer[safeheronCoinCatalogSnapshot]

	refreshMu sync.Mutex
	runMu     sync.Mutex
	running   bool
	runCancel context.CancelFunc
	runDone   chan struct{}
}

func NewSafeheronCoinCatalog(client SafeheronCoinLister, config SafeheronCoinCatalogConfig) (*SafeheronCoinCatalog, error) {
	if client == nil {
		return nil, fmt.Errorf("Safeheron coin client is required")
	}
	interval := config.RefreshInterval
	if interval == 0 {
		interval = defaultSafeheronCoinCatalogRefreshInterval
	}
	if interval <= 0 {
		return nil, fmt.Errorf("Safeheron coin catalog refresh interval must be positive")
	}
	coldRetryInterval := config.ColdRetryInterval
	if coldRetryInterval == 0 {
		coldRetryInterval = defaultSafeheronCoinCatalogColdRetryInterval
	}
	if coldRetryInterval <= 0 {
		return nil, fmt.Errorf("Safeheron coin catalog cold retry interval must be positive")
	}
	return &SafeheronCoinCatalog{
		client:            client,
		interval:          interval,
		coldRetryInterval: coldRetryInterval,
	}, nil
}

func (c *SafeheronCoinCatalog) RefreshInterval() time.Duration {
	if c == nil || c.interval <= 0 {
		return defaultSafeheronCoinCatalogRefreshInterval
	}
	return c.interval
}

func (c *SafeheronCoinCatalog) Refresh(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("Safeheron coin catalog is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()
	coins, err := c.client.ListCoin(ctx)
	if err != nil {
		return fmt.Errorf("refresh Safeheron coin catalog: %w", err)
	}
	snapshot, err := buildSafeheronCoinCatalogSnapshot(coins)
	if err != nil {
		return err
	}
	c.snapshot.Store(snapshot)
	return nil
}

func buildSafeheronCoinCatalogSnapshot(coins []safeheron.Coin) (*safeheronCoinCatalogSnapshot, error) {
	if len(coins) == 0 {
		return nil, fmt.Errorf("Safeheron coin catalog is empty")
	}
	byKey := make(map[string]safeheron.Coin, len(coins))
	ordered := make([]safeheron.Coin, 0, len(coins))
	for _, coin := range coins {
		if coin.CoinKey == "" || coin.CoinKey != strings.TrimSpace(coin.CoinKey) {
			return nil, fmt.Errorf("Safeheron coin catalog contains an invalid exact CoinKey")
		}
		previous, exists := byKey[coin.CoinKey]
		if exists {
			if previous != coin {
				return nil, fmt.Errorf("Safeheron coin %q has conflicting immutable metadata", coin.CoinKey)
			}
			continue
		}
		byKey[coin.CoinKey] = coin
		ordered = append(ordered, coin)
	}
	return &safeheronCoinCatalogSnapshot{coins: byKey, ordered: ordered}, nil
}

func (c *SafeheronCoinCatalog) Lookup(coinKey string) (safeheron.Coin, error) {
	if c == nil {
		return safeheron.Coin{}, &SafeheronCoinCatalogColdMissError{CoinKey: coinKey}
	}
	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return safeheron.Coin{}, &SafeheronCoinCatalogColdMissError{CoinKey: coinKey}
	}
	coin, found := snapshot.coins[coinKey]
	if !found {
		return safeheron.Coin{}, fmt.Errorf("%w: %q", ErrSafeheronCoinNotFound, coinKey)
	}
	return coin, nil
}

func (c *SafeheronCoinCatalog) Snapshot() []safeheron.Coin {
	if c == nil {
		return nil
	}
	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return nil
	}
	return slices.Clone(snapshot.ordered)
}

func (c *SafeheronCoinCatalog) Start(parent context.Context) {
	if c == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	c.runMu.Lock()
	if c.running {
		c.runMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	c.running, c.runCancel, c.runDone = true, cancel, done
	interval := c.interval
	c.runMu.Unlock()
	go c.run(ctx, done, interval)
}

func (c *SafeheronCoinCatalog) run(ctx context.Context, done chan struct{}, interval time.Duration) {
	defer recoverCompanyFundTask("safeheron_coin_catalog")
	defer func() {
		c.runMu.Lock()
		if c.runDone == done {
			c.running, c.runCancel, c.runDone = false, nil, nil
		}
		c.runMu.Unlock()
		close(done)
	}()
	loop, err := adaptiveschedule.New(adaptiveschedule.Config{
		Name:    "company-fund-safeheron-coin-catalog",
		MinIdle: interval,
		MaxIdle: adaptiveschedule.MaxIdleAtLeast(interval),
		OnCycle: func(_ adaptiveschedule.CycleOutcome, err error, elapsed time.Duration) {
			loaded, coinCount := c.snapshotState()
			if err != nil {
				log.Printf(
					"company-fund Safeheron coin catalog refresh: result=failed loaded=%t coinCount=%d elapsed=%s",
					loaded, coinCount, elapsed.Round(time.Millisecond),
				)
				return
			}
			log.Printf(
				"company-fund Safeheron coin catalog refresh: result=success loaded=%t coinCount=%d elapsed=%s",
				loaded, coinCount, elapsed.Round(time.Millisecond),
			)
		},
	}, func(ctx context.Context) (adaptiveschedule.CycleOutcome, error) {
		err := c.Refresh(ctx)
		outcome := adaptiveschedule.CycleOutcome{}
		if err != nil {
			loaded, _ := c.snapshotState()
			if !loaded {
				outcome.NextDue = time.Now().Add(c.coldRetryInterval)
			}
		}
		return outcome, err
	})
	if err != nil {
		return
	}
	loop.Run(ctx)
}

func (c *SafeheronCoinCatalog) snapshotState() (bool, int) {
	if c == nil {
		return false, 0
	}
	snapshot := c.snapshot.Load()
	if snapshot == nil {
		return false, 0
	}
	return true, len(snapshot.ordered)
}

func (c *SafeheronCoinCatalog) Stop() {
	if c == nil {
		return
	}
	c.runMu.Lock()
	cancel, done := c.runCancel, c.runDone
	c.runMu.Unlock()
	if cancel == nil || done == nil {
		return
	}
	cancel()
	<-done
}
