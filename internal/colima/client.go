// Package colima locates the Colima executable, runs its commands, parses the
// resulting profile status, and streams Lima lifecycle events.
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
	limaPath := locateLimactl(path)
	return &Client{
		path:          path,
		profile:       profile,
		limaPath:      limaPath,
		limaHome:      resolveLimaHome(),
		limaInstance:  limaInstanceName(profile),
		runner:        ExecRunner{lookupPaths: executableDirectories(limaPath)},
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
		return Profile{}, fmt.Errorf("Colima status could not be queried: %w", err)
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
	return client.run(ctx, "Colima could not be started", client.profileArgs("start")...)
}

func (client *Client) Stop(ctx context.Context, force bool) error {
	args := client.profileArgs("stop")
	if force {
		args = append(args, "--force")
	}
	return client.run(ctx, "Colima could not be stopped", args...)
}

// profileArgs selects the profile with the documented global -p flag rather
// than the positional form. Both work today, but -p is the one Colima's help
// describes for every subcommand, so it is the safer contract to depend on.
func (client *Client) profileArgs(action string) []string {
	if client.profile == defaultProfile {
		return []string{action}
	}
	return []string{action, "-p", client.profile}
}

func (client *Client) run(ctx context.Context, message string, args ...string) error {
	actionContext, cancel := context.WithTimeout(ctx, client.actionTimeout)
	defer cancel()
	if _, err := client.runner.Run(actionContext, client.path, args...); err != nil {
		return fmt.Errorf("%s: %w", message, err)
	}
	return nil
}
