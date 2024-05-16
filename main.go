package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/KarlGW/azcfg"
	"github.com/KarlGW/azcfg/authopts"
	"golang.org/x/sync/errgroup"
)

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("failed to create azure credential: %w", err)
	}

	cfg := &config{}
	mu := &sync.RWMutex{}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				var err error
				mu.Lock()
				cfg, err = newConfig(ctx, cred)
				mu.Unlock()

				if err != nil {
					return fmt.Errorf("failed to create config: %w", err)
				}
			}
		}
	})

	g.Go(func() error {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				mu.RLock()
				fmt.Printf("Config:\n%s\n", cfg)
				mu.RUnlock()
			}
		}
	})

	return g.Wait()
}

type config struct {
	KeyVaultFoo string `secret:"ze-kv-foo"`
	KeyVaultBar string `secret:"ze-kv-bar"`

	AppConfigFoo string `setting:"ze-ac-foo"`
	AppConfigBar string `setting:"ze-ac-bar"`
}

func (cfg *config) String() string {
	return fmt.Sprintf("\tKeyVaultFoo=%s\n\tKeyVaultBar=%s\n\tAppConfigFoo=%s\n\tAppConfigBar=%s",
		cfg.KeyVaultFoo, cfg.KeyVaultBar, cfg.AppConfigFoo, cfg.AppConfigBar)
}

func newConfig(ctx context.Context, cred *azidentity.DefaultAzureCredential) (*config, error) {
	cfg := &config{}
	err := azcfg.Parse(ctx, cfg,
		authopts.WithTokenCredential(cred),
		azcfg.WithKeyVault("kv-lab-sc-azcfg"),
		azcfg.WithAppConfiguration("ac-lab-sc-azcfg"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}
