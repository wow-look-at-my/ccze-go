// Package plugin defines the plugin framework for CCZE log colorization.
package plugin

import (
	"io"
	"strconv"
	"strings"
	"time"

	"ccze-go/color"
)

// Type represents the type of a plugin.
type Type int

const (
	// TypeFull indicates a plugin that handles complete log lines.
	TypeFull Type = iota
	// TypePartial indicates a plugin that handles partial log lines.
	TypePartial
	// TypeAny indicates a plugin that can handle any log line.
	TypeAny
)

// Plugin is the interface that all CCZE plugins must implement.
type Plugin interface {
	// Name returns the plugin's name.
	Name() string
	// Type returns the plugin's type.
	Type() Type
	// Description returns a human-readable description of the plugin.
	Description() string
	// Handle attempts to colorize a log line. It returns whether the line
	// was handled and any remaining unprocessed text.
	Handle(line string) (handled bool, rest string)
}

// Registry holds a collection of registered plugins.
type Registry struct {
	plugins []Plugin
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin) {
	r.plugins = append(r.plugins, p)
}

// Run iterates over plugins of the matching type and returns on the first
// plugin that successfully handles the line.
func (r *Registry) Run(line string, ptype Type) (handled bool, rest string) {
	for _, p := range r.plugins {
		if p.Type() == ptype || p.Type() == TypeAny {
			h, rest := p.Handle(line)
			if h {
				return true, rest
			}
		}
	}
	return false, ""
}

// Plugins returns all registered plugins (for listing).
func (r *Registry) Plugins() []Plugin {
	return r.plugins
}

// Filter removes plugins whose names are not in the given set.
func (r *Registry) Filter(names map[string]bool) {
	var filtered []Plugin
	for _, p := range r.plugins {
		if names[p.Name()] {
			filtered = append(filtered, p)
		}
	}
	r.plugins = filtered
}

// digitAt reports whether line has a decimal digit at index i.
// Used by cheap byte-level prefilters that reject lines before running an
// expensive regexp; every prefilter condition is a necessary condition of
// the regexp it guards, so match behavior is unchanged.
func digitAt(line string, i int) bool {
	return i < len(line) && line[i] >= '0' && line[i] <= '9'
}

// HTTPAction maps an HTTP method string to its corresponding color.
func HTTPAction(method string) color.Color {
	switch strings.ToUpper(method) {
	case "GET":
		return color.HTTPGet
	case "POST":
		return color.HTTPPost
	case "HEAD":
		return color.HTTPHead
	case "PUT":
		return color.HTTPPut
	case "CONNECT":
		return color.HTTPConnect
	case "TRACE":
		return color.HTTPTrace
	default:
		return color.Unknown
	}
}

// PrintDate prints a date string, optionally converting a unix timestamp
// to a human-readable format.
func PrintDate(w io.Writer, ct *color.Table, date string, convdate bool) {
	if convdate {
		// Try to parse as unix timestamp
		if ts, err := strconv.ParseInt(strings.TrimSpace(date), 10, 64); err == nil && ts >= 0 {
			t := time.Unix(ts, 0).UTC()
			ct.WriteColored(w, color.Date, t.Format("Jan  2 15:04:05"))
			return
		}
	}
	ct.WriteColored(w, color.Date, date)
}
