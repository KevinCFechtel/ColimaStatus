package colima

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultProfile       = "default"
	defaultStatusTimeout = 10 * time.Second
	defaultActionTimeout = 10 * time.Minute
)

type Client struct {
	path          string
	profile       string
	limaPath      string
	limaHome      string
	limaInstance  string
	runner        Runner
	now           func() time.Time
	statusTimeout time.Duration
	actionTimeout time.Duration
}

func NewClient(path, profile string) *Client {
	if profile == "" {
		profile = defaultProfile
	}
	return &Client{
		path:          path,
		profile:       profile,
		limaPath:      locateLimactl(path),
		limaHome:      resolveLimaHome(),
		limaInstance:  limaInstanceName(profile),
		runner:        ExecRunner{},
		now:           time.Now,
		statusTimeout: defaultStatusTimeout,
		actionTimeout: defaultActionTimeout,
	}
}

func (client *Client) Status(ctx context.Context) (Profile, error) {
	statusContext, cancel := context.WithTimeout(ctx, client.statusTimeout)
	defer cancel()

	output, err := client.runner.Run(statusContext, client.path, "list", "--json")
	if err != nil {
		return Profile{}, fmt.Errorf("Colima-Status konnte nicht abgefragt werden: %w", err)
	}
	profiles, err := ParseProfiles(strings.NewReader(output.Stdout))
	if err != nil {
		return Profile{}, err
	}
	for _, profile := range profiles {
		if profile.Name == client.profile {
			profile.CheckedAt = client.now()
			return profile, nil
		}
	}
	return Profile{
		Name:      client.profile,
		State:     StateMissing,
		CheckedAt: client.now(),
	}, nil
}

func (client *Client) Start(ctx context.Context) error {
	return client.runAction(ctx, "start")
}

func (client *Client) Stop(ctx context.Context, force bool) error {
	args := []string{"stop"}
	if client.profile != defaultProfile {
		args = append(args, client.profile)
	}
	if force {
		args = append(args, "--force")
	}
	return client.run(ctx, "Colima konnte nicht gestoppt werden", args...)
}

func (client *Client) runAction(ctx context.Context, action string) error {
	args := []string{action}
	if client.profile != defaultProfile {
		args = append(args, client.profile)
	}
	return client.run(ctx, "Colima konnte nicht gestartet werden", args...)
}

func (client *Client) run(ctx context.Context, message string, args ...string) error {
	actionContext, cancel := context.WithTimeout(ctx, client.actionTimeout)
	defer cancel()
	if _, err := client.runner.Run(actionContext, client.path, args...); err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
