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
	provider       string
	apiKey         string
	model          string
	title          string
	color          bool
	conversationID string
	// localHistory is used when the gateway has no conversation store (no Postgres).
	localHistory []Message
	useRemote    bool
}

func runChat(ctx context.Context, client *Client, provider, apiKey, model, title string, color bool) error {
	s := &chatSession{
		client:   client,
		provider: provider,
		apiKey:   apiKey,
		model:    model,
		title:    title,
		color:    color,
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
			if quit, cmdErr := s.handleCommand(ctx, reader, line); quit {
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
	var result GenerateResult
	var err error
	started := time.Now()

	done := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprint(os.Stdout, "\r\033[K")
				return
			default:
				frame := brailleSpinner[i%len(brailleSpinner)]
				fmt.Fprintf(os.Stdout, "\r  %s %s", colorize(s.color, cyan, frame), colorize(s.color, dim, "generating output…"))
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	if s.useRemote {
		result, err = s.client.PostMessage(ctx, s.conversationID, content, s.provider, s.apiKey, s.model, "act")
	} else {
		s.localHistory = append(s.localHistory, Message{Role: "user", Content: content})
		result, err = s.client.Generate(ctx, GenerateRequest{
			Messages:  s.localHistory,
			Provider:  s.provider,
			APIKey:    s.apiKey,
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
	close(done)
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

func (s *chatSession) handleCommand(ctx context.Context, reader *bufio.Reader, line string) (quit bool, err error) {
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
			fmt.Fprintf(os.Stdout, "  current model: %s\n", colorize(s.color, magenta, s.model))
			return false, nil
		}
		s.model = parts[1]
		fmt.Fprintf(os.Stdout, "  model set to %s\n", colorize(s.color, magenta, s.model))
		return false, nil
	case "/provider":
		if len(parts) < 2 {
			fmt.Fprintf(os.Stdout, "  current provider: %s\n", colorize(s.color, green, s.provider))
			return false, nil
		}
		s.provider = parts[1]
		fmt.Fprintf(os.Stdout, "  provider set to %s\n", colorize(s.color, green, s.provider))
		return false, nil
	case "/key":
		if len(parts) < 2 {
			masked := "not set"
			if s.apiKey != "" {
				if len(s.apiKey) > 8 {
					masked = s.apiKey[:4] + "..." + s.apiKey[len(s.apiKey)-4:]
				} else {
					masked = "*****"
				}
			}
			fmt.Fprintf(os.Stdout, "  current api key: %s\n", colorize(s.color, yellow, masked))
			return false, nil
		}
		s.apiKey = parts[1]
		fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  api key updated"))
		return false, nil
	case "/config", "/settings":
		s.runInteractiveConfig(reader)
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

func (s *chatSession) runInteractiveConfig(reader *bufio.Reader) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, colorize(s.color, bold+cyan, "  ┌──────────────────────────────────────────────┐"))
	fmt.Fprintln(os.Stdout, colorize(s.color, bold+cyan, "  │        PROJECT ATLAS — MODEL CONFIG          │"))
	fmt.Fprintln(os.Stdout, colorize(s.color, bold+cyan, "  └──────────────────────────────────────────────┘"))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, colorize(s.color, dim, "  Select Model Provider:"))
	fmt.Fprintln(os.Stdout, colorize(s.color, cyan, "    1) nvidia  (NVIDIA NIM OpenAI-compatible API)"))
	fmt.Fprintln(os.Stdout, colorize(s.color, cyan, "    2) openai  (OpenAI Chat Completions API)"))
	fmt.Fprintln(os.Stdout, colorize(s.color, cyan, "    3) ollama  (Local Ollama instance)"))
	fmt.Fprintln(os.Stdout, colorize(s.color, cyan, "    4) fake    (Deterministic test provider)"))
	fmt.Fprintln(os.Stdout, colorize(s.color, cyan, "    5) auto    (Dynamic router heuristic)"))
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, colorize(s.color, bold+magenta, "  Provider choice [1-5 or name] (%s) › "), s.provider)

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch choice {
	case "1", "nvidia":
		s.provider = "nvidia"
	case "2", "openai":
		s.provider = "openai"
	case "3", "ollama":
		s.provider = "ollama"
	case "4", "fake":
		s.provider = "fake"
	case "5", "auto":
		s.provider = "auto"
	case "":
		// Keep current provider
	default:
		s.provider = choice
	}
	fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  Provider set to: "+s.provider))
	fmt.Fprintln(os.Stdout)

	if s.provider == "nvidia" || s.provider == "openai" {
		masked := "not set"
		if s.apiKey != "" {
			if len(s.apiKey) > 8 {
				masked = s.apiKey[:4] + "..." + s.apiKey[len(s.apiKey)-4:]
			} else {
				masked = "*****"
			}
		}
		fmt.Fprintf(os.Stdout, colorize(s.color, bold+magenta, "  Enter API key (%s) [Press Enter to keep] › "), masked)
		keyInput, _ := reader.ReadString('\n')
		keyInput = strings.TrimSpace(keyInput)
		if keyInput != "" {
			s.apiKey = keyInput
			fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  API Key updated"))
		}
		fmt.Fprintln(os.Stdout)

		// Fetch available models from endpoint dynamically
		models, err := s.client.ListModels(context.Background(), s.provider, s.apiKey)
		if err == nil && len(models) > 0 {
			fmt.Fprintln(os.Stdout, colorize(s.color, dim, "  Available Models from "+s.provider+":"))
			limit := 10
			if len(models) < limit {
				limit = len(models)
			}
			for idx := 0; idx < limit; idx++ {
				fmt.Fprintf(os.Stdout, colorize(s.color, cyan, "    %2d) %s\n"), idx+1, models[idx].ID)
			}
			fmt.Fprintln(os.Stdout)
		}

		defaultModel := "meta/llama-3.1-70b-instruct"
		if s.provider == "openai" {
			defaultModel = "gpt-4o"
		}
		currModel := s.model
		if currModel == "" {
			currModel = defaultModel
		}
		fmt.Fprintf(os.Stdout, colorize(s.color, bold+magenta, "  Select model [1-%d or type custom name] (%s) › "), len(models), currModel)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput != "" {
			idx := 0
			if _, parseErr := fmt.Sscanf(modelInput, "%d", &idx); parseErr == nil && idx > 0 && idx <= len(models) {
				s.model = models[idx-1].ID
			} else {
				s.model = modelInput
			}
		} else if s.model == "" {
			s.model = defaultModel
		}
		fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  Model set to: "+s.model))
	} else if s.provider == "ollama" {
		currModel := s.model
		if currModel == "" {
			currModel = "llama3"
		}
		fmt.Fprintf(os.Stdout, colorize(s.color, bold+magenta, "  Enter Ollama Model (%s) › "), currModel)
		modelInput, _ := reader.ReadString('\n')
		modelInput = strings.TrimSpace(modelInput)
		if modelInput != "" {
			s.model = modelInput
		} else if s.model == "" {
			s.model = currModel
		}
		fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  Model set to: "+s.model))
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, colorize(s.color, green, "  ✓  Configuration applied for session."))
	fmt.Fprintln(os.Stdout)
}

func printHelp(w io.Writer, color bool) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, colorize(color, bold, "  commands"))
	rows := [][2]string{
		{"/config", "open interactive provider & API key settings GUI"},
		{"/provider <name>", "switch provider (nvidia, openai, ollama, fake, auto)"},
		{"/key <api_key>", "set/update API key for current provider"},
		{"/model <name>", "switch model for next turns"},
		{"/help", "show this help"},
		{"/new", "start a fresh conversation"},
		{"/history", "print messages so far"},
		{"/id", "show conversation id"},
		{"/clear", "clear the screen + redraw banner"},
		{"/quit", "exit the console"},
	}
	for _, r := range rows {
		fmt.Fprintf(w, "    %s  %s\n",
			colorize(color, cyan, fmt.Sprintf("%-18s", r[0])),
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
