package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

// Lightmap bake command (System Layer)

func runLightmap(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("lightmap requires a subcommand: bake")
	}
	switch args[0] {
	case "bake":
		return runLightmapBake(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown lightmap subcommand: %s", args[0])
	}
}

func runLightmapBake(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("lightmap bake")
	path := fs.String("path", "", "LightmapGI node path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return fmt.Errorf("lightmap bake requires --path")
	}
	result, err := client.LightmapBake(ctx, requestID(), *path)
	if err != nil {
		return err
	}
	status, _ := result["status"].(string)
	note, _ := result["note"].(string)
	fmt.Fprintf(stdout, "LightmapGI bake: %s (%s)\n", *path, status)
	if note != "" {
		fmt.Fprintf(stdout, "  Note: %s\n", note)
	}
	return nil
}

// runRunProfile is called by the run subcommand dispatcher (cli_run.go).

func runRunProfile(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("run profile")
	metric := fs.String("metric", "fps", "comma-separated metrics: fps,draw_calls,physics_time,memory_usage")
	duration := fs.Duration("duration", 5*time.Second, "sampling duration (e.g. 5s, 30s)")
	timeout := fs.Duration("timeout", 120*time.Second, "maximum time to wait for profile result")
	if err := fs.Parse(args); err != nil {
		return err
	}
	metrics := strings.Split(*metric, ",")
	for i, m := range metrics {
		metrics[i] = strings.TrimSpace(m)
	}
	durationMS := float64(duration.Milliseconds())
	result, err := client.RunProfile(ctx, requestID(), metrics, durationMS)
	if err != nil {
		return err
	}
	if result.JobID == "" {
		return fmt.Errorf("run profile did not return a job id")
	}
	job, err := waitForJob(ctx, client, result.JobID, *timeout, "run profile")
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Profile result (%.0fms, %d samples):\n", durationMS, int(floatFromResult(job.Result["sample_count"])))
	for _, m := range metrics {
		avgKey := m + "_avg"
		minKey := m + "_min"
		avg := floatFromResult(job.Result[avgKey])
		min := floatFromResult(job.Result[minKey])
		if avg > 0 || min > 0 {
			fmt.Fprintf(stdout, "  %s: avg=%.2f min=%.2f\n", m, avg, min)
		}
	}
	return nil
}

func floatFromResult(v any) float64 {
	switch f := v.(type) {
	case float64:
		return f
	case float32:
		return float64(f)
	}
	return 0
}
