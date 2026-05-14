package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gdctl/internal/bridge"
)

type edgeProfile struct {
	ID    int     `json:"id"`
	Mode  string  `json:"mode"`
	Mix   float64 `json:"mix"`
	Blur  float64 `json:"blur"`
	Width float64 `json:"width"`
}

func runLUTWrite(ctx context.Context, client *bridge.Client, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("file lut-write", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("path", "", "resource PNG path")
	profilesPath := fs.String("profiles", "", "local edge profile JSON path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *profilesPath == "" {
		return fmt.Errorf("file lut-write requires --path and --profiles")
	}
	data, err := os.ReadFile(*profilesPath)
	if err != nil {
		return err
	}
	var profiles []edgeProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("parse edge profiles: %w", err)
	}
	pngData, err := buildEdgeLUTPNG(profiles)
	if err != nil {
		return err
	}
	result, err := client.WriteFileBytes(ctx, requestID(), *path, base64.StdEncoding.EncodeToString(pngData))
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "LUT written: %s (%d bytes)\n", result.Path, result.Bytes)
	fmt.Fprintf(stdout, "Profiles: %d\n", len(profiles))
	return nil
}

func buildEdgeLUTPNG(profiles []edgeProfile) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 256, 1))
	for _, profile := range profiles {
		if profile.ID < 0 || profile.ID > 255 {
			return nil, fmt.Errorf("edge profile id must be between 0 and 255: %d", profile.ID)
		}
		img.SetNRGBA(profile.ID, 0, color.NRGBA{
			R: byteFromUnit(profile.Mix),
			G: byteFromUnit(profile.Blur),
			B: byteFromUnit(profile.Width),
			A: byteFromUnit(modeValue(profile.Mode)),
		})
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func byteFromUnit(value float64) uint8 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return uint8(value*255 + 0.5)
}

func modeValue(mode string) float64 {
	switch strings.ToLower(mode) {
	case "", "off":
		return 0.0
	case "blur":
		return 0.33
	case "bleed":
		return 0.66
	case "preserve", "ink-preserve", "crisp":
		return 1.0
	default:
		return 0.0
	}
}

func writeScreenshotJob(outPath string, job bridge.Job) error {
	content, _ := job.Result["content_base64"].(string)
	if content == "" {
		return fmt.Errorf("%s job did not return PNG data", strings.ReplaceAll(job.Kind, ".", " "))
	}
	data, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return fmt.Errorf("decode screenshot PNG: %w", err)
	}
	if dir := filepath.Dir(outPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	return nil
}

func defaultScreenshotPath() string {
	return filepath.Join("screenshots", time.Now().UTC().Format("20060102-150405")+".png")
}

func isSuspectedEditorCapture(pngData []byte) bool {
	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return false
	}
	b := img.Bounds()
	w, h := b.Max.X-b.Min.X, b.Max.Y-b.Min.Y
	if w < 10 || h < 10 {
		return false
	}
	const grid = 10
	var samples []color.RGBA
	for gy := 0; gy < grid; gy++ {
		for gx := 0; gx < grid; gx++ {
			x := b.Min.X + (gx*w)/grid + w/(grid*2)
			y := b.Min.Y + (gy*h)/grid + h/(grid*2)
			c := img.At(x, y)
			r, g, bv, a := c.RGBA()
			samples = append(samples, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bv >> 8), A: uint8(a >> 8)})
		}
	}
	if len(samples) == 0 {
		return false
	}
	dom := samples[0]
	match := 0
	const delta = 10
	for _, s := range samples {
		dr := int(s.R) - int(dom.R)
		dg := int(s.G) - int(dom.G)
		db := int(s.B) - int(dom.B)
		if dr < 0 {
			dr = -dr
		}
		if dg < 0 {
			dg = -dg
		}
		if db < 0 {
			db = -db
		}
		if dr <= delta && dg <= delta && db <= delta {
			match++
		}
	}
	return match*10 >= len(samples)*9
}
