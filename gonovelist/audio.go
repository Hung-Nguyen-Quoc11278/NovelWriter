package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
)

// Các khóa lưu trữ cài đặt Âm thanh giao diện trong Fyne Preferences
const (
	PrefKeySoundEnabled       string = "gonovelist.audio.enabled"
	PrefKeyTypingSoundEnabled string = "gonovelist.audio.typing_enabled"
	PrefKeyClickSoundEnabled  string = "gonovelist.audio.click_enabled"
	PrefKeySoundProfile       string = "gonovelist.audio.profile"
	PrefKeySoundVolume        string = "gonovelist.audio.volume"
)

// SoundProfile định danh kiểu âm sắc gõ phím / nhấp chuột.
type SoundProfile string

const (
	SoundProfileTypewriter SoundProfile = "typewriter" // Máy chữ cổ điển (Typewriter)
	SoundProfileMechanical SoundProfile = "mechanical" // Phím cơ trầm (Mechanical Brown)
	SoundProfileSoftTick   SoundProfile = "soft"       // Giọt nước nhẹ (Soft Tick)
)

// SoundProfileLabel trả về tên hiển thị Tiếng Việt của kiểu âm thanh.
func SoundProfileLabel(p SoundProfile) string {
	switch p {
	case SoundProfileMechanical:
		return "Phím cơ trầm (Mechanical)"
	case SoundProfileSoftTick:
		return "Tách nhẹ êm tai (Soft Tick)"
	default:
		return "Máy chữ cổ điển (Typewriter)"
	}
}

// AllSoundProfileLabels trả về danh sách nhãn Tiếng Việt cho hộp chọn kiểu âm thanh.
func AllSoundProfileLabels() []string {
	return []string{
		SoundProfileLabel(SoundProfileTypewriter),
		SoundProfileLabel(SoundProfileMechanical),
		SoundProfileLabel(SoundProfileSoftTick),
	}
}

// ParseSoundProfileLabel chuyển đổi nhãn Tiếng Việt về mã SoundProfile.
func ParseSoundProfileLabel(raw string) SoundProfile {
	switch strings.TrimSpace(raw) {
	case "Phím cơ trầm (Mechanical)", string(SoundProfileMechanical):
		return SoundProfileMechanical
	case "Tách nhẹ êm tai (Soft Tick)", string(SoundProfileSoftTick):
		return SoundProfileSoftTick
	default:
		return SoundProfileTypewriter
	}
}

// SoundEventType phân biệt loại sự kiện âm thanh giao diện.
type SoundEventType int

const (
	SoundEventKeyTap SoundEventType = iota
	SoundEventKeySpaceOrReturn
	SoundEventButtonClick
)

// CustomPCMStreamPlayer cho phép gắn trực tiếp bộ phát âm thanh bộ nhớ (như ebitengine/oto/v3) nếu có.
type CustomPCMStreamPlayer func(wavBytes []byte) error

// SoundManager quản lý trạng thái Âm thanh giao diện, bộ tổng hợp âm thanh WAV 16-bit PCM trong bộ nhớ
// và bộ phát bất đồng bộ (non-blocking) an toàn cho giao diện Fyne v2.
type SoundManager struct {
	mu                 sync.RWMutex
	enabled            bool
	typingSoundEnabled bool
	clickSoundEnabled  bool
	profile            SoundProfile
	volume             float64 // 0.10 -> 1.00

	lastPlayedAt time.Time
	minInterval  time.Duration

	sfxDir       string
	cachedFiles  map[string]string
	cachedMemory map[string][]byte

	playerCmd    string
	playerArgs   []string
	customPlayer CustomPCMStreamPlayer
}

// GlobalSoundManager là thực thể quản lý âm thanh giao diện toàn cục của GoNovelist.
var GlobalSoundManager = NewSoundManager()

// NewSoundManager khởi tạo bộ quản lý âm thanh giao diện với cấu hình mặc định.
func NewSoundManager() *SoundManager {
	sm := &SoundManager{
		enabled:            true,
		typingSoundEnabled: true,
		clickSoundEnabled:  true,
		profile:            SoundProfileTypewriter,
		volume:             0.70,
		minInterval:        22 * time.Millisecond,
		cachedFiles:        make(map[string]string),
		cachedMemory:       make(map[string][]byte),
	}
	sm.detectSystemAudioBackend()
	sm.rebuildSynthesizedWAVCache()
	return sm
}

