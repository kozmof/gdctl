package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

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

type commandRoute struct {
	layer   string
	handler func(ctx context.Context, client *bridge.Client, addonMgr addon.Manager, cfg bridge.Config, rest []string, stdout, stderr io.Writer) error
}

// routes is initialized in init() to break the cycle:
// routeWorkflow → runWorkflow → runWorkflowRun → Run → routes
var routes map[string]commandRoute

func init() {
	routes = map[string]commandRoute{
		// Object Layer
		"scene":    {layer: "object", handler: routeScene},
		"node":     {layer: "object", handler: routeNode},
		"script":   {layer: "object", handler: routeScript},
		"shader":   {layer: "object", handler: routeShader},
		"resource": {layer: "object", handler: routeResource},
		"file":     {layer: "object", handler: routeFile},
		// System Layer
		"navigation":    {layer: "system", handler: routeNavigation},
		"localization":  {layer: "system", handler: routeLocalization},
		"audio":         {layer: "system", handler: routeAudio},
		"animation":     {layer: "system", handler: routeAnimation},
		"tilemap":       {layer: "system", handler: routeTilemap},
		"theme":         {layer: "system", handler: routeTheme},
		"viewport":      {layer: "system", handler: routeViewport},
		"window":        {layer: "system", handler: routeWindow},
		"accessibility": {layer: "system", handler: routeAccessibility},
		"lightmap":      {layer: "system", handler: routeLightmap},
		"save":          {layer: "system", handler: routeSave},
		// Policy Layer
		"policy": {layer: "policy", handler: routePolicy},
		// Workflow Layer
		"apply":    {layer: "workflow", handler: routeApply},
		"plan":     {layer: "workflow", handler: routePlan},
		"diff":     {layer: "workflow", handler: routeDiff},
		"tx":       {layer: "workflow", handler: routeTx},
		"workflow": {layer: "workflow", handler: routeWorkflow},
		"scaffold": {layer: "workflow", handler: routeScaffold},
		// Execution Layer
		"asset": {layer: "execution", handler: routeAsset},
		"lint":  {layer: "execution", handler: routeLint},
		"test":  {layer: "execution", handler: routeTest},
		"gate":  {layer: "execution", handler: routeGate},
		"perf":  {layer: "execution", handler: routePerf},
		// Recipe Layer
		"recipe": {layer: "recipe", handler: routeRecipe},
		// Infrastructure
		"ping":     {layer: "infra", handler: routePing},
		"doctor":   {layer: "infra", handler: routeDoctor},
		"addon":    {layer: "infra", handler: routeAddon},
		"bridge":   {layer: "infra", handler: routeBridge},
		"autoload": {layer: "infra", handler: routeAutoload},
		"input":    {layer: "infra", handler: routeInput},
		"run":      {layer: "infra", handler: routeRun},
		"signal":   {layer: "infra", handler: routeSignal},
		"project":  {layer: "infra", handler: routeProject},
		"import":   {layer: "infra", handler: routeImport},
		"help":     {layer: "infra", handler: routeHelp},
	}
}

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

	route, ok := routes[rest[0]]
	if !ok {
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", strings.Join(rest, " "))
	}
	return route.handler(ctx, client, addonManager, cfg, rest[1:], stdout, stderr)
}

// Route dispatchers — Object Layer

func routeScene(ctx context.Context, client *bridge.Client, _ addon.Manager, cfg bridge.Config, rest []string, stdout, stderr io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("scene requires a subcommand: create, open, instance, tree, save, apply, batch, apply-blueprint, list, run")
	}
	switch rest[0] {
	case "create":
		return runSceneCreate(ctx, client, rest[1:], stdout)
	case "open":
		return runSceneOpen(ctx, client, rest[1:], stdout)
	case "instance":
		return runSceneInstance(ctx, client, rest[1:], stdout)
	case "tree":
		return runSceneTree(ctx, client, stdout)
	case "save":
		return runSceneSave(ctx, client, rest[1:], stdout)
	case "apply":
		return runSceneApply(ctx, client, rest[1:], stdout)
	case "batch":
		return runSceneBatch(ctx, client, rest[1:], stdout)
	case "apply-blueprint":
		return runSceneApplyBlueprint(ctx, client, rest[1:], stdout)
	case "list":
		return runSceneList(ctx, client, rest[1:], stdout)
	case "run":
		return runSceneRun(ctx, cfg, rest[1:], stdout)
	}
	return fmt.Errorf("unknown scene subcommand: %s", rest[0])
}

