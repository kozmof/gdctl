# Spider CLI Creation Report

Date: 2026-05-17
Project: New Game Project (`C:/Users/kozoo/OneDrive/Documents/new-game-project/`)
Bridge: Godot 4.6.2-stable, gdctl bridge plugin 0.2.0

## Created

- Added `res://SpiderGame.gd` through `gdctl script write`.
- Added `res://SpiderGame.tscn` through `gdctl scene create`.
- Attached the script to `/root/SpiderGame` and saved the scene.
- Set `application/run/main_scene` to `res://SpiderGame.tscn`.
- Added local CLI playback input at `generated/spider_walk_input.json`.
- Captured screen evidence at `screenshots/spider_game_screen.png`.

## Game Design

The scene is a 2D four-legged spider-like player. The script builds the visible world at runtime: dark background, curved terrain, four articulated legs, foot contact markers, body, head, eyes, and a HUD. Movement supports `A`/`D` plus `ui_left`/`ui_right`. Each leg chooses a ground target from the terrain function and snaps the foot y-position to the computed ground height every frame, so the intended max foot-ground error is `0.0 px` after each update.

## CLI Validation

Successful checks:

- `gdctl bridge info` reached the authenticated bridge.
- `gdctl script write --path res://SpiderGame.gd --body-file generated/spider_game.gd` succeeded after one type-inference fix.
- `gdctl script check --path res://SpiderGame.gd` returned `Script OK`.
- `gdctl scene create`, `node attach-script`, `scene save`, and `project setting set` succeeded.
- `go test ./...` passed for all Go packages.

Runtime findings:

- `gdctl run start --scene res://SpiderGame.tscn` reported the scene as running.
- `gdctl run status --json` reported `playing_scene: res://SpiderGame.tscn` and `paused: false`.
- `gdctl run wait-probe` and `gdctl run input` timed out because the runtime helper did not check in.
- The scene script now attempts to instantiate `res://addons/godot_tcp_bridge/runtime/runtime_bridge.gd` as `/root/GdctlRuntimeBridge` if the autoload is absent, but helper logs still did not appear in this session.
- `gdctl run screenshot --source screen` showed the Godot debug/project window rather than a confirmed game viewport, so automated runtime movement validation remains inconclusive.

## Findings For CLI Improvement

1. `run status` can report a scene as running while the runtime helper has not checked in. A stronger status state such as `running_without_helper` would make this easier to diagnose.
2. `run input` times out only after polling the helper job. It would be friendlier if it failed fast when helper status says `runtime helper has not checked in`.
3. Shell users can accidentally pass `--assert grounded_feet>=4` and have `>` interpreted by the shell. The split flags worked better: `--assert-key grounded_feet --assert-op ">=" --assert-value 4`.
4. `run screenshot --source screen` is useful as a fallback, but the captured host screen can show the editor/project manager rather than the game viewport. A CLI warning when helper screenshots are unavailable would help.
5. Creating a whole game through the CLI worked well for script, scene, attachment, saving, and project settings. The weak point is runtime-helper observability after launch.

## Next Steps

- Restart Godot or reload the project once, then re-run `gdctl run start --scene res://SpiderGame.tscn`. The autoload setting was written, and a fresh run may allow the helper to check in before input playback.
- Add a CLI preflight inside `run input`, `run screenshot --source game`, and `run wait-probe` to surface missing helper status before queueing jobs.
- Consider adding a direct runtime error/log stream independent of `GdctlRuntimeBridge`, so script startup failures are visible even when the helper is absent.