// LoadFromPreferences nạp cấu hình Âm thanh giao diện đã lưu từ Fyne Preferences.
func (sm *SoundManager) LoadFromPreferences(prefs fyne.Preferences) {
	if prefs == nil {
		return
	}
	sm.mu.Lock()
	sm.enabled = prefs.BoolWithFallback(PrefKeySoundEnabled, true)
	sm.typingSoundEnabled = prefs.BoolWithFallback(PrefKeyTypingSoundEnabled, true)
	sm.clickSoundEnabled = prefs.BoolWithFallback(PrefKeyClickSoundEnabled, true)
	sm.profile = ParseSoundProfileLabel(prefs.StringWithFallback(PrefKeySoundProfile, string(SoundProfileTypewriter)))
	vol := prefs.FloatWithFallback(PrefKeySoundVolume, 0.70)
	if vol < 0.05 {
		vol = 0.05
	}
	if vol > 1.0 {
		vol = 1.0
	}
	sm.volume = vol
	sm.mu.Unlock()

	sm.rebuildSynthesizedWAVCache()
}

// SaveToPreferences lưu cấu hình Âm thanh giao diện hiện tại vào Fyne Preferences.
func (sm *SoundManager) SaveToPreferences(prefs fyne.Preferences) {
	if prefs == nil {
		return
	}
	sm.mu.RLock()
	enabled := sm.enabled
	typing := sm.typingSoundEnabled
	click := sm.clickSoundEnabled
	profile := string(sm.profile)
	vol := sm.volume
	sm.mu.RUnlock()

	prefs.SetBool(PrefKeySoundEnabled, enabled)
	prefs.SetBool(PrefKeyTypingSoundEnabled, typing)
	prefs.SetBool(PrefKeyClickSoundEnabled, click)
	prefs.SetString(PrefKeySoundProfile, profile)
	prefs.SetFloat(PrefKeySoundVolume, vol)
}

// SetCustomStreamPlayer cho phép tích hợp trực tiếp thư viện âm thanh bộ nhớ (ví dụ oto.Context).
func (sm *SoundManager) SetCustomStreamPlayer(fn CustomPCMStreamPlayer) {
	sm.mu.Lock()
	sm.customPlayer = fn
	sm.mu.Unlock()
}

func (sm *SoundManager) IsEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.enabled
}

func (sm *SoundManager) SetEnabled(v bool) {
	sm.mu.Lock()
	sm.enabled = v
	sm.mu.Unlock()
}

func (sm *SoundManager) IsTypingSoundEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.typingSoundEnabled
}

func (sm *SoundManager) SetTypingSoundEnabled(v bool) {
	sm.mu.Lock()
	sm.typingSoundEnabled = v
	sm.mu.Unlock()
}

func (sm *SoundManager) IsClickSoundEnabled() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.clickSoundEnabled
}

func (sm *SoundManager) SetClickSoundEnabled(v bool) {
	sm.mu.Lock()
	sm.clickSoundEnabled = v
	sm.mu.Unlock()
}

func (sm *SoundManager) Profile() SoundProfile {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.profile
}

func (sm *SoundManager) SetProfile(p SoundProfile) {
	sm.mu.Lock()
	sm.profile = p
	sm.mu.Unlock()
	sm.rebuildSynthesizedWAVCache()
}

func (sm *SoundManager) Volume() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.volume
}

func (sm *SoundManager) SetVolume(v float64) {
	if v < 0.05 {
		v = 0.05
	}
	if v > 1.0 {
		v = 1.0
	}
	sm.mu.Lock()
	sm.volume = v
	sm.mu.Unlock()
	sm.rebuildSynthesizedWAVCache()
}

