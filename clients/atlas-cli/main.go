// Command atlas-cli is a colorful terminal console for Project Atlas.
//
// It talks to the inference gateway over HTTP and supports interactive chat
// plus one-shot generate / health checks.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("atlas", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	baseURL := fs.String("url", envOr("ATLAS_GATEWAY_URL", "http://localhost:8080"), "inference gateway base URL")
	orchURL := fs.String("orch-url", envOr("ATLAS_ORCHESTRATOR_URL", "http://localhost:8090"), "agent orchestrator base URL")
	model := fs.String("model", envOr("ATLAS_INFERENCE_MODEL", "fake-model"), "model name sent to the gateway")
	title := fs.String("title", "", "conversation title (chat mode)")
	noColor := fs.Bool("no-color", false, "disable ANSI colors")
	autoApprove := fs.Bool("auto-approve", false, "auto-approve waiting_approval agent runs")
	stream := fs.Bool("stream", false, "stream generate output via SSE (generate command)")
	showVersion := fs.Bool("version", false, "print version and exit")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `atlas-cli — Project Atlas terminal console

Usage:
  atlas                 start interactive chat (default)
  atlas chat            start interactive chat
  atlas generate MSG    one-shot generation (no conversation)
  atlas health          check gateway /healthz and /readyz
  atlas run "GOAL"      create+start+poll agent orchestrator run
  atlas gateway-run ID  inspect inference-gateway persisted run
  atlas conversation <id>  show a durable conversation (needs Postgres)
  atlas openapi         fetch /openapi.yaml from the gateway
  atlas version         print CLI version

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Environment:
  ATLAS_GATEWAY_URL         default for -url
  ATLAS_ORCHESTRATOR_URL    default for -orch-url
  ATLAS_INFERENCE_MODEL     default for -model

Examples:
  atlas run "calculate 2+2 and summarize"
  atlas run -auto-approve "danger echo ship it"
  atlas generate "hello from the CLI"
  atlas -stream generate "stream this reply"
  atlas health
`)
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println("atlas-cli", version)
		return 0
	}

	cmd := "chat"
	rest := fs.Args()
	if len(rest) > 0 {
		cmd = rest[0]
		rest = rest[1:]
	}

	color := !*noColor && isTTY(os.Stdout)
	client := NewClient(*baseURL)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch strings.ToLower(cmd) {
	case "version", "v":
		fmt.Println("atlas-cli", version)
		return 0
	case "health", "status":
		return cmdHealth(ctx, client, color)
	case "generate", "gen", "g":
		msg := strings.Join(rest, " ")
		if strings.TrimSpace(msg) == "" {
			fmt.Fprintln(os.Stderr, "usage: atlas generate <message>")
			return 2
		}
		printBanner(os.Stdout, *baseURL, *model, color)
		return cmdGenerate(ctx, client, *model, msg, color, *stream)
	case "run", "agent", "goal":
		goal := strings.TrimSpace(strings.Join(rest, " "))
		if goal == "" {
			fmt.Fprintln(os.Stderr, `usage: atlas run "your goal"`)
			return 2
		}
		printBanner(os.Stdout, *orchURL, *model, color)
		return cmdAgentRun(ctx, NewOrchestratorClient(*orchURL), goal, color, *autoApprove)
	case "gateway-run", "irun", "inference-run":
		if len(rest) < 1 || strings.TrimSpace(rest[0]) == "" {
			fmt.Fprintln(os.Stderr, "usage: atlas gateway-run <run_id>")
			return 2
		}
		return cmdGetRun(ctx, client, rest[0], color)
	case "conversation", "conv", "cshow":
		if len(rest) < 1 || strings.TrimSpace(rest[0]) == "" {
			fmt.Fprintln(os.Stderr, "usage: atlas conversation <conversation_id>")
			return 2
		}
		return cmdGetConversation(ctx, client, rest[0], color)
	case "openapi", "spec":
		return cmdOpenAPI(ctx, client, color)
	case "chat", "c", "console", "repl":
		printBanner(os.Stdout, *baseURL, *model, color)
		if err := runChat(ctx, client, *model, *title, color); err != nil {
			fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
			return 1
		}
		return 0
	case "help", "-h", "--help":
		fs.Usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		fs.Usage()
		return 2
	}
}

