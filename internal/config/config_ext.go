package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
)

func (c *Configuration) Validate() error {
	for i, p := range c.Pattern {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("pattern #%d: %w", i, err)
		}
	}
	return nil
}

func (p *DevicePattern) Validate() error {
	if len(p.AcceptCidr) == 0 {
		return errors.New("device pattern matches nothing (must have at least one accepted_cidr)")
	}
	for _, cidr := range p.AcceptCidr {
		if _, err := netip.ParsePrefix(cidr); err != nil {
			return fmt.Errorf("parsing %s: %w", cidr, err)
		}
	}
	return nil
}

func (p *DevicePattern) MatchesAddress(addr netip.Addr) bool {
	if p == nil {
		return false
	}
	if len(p.AcceptCidr) == 0 {
		return false
	}

	for _, cidr := range p.AcceptCidr {
		pref, err := netip.ParsePrefix(cidr)
		if err != nil {
			slog.Error("Failed to parse CIDR", "cidr", cidr, "error", err)
			continue
		}
		if pref.Contains(addr) {
			return true
		}
	}

	return false
}
