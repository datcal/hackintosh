//go:build windows

package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// fetchNowPlaying invokes PowerShell to query SMTC. The snippet is small but
// async — we await it inside the script and emit a single JSON line.
//
// The script returns "{}" if no session exists, otherwise a JSON object with
// Title, Artist, IsPlaying, PositionMs, LengthMs fields.
func fetchNowPlaying(ctx context.Context) (store.Media, error) {
	const script = `
$ErrorActionPreference = "SilentlyContinue"
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGen = ([System.WindowsRuntimeSystemExtensions].GetMethods() |
    Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and
        $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0]
function Await($op, $ret) {
    $task = $asTaskGen.MakeGenericMethod($ret).Invoke($null, @($op))
    $task.Wait(2000) | Out-Null
    if ($task.IsCompleted) { return $task.Result }
    return $null
}
[Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager,Windows.Media.Control,ContentType=WindowsRuntime] | Out-Null
[Windows.Media.Control.GlobalSystemMediaTransportControlsSession,Windows.Media.Control,ContentType=WindowsRuntime] | Out-Null
$mgrOp = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()
$mgr = Await $mgrOp ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])
if ($mgr -eq $null) { '{}'; exit }
$session = $mgr.GetCurrentSession()
if ($session -eq $null) { '{}'; exit }
$propsOp = $session.TryGetMediaPropertiesAsync()
$props = Await $propsOp ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
if ($props -eq $null) { '{}'; exit }
$timeline = $session.GetTimelineProperties()
$playback = $session.GetPlaybackInfo()
$result = @{
    Title    = $props.Title
    Artist   = $props.Artist
    IsPlaying = ($playback.PlaybackStatus -eq [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionPlaybackStatus]::Playing)
    PositionMs = [int]$timeline.Position.TotalMilliseconds
    LengthMs   = [int]$timeline.EndTime.TotalMilliseconds
}
$result | ConvertTo-Json -Compress
`
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, "powershell",
		"-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return store.Media{}, fmt.Errorf("powershell smtc: %w", err)
	}
	out := strings.TrimSpace(stdout.String())
	if out == "" || out == "{}" {
		return store.Media{Valid: false}, nil
	}

	var data struct {
		Title      string `json:"Title"`
		Artist     string `json:"Artist"`
		IsPlaying  bool   `json:"IsPlaying"`
		PositionMs int    `json:"PositionMs"`
		LengthMs   int    `json:"LengthMs"`
	}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return store.Media{Valid: false}, nil
	}
	if data.Title == "" && data.Artist == "" {
		return store.Media{Valid: false}, nil
	}
	return store.Media{
		Valid:    true,
		Title:    data.Title,
		Artist:   data.Artist,
		Playing:  data.IsPlaying,
		Position: time.Duration(data.PositionMs) * time.Millisecond,
		Length:   time.Duration(data.LengthMs) * time.Millisecond,
	}, nil
}

func isUnusual(err error) bool { return false }
