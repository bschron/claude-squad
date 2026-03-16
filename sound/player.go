package sound

import (
	"claude-squad/config"
	"claude-squad/log"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var (
	mu       sync.Mutex
	lastPlay time.Time
	cooldown = 2 * time.Second
)

// Play plays the given system sound in a goroutine.
// It uses a mutex and cooldown to prevent audio spam.
// Errors are logged, never surfaced.
func Play(sound config.SoundOption) {
	go func() {
		mu.Lock()
		if time.Since(lastPlay) < cooldown {
			mu.Unlock()
			return
		}
		lastPlay = time.Now()
		mu.Unlock()

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			path := fmt.Sprintf("/System/Library/Sounds/%s.aiff", string(sound))
			cmd = exec.Command("afplay", path)
		case "linux":
			// Try paplay first (PulseAudio), then aplay (ALSA)
			fdPath := fmt.Sprintf("/usr/share/sounds/freedesktop/stereo/%s.oga", freedesktopSound(sound))
			if _, err := exec.LookPath("paplay"); err == nil {
				cmd = exec.Command("paplay", fdPath)
			} else if _, err := exec.LookPath("aplay"); err == nil {
				cmd = exec.Command("aplay", fdPath)
			} else {
				log.WarningLog.Printf("No sound player found on Linux")
				return
			}
		default:
			log.WarningLog.Printf("Sound alerts not supported on %s", runtime.GOOS)
			return
		}

		if err := cmd.Run(); err != nil {
			log.WarningLog.Printf("Failed to play sound %s: %v", sound, err)
		}
	}()
}

// freedesktopSound maps macOS sound names to FreeDesktop equivalents.
func freedesktopSound(sound config.SoundOption) string {
	switch sound {
	case config.SoundBasso, config.SoundFunk:
		return "dialog-error"
	case config.SoundBlow, config.SoundHero:
		return "dialog-information"
	case config.SoundBottle, config.SoundPop:
		return "message-new-instant"
	case config.SoundGlass, config.SoundPing, config.SoundTink:
		return "bell"
	case config.SoundFrog, config.SoundMorse, config.SoundSosumi:
		return "complete"
	case config.SoundPurr:
		return "service-login"
	case config.SoundSubmarine:
		return "service-logout"
	default:
		return "bell"
	}
}
