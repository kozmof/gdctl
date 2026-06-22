package cli

import (
	"context"
	"fmt"
	"io"

	"gdctl/internal/bridge"
)

func runLocalization(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("localization requires a subcommand: locale-set, string-add")
	}
	switch args[0] {
	case "locale-set":
		return runLocalizationLocaleSet(ctx, client, args[1:], stdout)
	case "string-add":
		return runLocalizationStringAdd(ctx, client, args[1:], stdout)
	default:
		return fmt.Errorf("unknown localization subcommand: %s", args[0])
	}
}

func runLocalizationLocaleSet(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("localization locale-set")
	locale := fs.String("locale", "", "locale code (e.g. en, ja, ko, fr)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *locale == "" {
		return fmt.Errorf("localization locale-set requires --locale")
	}
	result, err := client.I18nLocaleSet(ctx, requestID(), *locale)
	if err != nil {
		return err
	}
	current, _ := result["locale"].(string)
	fmt.Fprintf(stdout, "Locale set: %s\n", current)
	return nil
}

func runLocalizationStringAdd(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := newFlagSet("localization string-add")
	key := fs.String("key", "", "translation key")
	locale := fs.String("locale", "", "locale code")
	text := fs.String("text", "", "translated text")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || *locale == "" || *text == "" {
		return fmt.Errorf("localization string-add requires --key, --locale, and --text")
	}
	result, err := client.I18nStringAdd(ctx, requestID(), *key, *locale, *text)
	if err != nil {
		return err
	}
	_ = result
	fmt.Fprintf(stdout, "Translation added: %s[%s] = %q\n", *key, *locale, *text)
	return nil
}
