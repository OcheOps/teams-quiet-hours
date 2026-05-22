package enforcer

import (
	"fmt"
	"time"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
)

type Enforcer struct {
	Runtime platform.Runtime
}

func (e Enforcer) Block(cfg config.Config, dryRun bool) error {
	fmt.Printf("Applying %s mode quiet hours\n", cfg.Mode)
	if err := e.Runtime.Policy.Block(cfg.Browser, dryRun); err != nil {
		return err
	}

	if cfg.Mode == "soft" {
		fmt.Println("Soft mode: browser Teams notifications are blocked; Teams apps are left running.")
		return nil
	}

	if cfg.CloseBrowserTabs {
		if err := e.Runtime.Tabs.CloseTeamsTabs(dryRun); err != nil {
			return err
		}
	} else {
		fmt.Println("Browser tab closing is disabled. Use --close-browser-tabs to opt in where supported.")
	}

	if err := e.Runtime.Process.Stop(dryRun); err != nil {
		return err
	}

	if cfg.Mode == "nuclear" || cfg.BlockNetwork {
		if !cfg.BlockNetwork {
			fmt.Println("Nuclear mode requested, but network blocking is disabled. Use --block-network to opt in.")
			return nil
		}
		if err := e.Runtime.Network.Block(dryRun); err != nil {
			return err
		}
	}
	return nil
}

func (e Enforcer) Allow(cfg config.Config, dryRun bool) error {
	if err := e.Runtime.Policy.Allow(cfg.Browser, dryRun); err != nil {
		return err
	}
	if cfg.BlockNetwork || cfg.Mode == "nuclear" {
		if err := e.Runtime.Network.Allow(dryRun); err != nil {
			return err
		}
	}
	if cfg.LaunchOnAllow {
		if err := e.Runtime.Process.Start(dryRun); err != nil {
			return err
		}
	} else {
		fmt.Println("Teams launch on allow is disabled. Use --launch-on-allow to opt in.")
	}
	return nil
}

func (e Enforcer) Enforce(cfg config.Config, now time.Time, dryRun bool) error {
	if cfg.Paused {
		fmt.Println("teams-quiet-hours is paused; allowing Teams until you run resume.")
		return e.Allow(cfg, dryRun)
	}
	allowed, err := config.IsAllowedNow(cfg, now)
	if err != nil {
		return err
	}
	if allowed {
		fmt.Println("Current time is inside the allowed work window; allowing Teams.")
		return e.Allow(cfg, dryRun)
	}
	fmt.Println("Current time is outside the allowed work window; blocking Teams.")
	return e.Block(cfg, dryRun)
}
