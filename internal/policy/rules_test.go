package policy

import (
	"testing"
)

func TestCheckAssetFileSize(t *testing.T) {
	policy := &AssetPolicy{MaxFileSizeMB: 1}

	t.Run("nil policy returns no violations", func(t *testing.T) {
		v := CheckAssetFileSize(nil, []FileInfo{{Path: "big.png", SizeBytes: 100 * 1024 * 1024}})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("zero limit returns no violations", func(t *testing.T) {
		v := CheckAssetFileSize(&AssetPolicy{MaxFileSizeMB: 0}, []FileInfo{{Path: "big.png", SizeBytes: 100}})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("file at limit is not a violation", func(t *testing.T) {
		limit := FileInfo{Path: "ok.png", SizeBytes: 1024 * 1024}
		v := CheckAssetFileSize(policy, []FileInfo{limit})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("file over limit is a violation", func(t *testing.T) {
		over := FileInfo{Path: "over.png", SizeBytes: 1024*1024 + 1}
		v := CheckAssetFileSize(policy, []FileInfo{over})
		if len(v) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(v))
		}
		if v[0].Rule != "assets.max_file_size_mb" {
			t.Errorf("unexpected rule %q", v[0].Rule)
		}
		if v[0].Path != "over.png" {
			t.Errorf("unexpected path %q", v[0].Path)
		}
	})

	t.Run("only over-limit files are reported", func(t *testing.T) {
		files := []FileInfo{
			{Path: "ok.png", SizeBytes: 512 * 1024},
			{Path: "big.png", SizeBytes: 2 * 1024 * 1024},
		}
		v := CheckAssetFileSize(policy, files)
		if len(v) != 1 || v[0].Path != "big.png" {
			t.Fatalf("expected 1 violation for big.png, got %v", v)
		}
	})
}

func TestCheckTextureFormat(t *testing.T) {
	policy := &TexturePolicy{AllowedFormats: []string{"png", "webp"}}

	t.Run("nil policy returns no violations", func(t *testing.T) {
		v := CheckTextureFormat(nil, []string{"tex.bmp"})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("empty allowed list returns no violations", func(t *testing.T) {
		v := CheckTextureFormat(&TexturePolicy{}, []string{"tex.bmp"})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("allowed format is not a violation", func(t *testing.T) {
		v := CheckTextureFormat(policy, []string{"sprite.png", "icon.webp"})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d: %v", len(v), v)
		}
	})

	t.Run("disallowed format is a violation", func(t *testing.T) {
		v := CheckTextureFormat(policy, []string{"sprite.bmp"})
		if len(v) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(v))
		}
		if v[0].Rule != "textures.allowed_formats" {
			t.Errorf("unexpected rule %q", v[0].Rule)
		}
	})

	t.Run("case-insensitive extension matching", func(t *testing.T) {
		v := CheckTextureFormat(policy, []string{"sprite.PNG"})
		if len(v) != 0 {
			t.Fatalf("expected 0 violations for .PNG when png is allowed, got %d", len(v))
		}
	})

	t.Run("mixed batch reports only disallowed", func(t *testing.T) {
		files := []string{"a.png", "b.jpg", "c.webp", "d.tga"}
		v := CheckTextureFormat(policy, files)
		if len(v) != 2 {
			t.Fatalf("expected 2 violations (jpg, tga), got %d: %v", len(v), v)
		}
	})
}

func TestCheckSceneNodeCount(t *testing.T) {
	policy := &ScenePolicy{MaxNodeCount: 100}

	t.Run("nil policy returns no violations", func(t *testing.T) {
		v := CheckSceneNodeCount(nil, "scene.tscn", 9999)
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("zero limit returns no violations", func(t *testing.T) {
		v := CheckSceneNodeCount(&ScenePolicy{MaxNodeCount: 0}, "scene.tscn", 9999)
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("count at limit is not a violation", func(t *testing.T) {
		v := CheckSceneNodeCount(policy, "scene.tscn", 100)
		if len(v) != 0 {
			t.Fatalf("expected 0 violations, got %d", len(v))
		}
	})

	t.Run("count over limit is a violation", func(t *testing.T) {
		v := CheckSceneNodeCount(policy, "heavy.tscn", 101)
		if len(v) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(v))
		}
		if v[0].Rule != "scenes.max_node_count" {
			t.Errorf("unexpected rule %q", v[0].Rule)
		}
		if v[0].Path != "heavy.tscn" {
			t.Errorf("unexpected path %q", v[0].Path)
		}
	})
}

func TestViolationString(t *testing.T) {
	t.Run("with path", func(t *testing.T) {
		v := Violation{Rule: "assets.max_file_size_mb", Path: "big.png", Message: "2.0 MB > limit 1.0 MB"}
		got := v.String()
		want := "assets.max_file_size_mb: big.png (2.0 MB > limit 1.0 MB)"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without path", func(t *testing.T) {
		v := Violation{Rule: "scenes.max_node_count", Message: "101 nodes > limit 100"}
		got := v.String()
		want := "scenes.max_node_count: 101 nodes > limit 100"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
