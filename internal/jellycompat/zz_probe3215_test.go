package jellycompat

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Silo-Server/silo-server/internal/catalog"
	"github.com/Silo-Server/silo-server/internal/config"
	"github.com/Silo-Server/silo-server/internal/models"
)

func TestZZProbe3215(t *testing.T) {
	data, err := os.ReadFile("testdata/streamyfin_mpv_android_device_profile.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile DeviceProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	h264 := catalog.FileVersion{
		FileID: 1, Container: "mkv", CodecVideo: "h264", CodecAudio: "eac3", Resolution: "1080p", Bitrate: 9816,
		VideoTracks: []models.VideoTrack{{Codec: "h264", Profile: "High", Level: 41, Width: 1920, Height: 1080, BitDepth: 8}},
		AudioTracks: []models.AudioTrack{{Codec: "eac3", Channels: 6, Default: true}},
	}
	hevc := catalog.FileVersion{
		FileID: 2, Container: "mkv", CodecVideo: "hevc", CodecAudio: "truehd", Resolution: "2160p", Bitrate: 61173,
		VideoTracks: []models.VideoTrack{{Codec: "hevc", Profile: "Main 10", Level: 153, Width: 3840, Height: 2160, BitDepth: 10, ColorTransfer: "smpte2084"}},
		AudioTracks: []models.AudioTrack{{Codec: "truehd", Channels: 8, Default: true}, {Codec: "eac3", Channels: 6}},
	}
	for _, cfg := range []struct {
		name string
		h    *PlaybackHandler
	}{
		{"zero handler", &PlaybackHandler{codec: NewResourceIDCodec()}},
		{"default config", &PlaybackHandler{codec: NewResourceIDCodec(), PlaybackConfig: func() config.PlaybackConfig { return config.PlaybackConfig{Routing: config.DefaultPlaybackRoutingPolicy()} }}},
	} {
		for _, v := range []struct {
			name    string
			version catalog.FileVersion
		}{{"h264 1080p eac3", h264}, {"hevc 2160p truehd", hevc}} {
			for _, max := range []int64{0, 4_000_000, 8_000_000} {
				req := playbackInfoRequest{AudioStreamIndex: compatIntValueOf(1), MaxStreamingBitrate: max}
				s := cfg.h.buildPlaybackSource("item", "play", v.version, profile, req, true)
				t.Logf("%s | %s | max=%d -> direct=%v stream=%v transcode=%v reasons=%v target=%q hlsRemux=%v",
					cfg.name, v.name, max, s.SupportsDirectPlay, s.SupportsDirectStream, s.SupportsTranscoding,
					compatTranscodeReasons(s), compatSourceTargetVideoCodec(s), s.HLSRemux)
			}
		}
	}
}