// detectSystemAudioBackend tự động dò tìm công cụ phát âm thanh độ trễ thấp có sẵn trên hệ điều hành.
func (sm *SoundManager) detectSystemAudioBackend() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	sm.sfxDir = filepath.Join(homeDir, ".gonovelist", "sfx")
	_ = os.MkdirAll(sm.sfxDir, 0755)

	switch runtime.GOOS {
	case "linux":
		candidates := []struct {
			bin  string
			args []string
		}{
			{bin: "pw-play", args: []string{}},
			{bin: "paplay", args: []string{}},
			{bin: "aplay", args: []string{"-q"}},
			{bin: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet"}},
		}
		for _, c := range candidates {
			if path, err := exec.LookPath(c.bin); err == nil && path != "" {
				sm.playerCmd = path
				sm.playerArgs = c.args
				return
			}
		}
	case "darwin":
		if path, err := exec.LookPath("afplay"); err == nil && path != "" {
			sm.playerCmd = path
			sm.playerArgs = []string{}
			return
		}
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil && path != "" {
			sm.playerCmd = path
			return
		}
	}
}

// rebuildSynthesizedWAVCache tổng hợp sẵn 3 tệp âm thanh WAV 16-bit PCM chuẩn 44.1kHz trong bộ nhớ
// để phát tức thì không gây độ trễ khi gõ phím nhanh.
func (sm *SoundManager) rebuildSynthesizedWAVCache() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	vol := sm.volume
	prof := sm.profile

	tapWAV := synthesizeMechanicalSoundWAV(prof, SoundEventKeyTap, vol)
	spaceWAV := synthesizeMechanicalSoundWAV(prof, SoundEventKeySpaceOrReturn, vol)
	clickWAV := synthesizeMechanicalSoundWAV(prof, SoundEventButtonClick, vol)

	sm.cachedMemory["tap"] = tapWAV
	sm.cachedMemory["space"] = spaceWAV
	sm.cachedMemory["click"] = clickWAV

	if sm.sfxDir != "" {
		tapPath := filepath.Join(sm.sfxDir, "key_tap.wav")
		spacePath := filepath.Join(sm.sfxDir, "key_space.wav")
		clickPath := filepath.Join(sm.sfxDir, "ui_click.wav")

		if err := os.WriteFile(tapPath, tapWAV, 0644); err == nil {
			sm.cachedFiles["tap"] = tapPath
		}
		if err := os.WriteFile(spacePath, spaceWAV, 0644); err == nil {
			sm.cachedFiles["space"] = spacePath
		}
		if err := os.WriteFile(clickPath, clickWAV, 0644); err == nil {
			sm.cachedFiles["click"] = clickPath
		}
	}
}

// synthesizeMechanicalSoundWAV tạo dữ liệu âm thanh RIFF WAV 16-bit PCM (44100 Hz, Mono) mô phỏng
// tiếng gõ máy chữ cổ điển, phím cơ học trầm hoặc tiếng nhấp nút giao diện.
func synthesizeMechanicalSoundWAV(profile SoundProfile, eventType SoundEventType, volume float64) []byte {
	const sampleRate = 44100

	durationMs := 18.0
	baseFreq := 340.0
	clickFreq := 1950.0
	decayRate := 210.0
	noiseMix := 0.38

	switch profile {
	case SoundProfileMechanical:
		baseFreq = 240.0
		clickFreq = 1320.0
		decayRate = 185.0
		noiseMix = 0.25
		durationMs = 22.0
	case SoundProfileSoftTick:
		baseFreq = 480.0
		clickFreq = 2450.0
		decayRate = 310.0
		noiseMix = 0.12
		durationMs = 13.0
	default: // SoundProfileTypewriter
		baseFreq = 360.0
		clickFreq = 2150.0
		decayRate = 205.0
		noiseMix = 0.42
		durationMs = 19.0
	}

	if eventType == SoundEventKeySpaceOrReturn {
		baseFreq *= 0.72
		clickFreq *= 0.80
		durationMs += 8.0
		decayRate *= 0.82
	} else if eventType == SoundEventButtonClick {
		baseFreq = 520.0
		clickFreq = 2600.0
		durationMs = 14.0
		decayRate = 280.0
		noiseMix = 0.18
	}

	numSamples := int(float64(sampleRate) * (durationMs / 1000.0))
	pcmData := make([]byte, numSamples*2)

	// Bộ tạo nhiễu trắng xác định (deterministic LCG) tạo âm sắc cơ học chân thực
	var lcg uint32 = 0x6d2b79f5

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)

		// Đường bao biên độ giảm nhanh theo hàm mũ (Exponential Envelope)
		envBody := math.Exp(-t * decayRate)
		envTransient := math.Exp(-t * (decayRate * 3.4))

		// Sóng cộng hưởng thân phím + xung kim loại đầu hành trình phím
		bodyWave := math.Sin(2.0 * math.Pi * baseFreq * t)
		clickWave := math.Sin(2.0 * math.Pi * clickFreq * t)

		lcg = lcg*1664525 + 1013904223
		noise := (float64(int32(lcg)) / float64(math.MaxInt32)) * noiseMix

		sample := (0.48*bodyWave*envBody + 0.42*clickWave*envTransient + noise*envTransient) * volume
		if sample > 0.98 {
			sample = 0.98
		} else if sample < -0.98 {
			sample = -0.98
		}

		val := int16(sample * 32767.0)
		binary.LittleEndian.PutUint16(pcmData[i*2:], uint16(val))
	}

	// Đóng gói chuẩn RIFF WAV Header (44 bytes)
	var buf bytes.Buffer
	dataLen := uint32(len(pcmData))
	fileLen := 36 + dataLen

	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, fileLen)
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))             // PCM chunk size
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))              // AudioFormat = 1 (PCM)
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))              // NumChannels = 1 (Mono)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))     // SampleRate = 44100
	_ = binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2))   // ByteRate
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))              // BlockAlign
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))             // BitsPerSample = 16
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, dataLen)
	buf.Write(pcmData)

	return buf.Bytes()
}

