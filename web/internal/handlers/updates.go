package handlers

import (
	"encoding/json"
	"net/http"
)

func (a *App) handleUpdates(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Available bool   `json:"available"`
		Version   string `json:"version,omitempty"`
	}

	res := response{}
	if a.Updates != nil {
		updates, err := a.Updates.CheckUpdates(r.Context())
		if err != nil {
			a.Log.Warn("check updates", "err", err)
		} else if len(updates) > 0 {
			res.Available = true
			res.Version = updates[0].VersionLabel
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
