package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/livemsg"
)

type inboxCheckOutput struct {
	Team     string                   `json:"team"`
	Agent    string                   `json:"agent"`
	Unread   int                      `json:"unread"`
	Messages []inboxCheckMessageEntry `json:"messages"`
}

type inboxCheckMessageEntry struct {
	ID        string `json:"id"`
	Team      string `json:"team"`
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type inboxCheckOpts struct {
	Team  string
	Agent string
	DB    string
}

func runInboxCheckCommand(args []string, stdout, stderr io.Writer) int {
	opts, err := parseInboxCheckArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "harness inbox check: %v\n", err)
		return 0
	}

	out, err := executeInboxCheck(opts)
	if err != nil {
		fmt.Fprintf(stderr, "harness inbox check: %v\n", err)
		return 0
	}

	data, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(stderr, "harness inbox check: marshal: %v\n", err)
		return 0
	}
	fmt.Fprintf(stdout, "%s\n", data)
	return 0
}

func parseInboxCheckArgs(args []string) (inboxCheckOpts, error) {
	var opts inboxCheckOpts
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--team":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--team requires a value")
			}
			i++
			opts.Team = args[i]
		case "--agent":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--agent requires a value")
			}
			i++
			opts.Agent = args[i]
		case "--db":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--db requires a value")
			}
			i++
			opts.DB = args[i]
		default:
			return opts, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if opts.Team == "" {
		return opts, fmt.Errorf("--team is required")
	}
	if opts.Agent == "" {
		return opts, fmt.Errorf("--agent is required")
	}
	if opts.DB == "" {
		return opts, fmt.Errorf("--db is required")
	}
	return opts, nil
}

func executeInboxCheck(opts inboxCheckOpts) (inboxCheckOutput, error) {
	empty := inboxCheckOutput{
		Team:     opts.Team,
		Agent:    opts.Agent,
		Unread:   0,
		Messages: []inboxCheckMessageEntry{},
	}

	if _, err := os.Stat(opts.DB); os.IsNotExist(err) {
		return empty, nil
	}

	store, err := livemsg.Open(opts.DB)
	if err != nil {
		return empty, nil
	}
	defer store.Close()

	ctx := context.Background()
	messages, err := store.Inbox(ctx, opts.Team, opts.Agent)
	if err != nil {
		return empty, nil
	}

	entries := make([]inboxCheckMessageEntry, 0, len(messages))
	for _, msg := range messages {
		entries = append(entries, inboxCheckMessageEntry{
			ID:        msg.ID,
			Team:      msg.Team,
			FromAgent: msg.FromAgent,
			ToAgent:   msg.ToAgent,
			Subject:   msg.Subject,
			Body:      msg.Body,
			CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
		if err := store.MarkRead(ctx, opts.Team, msg.ID, opts.Agent); err != nil {
			return empty, nil
		}
	}

	return inboxCheckOutput{
		Team:     opts.Team,
		Agent:    opts.Agent,
		Unread:   len(entries),
		Messages: entries,
	}, nil
}