func cmdHealth(ctx context.Context, client *Client, color bool) int {
	fmt.Fprintln(os.Stdout, colorize(color, bold+cyan, "  ATLAS  health check"))
	fmt.Fprintln(os.Stdout, colorize(color, dim, "  gateway  "+client.BaseURL))
	fmt.Fprintln(os.Stdout)

	body, err := client.Health(ctx)
	if err != nil {
		fmt.Fprintln(os.Stdout, colorize(color, red, "  ✗  /healthz  "+err.Error()))
		return 1
	}
	fmt.Fprintln(os.Stdout, colorize(color, green, "  ✓  /healthz  "+body))

	body, err = client.Ready(ctx)
	if err != nil {
		fmt.Fprintln(os.Stdout, colorize(color, yellow, "  ⚠  /readyz   "+err.Error()))
		return 1
	}
	fmt.Fprintln(os.Stdout, colorize(color, green, "  ✓  /readyz   "+body))
	return 0
}

func cmdGenerate(ctx context.Context, client *Client, model, msg string, color bool, stream bool) int {
	req := GenerateRequest{
		Messages:  []Message{{Role: "user", Content: msg}},
		Model:     model,
		TaskClass: "act",
	}

	var result GenerateResult
	var err error
	if stream {
		fmt.Fprintln(os.Stdout, colorize(color, dim, "  … streaming"))
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, colorize(color, bold+magenta, "atlas ›"))
		fmt.Fprint(os.Stdout, "  ")
		result, err = client.GenerateStream(ctx, req, func(d StreamDelta) error {
			fmt.Fprint(os.Stdout, d.Delta)
			return nil
		})
		fmt.Fprintln(os.Stdout)
	} else {
		fmt.Fprintln(os.Stdout, colorize(color, dim, "  … thinking"))
		result, err = client.Generate(ctx, req)
		if err == nil {
			fmt.Fprintln(os.Stdout)
			fmt.Fprintln(os.Stdout, colorize(color, bold+magenta, "atlas ›"))
			fmt.Fprintln(os.Stdout, "  "+result.Message.Content)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
		return 1
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "  %s %s  ·  %s %s  ·  %s %s\n",
		colorize(color, dim, "provider"),
		colorize(color, green, nonEmpty(result.Provider, "?")),
		colorize(color, dim, "model"),
		colorize(color, cyan, nonEmpty(result.Model, model)),
		colorize(color, dim, "run"),
		colorize(color, dim, shortID(result.RunID)),
	)
	if result.RunID != "" {
		fmt.Fprintf(os.Stdout, "  %s %s\n", colorize(color, dim, "full run id"), result.RunID)
	}
	return 0
}

func cmdGetRun(ctx context.Context, client *Client, runID string, color bool) int {
	view, err := client.GetRun(ctx, runID)
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
		return 1
	}
	fmt.Fprintln(os.Stdout, colorize(color, bold+cyan, "  ATLAS  inference run"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "run_id"), view.RunID)
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "status"), colorize(color, statusColor(color, view.Status), view.Status))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "provider"), nonEmpty(view.Provider, "—"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "model"), nonEmpty(view.ResolvedModel, "—"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "request"), nonEmpty(view.RequestID, "—"))
	if view.Result != nil {
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, colorize(color, bold+magenta, "  result ›"))
		fmt.Fprintln(os.Stdout, "  "+view.Result.Message.Content)
	}
	if view.ErrorMessage != "" {
		fmt.Fprintln(os.Stdout, colorize(color, red, "  error: "+view.ErrorClass+" — "+view.ErrorMessage))
	}
	return 0
}

func cmdGetConversation(ctx context.Context, client *Client, id string, color bool) int {
	conv, err := client.GetConversation(ctx, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
		return 1
	}
	fmt.Fprintln(os.Stdout, colorize(color, bold+cyan, "  ATLAS  conversation"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "id"), conv.ConversationID)
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "title"), nonEmpty(conv.Title, "—"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "status"), conv.Status)
	fmt.Fprintln(os.Stdout)
	if len(conv.Messages) == 0 {
		fmt.Fprintln(os.Stdout, colorize(color, dim, "  (no messages yet)"))
		return 0
	}
	for _, m := range conv.Messages {
		style := dim
		if m.Role == "user" {
			style = cyan
		} else if m.Role == "assistant" {
			style = magenta
		}
		fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, bold+style, m.Role), m.Content)
	}
	return 0
}

func cmdOpenAPI(ctx context.Context, client *Client, color bool) int {
	doc, err := client.OpenAPI(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
		return 1
	}
	// Print raw YAML for piping into files or tools.
	fmt.Fprint(os.Stdout, doc)
	return 0
}

func statusColor(color bool, status string) string {
	if !color {
		return ""
	}
	switch status {
	case "succeeded":
		return green
	case "failed", "cancelled":
		return red
	case "running", "waiting_approval":
		return yellow
	default:
		return dim
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
