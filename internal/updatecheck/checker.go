package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const CurrentVersion = "0.1.0"

type Update struct {
	Current   string
	Latest    string
	URL       string
	Available bool
}

type Checker struct {
	client   *http.Client
	endpoint string
	current  string
	mu       sync.RWMutex
	update   Update
}

func New(endpoint, current string) *Checker {
	return &Checker{client: &http.Client{Timeout: 10 * time.Second}, endpoint: endpoint, current: strings.TrimPrefix(strings.TrimSpace(current), "v")}
}

func (c *Checker) Run(ctx context.Context) {
	c.checkAndLog(ctx)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.checkAndLog(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Checker) Snapshot() Update {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.update
}

func (c *Checker) Check(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "1984-update-checker")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		c.mu.Lock()
		c.update = Update{Current: c.current}
		c.mu.Unlock()
		return nil
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub respondeu HTTP %d", response.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err = json.NewDecoder(response.Body).Decode(&release); err != nil {
		return err
	}
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if latest == "" || release.HTMLURL == "" {
		return errors.New("versão inválida na resposta do GitHub")
	}
	available, err := newer(latest, c.current)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.update = Update{Current: c.current, Latest: latest, URL: release.HTMLURL, Available: available}
	c.mu.Unlock()
	return nil
}

func (c *Checker) checkAndLog(ctx context.Context) {
	if err := c.Check(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("update check failed", "error", err)
	}
}

func newer(latest, current string) (bool, error) {
	left, err := versionParts(latest)
	if err != nil {
		return false, err
	}
	right, err := versionParts(current)
	if err != nil {
		return false, err
	}
	for index := range left {
		if left[index] != right[index] {
			return left[index] > right[index], nil
		}
	}
	return false, nil
}

func versionParts(value string) ([3]int, error) {
	var result [3]int
	value = strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, fmt.Errorf("versão inválida: %q", value)
	}
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return result, fmt.Errorf("versão inválida: %q", value)
		}
		result[index] = number
	}
	return result, nil
}
