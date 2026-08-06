package main

import (
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/services"
)

// lastClustersDefaultVersion is the last released version that opened in the
// clusters view when default_view was unset.
const lastClustersDefaultVersion = "2.17.2"

// shouldPinClustersView reports whether this launch should write
// default_view = "clusters" into the user's config. Versions up to 2.17.2
// opened in the clusters view; newer versions default to apps. Users who
// already launched an older version get their old default pinned so the
// upgrade doesn't change their startup view.
func shouldPinClustersView(cfg *config.ArgonautConfig, configExisted bool) bool {
	if !configExisted || cfg.DefaultView != "" {
		return false
	}
	// An empty last_seen_version means the config predates version tracking —
	// definitely a clusters-default era user.
	return cfg.LastSeenVersion == "" ||
		!services.IsVersionNewer(cfg.LastSeenVersion, lastClustersDefaultVersion)
}
