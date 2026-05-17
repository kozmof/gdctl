package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	embeddedaddons "gdctl/addons"
	"gdctl/internal/addon"
	"gdctl/internal/bridge"
)

var newAddonManager = func() addon.Manager {
	return addon.NewManager(embeddedaddons.FS)
}

// sceneMu serializes all commands that open, mutate, and save a scene via --scene,
// preventing cross-wire races when multiple gdctl invocations run concurrently in one process.
var sceneMu sync.Mutex

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	cfg, rest, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}
	cfg, err = cfg.WithProjectToken()
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		printUsage(stderr)
		return fmt.Errorf("missing command")
	}
	rest = normalizeCommandArgs(rest)

	client := bridge.NewClient(cfg)
	addonManager := newAddonManager()
	switch rest[0] {
	case "ping":
		return runPing(ctx, client, stdout)
	case "doctor":
		return runDoctor(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "addon":
		return runAddon(ctx, cfg, client, addonManager, rest[1:], stdout)
	case "bridge":
		return runBridge(ctx, client, addonManager, rest[1:], stdout)
	case "autoload":
		return runAutoload(ctx, client, rest[1:], stdout)
	case "input":
		return runInputMap(ctx, client, rest[1:], stdout)
	case "run":
		return runRun(ctx, client, rest[1:], stdout, stderr)
	case "test":
		return runTest(ctx, client, rest[1:], stdout, stderr)
	case "scene":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runSceneCreate(ctx, client, rest[2:], stdout)
			case "open":
				return runSceneOpen(ctx, client, rest[2:], stdout)
			case "instance":
				return runSceneInstance(ctx, client, rest[2:], stdout)
			case "tree":
				return runSceneTree(ctx, client, stdout)
			case "save":
				return runSceneSave(ctx, client, rest[2:], stdout)
			case "apply":
				return runSceneApply(ctx, client, rest[2:], stdout)
			case "batch":
				return runSceneBatch(ctx, client, rest[2:], stdout)
			case "apply-blueprint":
				return runSceneApplyBlueprint(ctx, client, rest[2:], stdout)
			case "list":
				return runSceneList(ctx, client, rest[2:], stdout)
			case "run":
				return runSceneRun(ctx, cfg, rest[2:], stdout)
			}
		}
	case "node":
		if len(rest) >= 2 {
			switch rest[1] {
			case "add":
				return runNodeAdd(ctx, client, rest[2:], stdout)
			case "remove":
				return runNodeRemove(ctx, client, rest[2:], stdout)
			case "rename":
				return runNodeRename(ctx, client, rest[2:], stdout)
			case "move":
				return runNodeMove(ctx, client, rest[2:], stdout)
			case "get":
				return runNodeGet(ctx, client, rest[2:], stdout)
			case "set":
				return runNodeSet(ctx, client, rest[2:], stdout)
			case "set-many":
				return runNodeSetMany(ctx, client, rest[2:], stdout)
			case "set-resource":
				return runNodeSetResource(ctx, client, rest[2:], stdout)
			case "attach-script":
				return runNodeAttachScript(ctx, client, rest[2:], stdout)
			case "group":
				if len(rest) >= 3 {
					switch rest[2] {
					case "add":
						return runNodeGroupAdd(ctx, client, rest[3:], stdout)
					case "remove":
						return runNodeGroupRemove(ctx, client, rest[3:], stdout)
					case "list":
						return runNodeGroupList(ctx, client, rest[3:], stdout)
					}
				}
			case "duplicate":
				return runNodeDuplicate(ctx, client, rest[2:], stdout)
			case "list-properties":
				return runNodeListProperties(ctx, client, rest[2:], stdout)
			}
		}
	case "script":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runScriptCreate(ctx, client, rest[2:], stdout)
			case "write":
				return runScriptWrite(ctx, client, rest[2:], stdout)
			case "check":
				return runScriptCheck(ctx, client, rest[2:], stdout)
			}
		}
	case "shader":
		if len(rest) >= 2 {
			switch rest[1] {
			case "write":
				return runShaderWrite(ctx, client, rest[2:], stdout)
			case "check":
				return runShaderCheck(ctx, client, rest[2:], stdout)
			}
		}
	case "resource":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runResourceCreate(ctx, client, rest[2:], stdout)
			case "list":
				return runResourceList(ctx, client, rest[2:], stdout)
			}
		}
	case "import":
		if len(rest) >= 2 {
			switch rest[1] {
			case "set":
				return runImportSet(ctx, client, rest[2:], stdout)
			}
		}
	case "file":
		if len(rest) >= 2 {
			switch rest[1] {
			case "write-bytes":
				return runFileWriteBytes(ctx, client, rest[2:], stdout)
			case "lut-write":
				return runLUTWrite(ctx, client, rest[2:], stdout)
			case "list":
				return runFileList(ctx, client, rest[2:], stdout)
			case "mkdir":
				return runFileMkdir(ctx, client, rest[2:], stdout)
			case "delete":
				return runFileDelete(ctx, client, rest[2:], stdout)
			case "exists":
				return runFileExists(ctx, client, rest[2:], stdout)
			}
		}
	case "navigation":
		if len(rest) >= 2 {
			switch rest[1] {
			case "bake":
				return runNavigationBake(ctx, client, rest[2:], stdout)
			}
		}
	case "signal":
		if len(rest) >= 2 {
			switch rest[1] {
			case "connect":
				return runSignalConnect(ctx, client, rest[2:], stdout)
			case "disconnect":
				return runSignalDisconnect(ctx, client, rest[2:], stdout)
			}
		}
	case "project":
		if len(rest) >= 2 {
			switch rest[1] {
			case "setting":
				if len(rest) >= 3 {
					switch rest[2] {
					case "get":
						return runProjectSettingGet(ctx, client, rest[3:], stdout)
					case "set":
						return runProjectSettingSet(ctx, client, rest[3:], stdout)
					}
				}
			case "run":
				return runProjectRun(ctx, cfg, rest[2:], stdout)
			}
		}
	case "viewport":
		if len(rest) >= 2 {
			switch rest[1] {
			case "screenshot":
				return runViewportScreenshot(ctx, client, rest[2:], stdout)
			case "set-size":
				return runViewportSetSize(ctx, client, rest[2:], stdout)
			case "add":
				return runViewportAdd(ctx, client, rest[2:], stdout)
			case "camera-assign":
				return runViewportCameraAssign(ctx, client, rest[2:], stdout)
			}
		}
	case "theme":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runThemeCreate(ctx, client, rest[2:], stdout)
			case "set-color":
				return runThemeSetColor(ctx, client, rest[2:], stdout)
			case "set-font-size":
				return runThemeSetFontSize(ctx, client, rest[2:], stdout)
			case "set-constant":
				return runThemeSetConstant(ctx, client, rest[2:], stdout)
			}
		}
	case "tilemap":
		if len(rest) >= 2 {
			switch rest[1] {
			case "tileset-create":
				return runTilesetCreate(ctx, client, rest[2:], stdout)
			case "source-add":
				return runTilesetSourceAdd(ctx, client, rest[2:], stdout)
			case "cell-set":
				return runTilemapCellSet(ctx, client, rest[2:], stdout)
			case "cell-set-rect":
				return runTilemapCellSetRect(ctx, client, rest[2:], stdout)
			case "cell-clear":
				return runTilemapCellClear(ctx, client, rest[2:], stdout)
			}
		}
	case "audio":
		if len(rest) >= 2 {
			switch rest[1] {
			case "bus-add":
				return runAudioBusAdd(ctx, client, rest[2:], stdout)
			case "bus-volume-set":
				return runAudioBusVolumeSet(ctx, client, rest[2:], stdout)
			case "bus-effect-add":
				return runAudioBusEffectAdd(ctx, client, rest[2:], stdout)
			case "listener-make-current":
				return runAudioListenerMakeCurrent(ctx, client, rest[2:], stdout)
			case "playlist-add":
				return runAudioPlaylistAdd(ctx, client, rest[2:], stdout)
			case "playlist-autoplay":
				return runAudioPlaylistAutoplay(ctx, client, rest[2:], stdout)
			}
		}
	case "animation":
		if len(rest) >= 2 {
			switch rest[1] {
			case "create":
				return runAnimationCreate(ctx, client, rest[2:], stdout)
			case "track-add":
				return runAnimationTrackAdd(ctx, client, rest[2:], stdout)
			case "keyframe-add":
				return runAnimationKeyframeAdd(ctx, client, rest[2:], stdout)
			case "length-set":
				return runAnimationLengthSet(ctx, client, rest[2:], stdout)
			case "player-play":
				return runAnimationPlayerPlay(ctx, client, rest[2:], stdout)
			case "tree":
				return runAnimationTree(ctx, client, rest[2:], stdout)
			}
		}
	case "softbody":
		return runSoftBody(ctx, client, rest[1:], stdout)
	case "lod":
		return runLOD(ctx, client, rest[1:], stdout)
	case "terrain":
		return runTerrain(ctx, client, rest[1:], stdout)
	case "lightmap":
		return runLightmap(ctx, client, rest[1:], stdout)
	case "voxelgi":
		return runVoxelGI(ctx, client, rest[1:], stdout)
	case "reflection-probe":
		return runReflectionProbe(ctx, client, rest[1:], stdout)
	case "window":
		return runWindow(ctx, client, rest[1:], stdout)
	case "graph-edit":
		return runGraphEdit(ctx, client, rest[1:], stdout)
	case "accessibility":
		return runAccessibility(ctx, client, rest[1:], stdout)
	case "i18n":
		return runI18n(ctx, client, rest[1:], stdout)
	case "csg":
		return runCSG(ctx, client, rest[1:], stdout)
	case "environment":
		return runEnvironment(ctx, client, rest[1:], stdout)
	case "decal":
		return runDecal(ctx, client, rest[1:], stdout)
	case "fog-volume":
		return runFogVolume(ctx, client, rest[1:], stdout)
	case "occluder":
		return runOccluder(ctx, client, rest[1:], stdout)
	case "help":
		return runHelp(rest[1:], stdout)
	}

	printUsage(stderr)
	return fmt.Errorf("unknown command: %s", strings.Join(rest, " "))
}

