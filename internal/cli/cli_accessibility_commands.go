package cli

import (
	"context"
	"fmt"
	"io"

	"gdctl/internal/bridge"
)

func runAccessibility(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("accessibility requires a subcommand: tts-speak, tts-configure, tts-stop")
	}
	switch args[0] {
	case "tts-speak":
		return runAccessibilityTTSSpeak(ctx, client, args[1:], stdout)
	case "tts-configure":
		return runAccessibilityTTSConfigure(ctx, client, args[1:], stdout)
	case "tts-stop":
		return runAccessibilityTTSStop(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown accessibility subcommand: %s", args[0])
	}
}

func runAccessibilityTTSSpeak(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("accessibility tts-speak")
	text := fs.String("text", "", "text to speak")
	interrupt := fs.Bool("interrupt", false, "interrupt current speech")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *text == "" {
		return fmt.Errorf("accessibility tts-speak requires --text")
	}
	result, err := client.AccessibilityTTSSpeak(ctx, requestID(), *text, *interrupt)
	if err != nil {
		return err
	}
	if voice, _ := result["voice"].(string); voice != "" {
		fmt.Fprintf(stdout, "TTS speak: %q (voice: %s)%s\n", *text, voice, serverNote(result))
	} else {
		fmt.Fprintf(stdout, "TTS speak: %q%s\n", *text, serverNote(result))
	}
	return nil
}

func runAccessibilityTTSConfigure(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("accessibility tts-configure")
	pitch := fs.Float64("pitch", 0, "pitch (0 = unchanged)")
	rate := fs.Float64("rate", 0, "rate (0 = unchanged)")
	voice := fs.String("voice", "", "voice ID (empty = unchanged)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.AccessibilityTTSConfigure(ctx, requestID(), *pitch, *rate, *voice)
	if err != nil {
		return err
	}
	// The bridge echoes back the effective configuration, filling in any values
	// left unchanged (0 / empty), so report what actually took effect.
	effPitch, _ := result["pitch"].(float64)
	effRate, _ := result["rate"].(float64)
	effVolume, _ := result["volume"].(float64)
	effVoice, _ := result["voice"].(string)
	fmt.Fprintf(stdout, "TTS configured: pitch=%.2f rate=%.2f volume=%d voice=%s\n",
		effPitch, effRate, int(effVolume), valueOrDash(effVoice))
	return nil
}

func runAccessibilityTTSStop(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("accessibility tts-stop")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := client.AccessibilityTTSStop(ctx, requestID())
	if err != nil {
		return err
	}
	stopped, _ := result["stopped"].(bool)
	if stopped {
		fmt.Fprintln(stdout, "TTS stopped")
	} else {
		note, _ := result["note"].(string)
		fmt.Fprintf(stdout, "TTS stop: %s\n", note)
	}
	return nil
}
