package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type chatSession struct {
	client         *Client
	model          string
	title          string
	color          bool
	conversationID string
	// localHistory is used when the gateway has no conversation store (no Postgres).
	localHistory []Message
	useRemote    bool
}

func runChat(ctx context.Context, client *Client, model, title string, color bool) error {
	s := &chatSession{
		client: client,
		model:  model,
		title:  title,
		color:  color,
	}

	fmt.Fprintln(os.Stdout, colorize(color, dim, "  Checking gateway…"))
	if _, err := client.Health(ctx); err != nil {
		return fmt.Errorf("%w\n\n  Start the gateway first, e.g.:\n    cd services/inference-gateway && go run ./cmd/inference-gateway\n  or:\n    docker compose -f deploy/docker-compose.yml up", err)
	}
	if _, err := client.Ready(ctx); err != nil {
		fmt.Fprintln(os.Stdout, colorize(color, yellow, "  ⚠  gateway is up but not ready: "+err.Error()))
	} else {
		fmt.Fprintln(os.Stdout, colorize(color, green, "  ✓  gateway healthy"))
	}

	if err := s.startConversation(ctx); err != nil {
		// Fall back to ephemeral local chat via /v1/generate.
		fmt.Fprintln(os.Stdout, colorize(color, yellow, "  ⚠  durable conversations unavailable — using ephemeral mode"))
		fmt.Fprintln(os.Stdout, colorize(color, dim, "     (start Postgres + set ATLAS_DATABASE_URL for saved chats)"))
		s.useRemote = false
	} else {
		s.useRemote = true
		fmt.Fprintf(os.Stdout, "  %s  %s\n",
			colorize(color, dim, "conversation"),
			colorize(color, cyan, s.conversationID),
		)
	}

	printHelp(os.Stdout, color)
	fmt.Fprintln(os.Stdout)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, colorize(color, bold+cyan, "you › "))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(os.Stdout)
				fmt.Fprintln(os.Stdout, colorize(color, dim, "  until next orbit. 🚀"))
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if quit, cmdErr := s.handleCommand(ctx, line); quit {
				fmt.Fprintln(os.Stdout, colorize(color, dim, "  until next orbit. 🚀"))
				return nil
			} else if cmdErr != nil {
				fmt.Fprintln(os.Stdout, colorize(color, red, "  ✗  "+cmdErr.Error()))
			}
			continue
		}

		if err := s.send(ctx, line); err != nil {
			fmt.Fprintln(os.Stdout, colorize(color, red, "  ✗  "+err.Error()))
		}
	}
}

func (s *chatSession) startConversation(ctx context.Context) error {
	title := s.title
	if title == "" {
		title = "atlas-cli " + time.Now().Format("2006-01-02 15:04")
	}
	conv, err := s.client.CreateConversation(ctx, title)
	if err != nil {
		return err
	}
	s.conversationID = conv.ConversationID
	s.localHistory = nil
	return nil
}

func (s *chatSession) send(ctx context.Context, content string) error {
	fmt.Fprintln(os.Stdout, colorize(s.color, dim, "  … thinking"))

	var result GenerateResult
	var err error
	started := time.Now()

	if s.useRemote {
		result, err = s.client.PostMessage(ctx, s.conversationID, content, s.model, "act")
	} else {
		s.localHistory = append(s.localHistory, Message{Role: "user", Content: content})
		result, err = s.client.Generate(ctx, GenerateRequest{
			Messages:  s.localHistory,
			Model:     s.model,
			TaskClass: "act",
		})
		if err == nil {
			s.localHistory = append(s.localHistory, Message{
				Role:    result.Message.Role,
				Content: result.Message.Content,
			})
		} else {
			// drop the failed user turn from local history
			s.localHistory = s.localHistory[:len(s.localHistory)-1]
		}
	}
	if err != nil {
		return err
	}

	elapsed := time.Since(started).Round(time.Millisecond)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%s\n", colorize(s.color, bold+magenta, "atlas ›"))
	fmt.Fprintln(os.Stdout, "  "+strings.ReplaceAll(result.Message.Content, "\n", "\n  "))
	fmt.Fprintln(os.Stdout)

	meta := fmt.Sprintf("  %s %s  ·  %s %s  ·  %s %s  ·  %s %s  ·  %s %d→%d",
		colorize(s.color, dim, "provider"),
		colorize(s.color, green, nonEmpty(result.Provider, "?")),
		colorize(s.color, dim, "model"),
		colorize(s.color, cyan, nonEmpty(result.Model, s.model)),
		colorize(s.color, dim, "run"),
		colorize(s.color, dim, shortID(result.RunID)),
		colorize(s.color, dim, "lat"),
		colorize(s.color, dim, elapsed.String()),
		colorize(s.color, dim, "tok"),
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
	)
	fmt.Fprintln(os.Stdout, meta)
	fmt.Fprintln(os.Stdout)
	return nil
}

