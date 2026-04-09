package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"bootcamp/web/internal/db"
	"bootcamp/web/internal/upload"
)

type Reporter struct {
	sdkURL string
	db     *db.DB
	upload *upload.Client
	log    *slog.Logger
	client *http.Client
}

func NewReporter(sdkURL string, database *db.DB, uploadClient *upload.Client, log *slog.Logger) *Reporter {
	return &Reporter{
		sdkURL: sdkURL,
		db:     database,
		upload: uploadClient,
		log:    log,
		client: &http.Client{},
	}
}

func (r *Reporter) Report(ctx context.Context) error {
	userCount, err := r.db.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	fileCount, err := r.upload.CountAllFiles(ctx)
	if err != nil {
		return fmt.Errorf("count files: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"user_count": userCount,
			"file_count": fileCount,
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		r.sdkURL+"/api/v1/app/custom-metrics", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("sdk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sdk response: status %d", resp.StatusCode)
	}
	return nil
}