// PlayEvent phát âm thanh giao diện tương ứng trên một goroutine độc lập (không bao giờ chặn luồng UI).
func (sm *SoundManager) PlayEvent(eventType SoundEventType) {
	sm.mu.Lock()
	if !sm.enabled {
		sm.mu.Unlock()
		return
	}
	if (eventType == SoundEventKeyTap || eventType == SoundEventKeySpaceOrReturn) && !sm.typingSoundEnabled {
		sm.mu.Unlock()
		return
	}
	if eventType == SoundEventButtonClick && !sm.clickSoundEnabled {
		sm.mu.Unlock()
		return
	}

	now := time.Now()
	if now.Sub(sm.lastPlayedAt) < sm.minInterval {
		sm.mu.Unlock()
		return
	}
	sm.lastPlayedAt = now

	key := "tap"
	switch eventType {
	case SoundEventKeySpaceOrReturn:
		key = "space"
	case SoundEventButtonClick:
		key = "click"
	}

	wavBytes := sm.cachedMemory[key]
	wavFile := sm.cachedFiles[key]
	customFn := sm.customPlayer
	cmdPath := sm.playerCmd
	cmdArgs := append([]string{}, sm.playerArgs...)
	sm.mu.Unlock()

	go func() {
		if customFn != nil && len(wavBytes) > 0 {
			_ = customFn(wavBytes)
			return
		}
		if cmdPath == "" || wavFile == "" {
			return
		}
		if runtime.GOOS == "windows" {
			psScript := `(New-Object Media.SoundPlayer '` + wavFile + `').PlaySync()`
			cmd := exec.Command(cmdPath, "-NoProfile", "-NonInteractive", "-Command", psScript)
			_ = cmd.Run()
			return
		}
		args := append(cmdArgs, wavFile)
		cmd := exec.Command(cmdPath, args...)
		_ = cmd.Run()
	}()
}

// PlayTypingSound phát âm thanh gõ phím máy chữ/phím cơ khi người dùng nhập văn bản trong khung soạn thảo.
func PlayTypingSound(r rune, keyEv *fyne.KeyEvent) {
	if keyEv != nil {
		switch keyEv.Name {
		case fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace, fyne.KeyBackspace:
			GlobalSoundManager.PlayEvent(SoundEventKeySpaceOrReturn)
		}
		return
	}
	if r == ' ' || r == '\n' || r == '\t' {
		GlobalSoundManager.PlayEvent(SoundEventKeySpaceOrReturn)
		return
	}
	GlobalSoundManager.PlayEvent(SoundEventKeyTap)
}

// PlayUIClickSound phát âm thanh nhấp nút nhẹ nhàng khi người dùng tương tác với nút bấm hoặc bảng chọn.
func PlayUIClickSound() {
	GlobalSoundManager.PlayEvent(SoundEventButtonClick)
}