func (s *chatSession) handleCommand(ctx context.Context, line string) (quit bool, err error) {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/help", "/h", "/?":
		printHelp(os.Stdout, s.color)
		return false, nil
	case "/quit", "/exit", "/q":
		return true, nil
	case "/clear", "/cls":
		fmt.Fprint(os.Stdout, "\033[H\033[2J")
		printBanner(os.Stdout, s.client.BaseURL, s.model, s.color)
		return false, nil
	case "/new":
		if s.useRemote {
			if err := s.startConversation(ctx); err != nil {
				return false, err
			}
			fmt.Fprintf(os.Stdout, "  %s  %s\n",
				colorize(s.color, green, "✓ new conversation"),
				colorize(s.color, cyan, s.conversationID),
			)
		} else {
			s.localHistory = nil
			fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  local history cleared"))
		}
		return false, nil
	case "/history":
		return false, s.printHistory(ctx)
	case "/id":
		if s.useRemote {
			fmt.Fprintf(os.Stdout, "  conversation: %s\n", s.conversationID)
		} else {
			fmt.Fprintln(os.Stdout, "  mode: ephemeral (no conversation id)")
		}
		return false, nil
	case "/model":
		if len(parts) < 2 {
			fmt.Fprintf(os.Stdout, "  current model: %s\n", s.model)
			return false, nil
		}
		s.model = parts[1]
		fmt.Fprintf(os.Stdout, "  model set to %s\n", colorize(s.color, magenta, s.model))
		return false, nil
	default:
		return false, fmt.Errorf("unknown command %q — try /help", cmd)
	}
}

func (s *chatSession) printHistory(ctx context.Context) error {
	if s.useRemote {
		conv, err := s.client.GetConversation(ctx, s.conversationID)
		if err != nil {
			return err
		}
		if len(conv.Messages) == 0 {
			fmt.Fprintln(os.Stdout, colorize(s.color, dim, "  (empty conversation)"))
			return nil
		}
		for _, m := range conv.Messages {
			role := m.Role
			style := dim
			if role == "user" {
				style = cyan
			} else if role == "assistant" {
				style = magenta
			}
			fmt.Fprintf(os.Stdout, "  %s  %s\n",
				colorize(s.color, bold+style, role),
				m.Content,
			)
		}
		return nil
	}
	if len(s.localHistory) == 0 {
		fmt.Fprintln(os.Stdout, colorize(s.color, dim, "  (empty history)"))
		return nil
	}
	for _, m := range s.localHistory {
		style := dim
		if m.Role == "user" {
			style = cyan
		} else if m.Role == "assistant" {
			style = magenta
		}
		fmt.Fprintf(os.Stdout, "  %s  %s\n",
			colorize(s.color, bold+style, m.Role),
			m.Content,
		)
	}
	return nil
}

func printHelp(w io.Writer, color bool) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, colorize(color, bold, "  commands"))
	rows := [][2]string{
		{"/help", "show this help"},
		{"/new", "start a fresh conversation"},
		{"/history", "print messages so far"},
		{"/id", "show conversation id"},
		{"/model <name>", "switch model for next turns"},
		{"/clear", "clear the screen + redraw banner"},
		{"/quit", "exit the console"},
	}
	for _, r := range rows {
		fmt.Fprintf(w, "    %s  %s\n",
			colorize(color, cyan, fmt.Sprintf("%-14s", r[0])),
			colorize(color, dim, r[1]),
		)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, colorize(color, dim, "  type a message and press Enter to chat."))
}

func shortID(id string) string {
	if id == "" {
		return "—"
	}
	if len(id) <= 12 {
		return id
	}
	return id[:8] + "…"
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