func routeNode(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("node requires a subcommand: add, remove, rename, move, get, set, set-many, set-resource, attach-script, group, duplicate, list-properties")
	}
	switch rest[0] {
	case "add":
		return runNodeAdd(ctx, client, rest[1:], stdout)
	case "remove":
		return runNodeRemove(ctx, client, rest[1:], stdout)
	case "rename":
		return runNodeRename(ctx, client, rest[1:], stdout)
	case "move":
		return runNodeMove(ctx, client, rest[1:], stdout)
	case "get":
		return runNodeGet(ctx, client, rest[1:], stdout)
	case "set":
		return runNodeSet(ctx, client, rest[1:], stdout)
	case "set-many":
		return runNodeSetMany(ctx, client, rest[1:], stdout)
	case "set-resource":
		return runNodeSetResource(ctx, client, rest[1:], stdout)
	case "attach-script":
		return runNodeAttachScript(ctx, client, rest[1:], stdout)
	case "group":
		if len(rest) >= 2 {
			switch rest[1] {
			case "add":
				return runNodeGroupAdd(ctx, client, rest[2:], stdout)
			case "remove":
				return runNodeGroupRemove(ctx, client, rest[2:], stdout)
			case "list":
				return runNodeGroupList(ctx, client, rest[2:], stdout)
			}
		}
		return fmt.Errorf("node group requires a subcommand: add, remove, list")
	case "duplicate":
		return runNodeDuplicate(ctx, client, rest[1:], stdout)
	case "list-properties":
		return runNodeListProperties(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown node subcommand: %s", rest[0])
}

func routeScript(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("script requires a subcommand: create, write, check")
	}
	switch rest[0] {
	case "create":
		return runScriptCreate(ctx, client, rest[1:], stdout)
	case "write":
		return runScriptWrite(ctx, client, rest[1:], stdout)
	case "check":
		return runScriptCheck(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown script subcommand: %s", rest[0])
}

func routeShader(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("shader requires a subcommand: write, check")
	}
	switch rest[0] {
	case "write":
		return runShaderWrite(ctx, client, rest[1:], stdout)
	case "check":
		return runShaderCheck(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown shader subcommand: %s", rest[0])
}

func routeResource(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("resource requires a subcommand: create, list")
	}
	switch rest[0] {
	case "create":
		return runResourceCreate(ctx, client, rest[1:], stdout)
	case "list":
		return runResourceList(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown resource subcommand: %s", rest[0])
}

func routeFile(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("file requires a subcommand: write-bytes, lut-write, list, mkdir, delete, exists")
	}
	switch rest[0] {
	case "write-bytes":
		return runFileWriteBytes(ctx, client, rest[1:], stdout)
	case "lut-write":
		return runLUTWrite(ctx, client, rest[1:], stdout)
	case "list":
		return runFileList(ctx, client, rest[1:], stdout)
	case "mkdir":
		return runFileMkdir(ctx, client, rest[1:], stdout)
	case "delete":
		return runFileDelete(ctx, client, rest[1:], stdout)
	case "exists":
		return runFileExists(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown file subcommand: %s", rest[0])
}

// Route dispatchers — System Layer

func routeNavigation(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("navigation requires a subcommand: bake")
	}
	switch rest[0] {
	case "bake":
		return runNavigationBake(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown navigation subcommand: %s", rest[0])
}

func routeLocalization(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runLocalization(ctx, client, rest, stdout)
}

func routeAudio(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("audio requires a subcommand: bus-add, bus-volume-set, bus-effect-add, listener-make-current, playlist-add, playlist-autoplay")
	}
	switch rest[0] {
	case "bus-add":
		return runAudioBusAdd(ctx, client, rest[1:], stdout)
	case "bus-volume-set":
		return runAudioBusVolumeSet(ctx, client, rest[1:], stdout)
	case "bus-effect-add":
		return runAudioBusEffectAdd(ctx, client, rest[1:], stdout)
	case "listener-make-current":
		return runAudioListenerMakeCurrent(ctx, client, rest[1:], stdout)
	case "playlist-add":
		return runAudioPlaylistAdd(ctx, client, rest[1:], stdout)
	case "playlist-autoplay":
		return runAudioPlaylistAutoplay(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown audio subcommand: %s", rest[0])
}

func routeAnimation(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("animation requires a subcommand: create, track-add, keyframe-add, length-set, player-play, tree")
	}
	switch rest[0] {
	case "create":
		return runAnimationCreate(ctx, client, rest[1:], stdout)
	case "track-add":
		return runAnimationTrackAdd(ctx, client, rest[1:], stdout)
	case "keyframe-add":
		return runAnimationKeyframeAdd(ctx, client, rest[1:], stdout)
	case "length-set":
		return runAnimationLengthSet(ctx, client, rest[1:], stdout)
	case "player-play":
		return runAnimationPlayerPlay(ctx, client, rest[1:], stdout)
	case "tree":
		return runAnimationTree(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown animation subcommand: %s", rest[0])
}

func routeTilemap(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("tilemap requires a subcommand: tileset-create, source-add, cell-set, cell-set-rect, cell-clear")
	}
	switch rest[0] {
	case "tileset-create":
		return runTilesetCreate(ctx, client, rest[1:], stdout)
	case "source-add":
		return runTilesetSourceAdd(ctx, client, rest[1:], stdout)
	case "cell-set":
		return runTilemapCellSet(ctx, client, rest[1:], stdout)
	case "cell-set-rect":
		return runTilemapCellSetRect(ctx, client, rest[1:], stdout)
	case "cell-clear":
		return runTilemapCellClear(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown tilemap subcommand: %s", rest[0])
}

func routeTheme(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("theme requires a subcommand: create, set-color, set-font-size, set-constant")
	}
	switch rest[0] {
	case "create":
		return runThemeCreate(ctx, client, rest[1:], stdout)
	case "set-color":
		return runThemeSetColor(ctx, client, rest[1:], stdout)
	case "set-font-size":
		return runThemeSetFontSize(ctx, client, rest[1:], stdout)
	case "set-constant":
		return runThemeSetConstant(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown theme subcommand: %s", rest[0])
}

func routeViewport(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("viewport requires a subcommand: screenshot, set-size, add, camera-assign")
	}
	switch rest[0] {
	case "screenshot":
		return runViewportScreenshot(ctx, client, rest[1:], stdout)
	case "set-size":
		return runViewportSetSize(ctx, client, rest[1:], stdout)
	case "add":
		return runViewportAdd(ctx, client, rest[1:], stdout)
	case "camera-assign":
		return runViewportCameraAssign(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown viewport subcommand: %s", rest[0])
}

func routeWindow(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runWindow(ctx, client, rest, stdout)
}

func routeAccessibility(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runAccessibility(ctx, client, rest, stdout)
}

func routeLightmap(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runLightmap(ctx, client, rest, stdout)
}

// Route dispatchers — Policy Layer

func routePolicy(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runPolicy(ctx, client, rest, stdout)
}

// Route dispatchers — System Layer (additional)

func routeSave(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runSceneSave(ctx, client, rest, stdout)
}

// Route dispatchers — Workflow Layer

func routeApply(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runApply(ctx, client, rest, stdout)
}

func routePlan(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runPlan(ctx, client, rest, stdout)
}

func routeDiff(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runDiff(ctx, client, rest, stdout)
}

func routeTx(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, stderr io.Writer) error {
	return runTx(ctx, client, rest, stdout, stderr)
}

func routeWorkflow(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, stderr io.Writer) error {
	return runWorkflow(ctx, client, rest, stdout, stderr)
}

func routeScaffold(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runScaffold(ctx, client, rest, stdout)
}

// Route dispatchers — Execution Layer

func routeAsset(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runAsset(ctx, client, rest, stdout)
}

func routeLint(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runLint(ctx, client, rest, stdout)
}

func routeTest(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, stderr io.Writer) error {
	return runTest(ctx, client, rest, stdout, stderr)
}

func routeGate(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, stderr io.Writer) error {
	return runGate(ctx, client, rest, stdout, stderr)
}

func routePerf(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runRunProfile(ctx, client, rest, stdout)
}

// Route dispatchers — Recipe Layer

func routeRecipe(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runRecipe(ctx, client, rest, stdout)
}

// Route dispatchers — Infrastructure

func routePing(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, _ []string, stdout, _ io.Writer) error {
	return runPing(ctx, client, stdout)
}

func routeDoctor(ctx context.Context, client *bridge.Client, addonMgr addon.Manager, cfg bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runDoctor(ctx, cfg, client, addonMgr, rest, stdout)
}

func routeAddon(ctx context.Context, client *bridge.Client, addonMgr addon.Manager, cfg bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runAddon(ctx, cfg, client, addonMgr, rest, stdout)
}

func routeBridge(ctx context.Context, client *bridge.Client, addonMgr addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runBridge(ctx, client, addonMgr, rest, stdout)
}

func routeAutoload(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runAutoload(ctx, client, rest, stdout)
}

func routeInput(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runInputMap(ctx, client, rest, stdout)
}

func routeRun(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, stderr io.Writer) error {
	return runRun(ctx, client, rest, stdout, stderr)
}

func routeSignal(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("signal requires a subcommand: connect, disconnect")
	}
	switch rest[0] {
	case "connect":
		return runSignalConnect(ctx, client, rest[1:], stdout)
	case "disconnect":
		return runSignalDisconnect(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown signal subcommand: %s", rest[0])
}

func routeProject(ctx context.Context, client *bridge.Client, _ addon.Manager, cfg bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("project requires a subcommand: setting, run")
	}
	switch rest[0] {
	case "setting":
		if len(rest) >= 2 {
			switch rest[1] {
			case "get":
				return runProjectSettingGet(ctx, client, rest[2:], stdout)
			case "set":
				return runProjectSettingSet(ctx, client, rest[2:], stdout)
			}
		}
		return fmt.Errorf("project setting requires a subcommand: get, set")
	case "run":
		return runProjectRun(ctx, cfg, rest[1:], stdout)
	}
	return fmt.Errorf("unknown project subcommand: %s", rest[0])
}

func routeImport(ctx context.Context, client *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	if len(rest) == 0 {
		return fmt.Errorf("import requires a subcommand: set")
	}
	switch rest[0] {
	case "set":
		return runImportSet(ctx, client, rest[1:], stdout)
	}
	return fmt.Errorf("unknown import subcommand: %s", rest[0])
}

func routeHelp(_ context.Context, _ *bridge.Client, _ addon.Manager, _ bridge.Config, rest []string, stdout, _ io.Writer) error {
	return runHelp(rest, stdout)
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

func parseGlobalFlags(args []string) (bridge.Config, []string, error) {
	cfg, err := bridge.ConfigFromEnv()
	if err != nil {
		return cfg, nil, err
	}
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
		if errors.Is(err, bridge.ErrUnreachable) {
			return fmt.Errorf("%w — is the Godot editor running with the TCP bridge addon enabled? Try 'gdctl doctor'", err)
		}
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

var reqCounter atomic.Int64

func requestID() string {
	return "cli-" + strconv.FormatInt(reqCounter.Add(1), 10)
}

// extractPositionalArg splits args into the first non-flag argument and the
// remaining flag args. This lets callers mix `cmd file --flag val` and
// `cmd --flag val file` without ambiguity.
func extractPositionalArg(args []string) (positional string, rest []string) {
	for i, a := range args {
		if len(a) > 0 && a[0] != '-' {
			return a, append(args[:i:i], args[i+1:]...)
		}
	}
	return "", args
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

// serverNote returns a parenthesized suffix for any "note" the bridge attached
// to a successful result, or "" when there is none. It lets server-side caveats
// (e.g. "TTS not supported on this platform") surface to the user instead of
// being silently discarded by the caller.
func serverNote(result map[string]any) string {
	if note, _ := result["note"].(string); note != "" {
		return " (" + note + ")"
	}
	return ""
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