func normalizeCommandArgs(args []string) []string {
	if len(args) >= 2 && args[0] == "help" && strings.Contains(args[1], ".") {
		parts := strings.Split(args[1], ".")
		normalized := []string{"help"}
		for _, part := range parts {
			if part != "" {
				normalized = append(normalized, part)
			}
		}
		if len(normalized) > 1 {
			return append(normalized, args[2:]...)
		}
	}
	if len(args) == 0 || !strings.Contains(args[0], ".") {
		return args
	}
	parts := strings.Split(args[0], ".")
	normalized := make([]string, 0, len(parts)+len(args)-1)
	for _, part := range parts {
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	if len(normalized) == 0 {
		return args
	}
	return append(normalized, args[1:]...)
}

// isSuspectedEditorCapture returns true when the image looks like a uniform
// desktop/editor background: samples a 10x10 grid and considers it uniform if
// ≥90% of sampled pixels are within ±10 of the dominant color per channel.
func parseGlobalFlags(args []string) (bridge.Config, []string, error) {
	cfg := bridge.ConfigFromEnv()
	fs := flag.NewFlagSet("gdctl", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	host := fs.String("host", cfg.Host, "bridge host")
	port := fs.Int("port", cfg.Port, "bridge port")
	token := fs.String("token", cfg.Token, "bridge bearer token")
	project := fs.String("project", cfg.Project, "Godot project path")
	godot := fs.String("godot", cfg.GodotPath, "headless Godot binary path")
	if err := fs.Parse(args); err != nil {
		return cfg, nil, err
	}
	cfg.Host = *host
	cfg.Port = *port
	cfg.Token = *token
	cfg.Project = *project
	cfg.GodotPath = *godot
	return cfg, fs.Args(), nil
}

func runPing(ctx context.Context, client *bridge.Client, stdout io.Writer) error {
	ping, err := client.Ping(ctx)
	if err != nil {
		return err
	}
	if !ping.OK {
		return fmt.Errorf("godot bridge returned not ok")
	}
	fmt.Fprintln(stdout, "Godot bridge: ok")
	fmt.Fprintf(stdout, "Engine: %s %s\n", ping.Engine, ping.EngineVersion)
	fmt.Fprintf(stdout, "Project: %s\n", ping.ProjectName)
	fmt.Fprintf(stdout, "Plugin: %s\n", ping.PluginVersion)
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Run 'gdctl help' to see all available commands.")
	return nil
}

func printBridgeInfo(stdout io.Writer, ping bridge.PingResponse) {
	fmt.Fprintln(stdout, "Godot bridge")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "  reachable: yes")
	fmt.Fprintf(stdout, "  service: %s\n", valueOrDash(ping.Service))
	fmt.Fprintf(stdout, "  engine: %s %s\n", valueOrDash(ping.Engine), valueOrDash(ping.EngineVersion))
	fmt.Fprintf(stdout, "  project: %s\n", valueOrDash(ping.ProjectName))
	fmt.Fprintf(stdout, "  project_path: %s\n", valueOrDash(ping.ProjectPath))
	fmt.Fprintf(stdout, "  plugin_version: %s\n", valueOrDash(ping.PluginVersion))
	fmt.Fprintf(stdout, "  protocol: %s\n", valueOrDash(ping.ProtocolVersion))
	fmt.Fprintf(stdout, "  auth_enabled: %s\n", yesNo(ping.AuthEnabled))
	fmt.Fprintf(stdout, "  listen: %s:%d\n", valueOrDash(ping.Host), ping.Port)
	fmt.Fprintf(stdout, "  scene_open: %s\n", yesNo(ping.SceneOpen))
	if len(ping.Capabilities) > 0 {
		fmt.Fprintf(stdout, "  capabilities: %s\n", strings.Join(ping.Capabilities, ", "))
	}
}

func requestID() string {
	return "cli-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
