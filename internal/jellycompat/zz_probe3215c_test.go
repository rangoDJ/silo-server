package jellycompat

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Silo-Server/silo-server/internal/catalog"
	"github.com/Silo-Server/silo-server/internal/config"
	"github.com/Silo-Server/silo-server/internal/models"
)

func TestZZProbe3215c(t *testing.T) {
	for _, file := range []string{"streamyfin_mpv_android_device_profile.json", "streamyfin_mpv_android_av1_device_profile.json"} {
		data, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		var profile DeviceProfile
		if err := json.Unmarshal(data, &profile); err != nil {
			t.Fatal(err)
		}
		titleD := catalog.FileVersion{
			FileID: 4, Container: "mkv", CodecVideo: "h264", CodecAudio: "eac3", Resolution: "1080p", Bitrate: 9816,
			VideoTracks: []models.VideoTrack{{Codec: "h264", Profile: "High", Level: 41, Width: 1920, Height: 1080, BitDepth: 8}},
			AudioTracks: []models.AudioTrack{{Codec: "eac3", Channels: 6, Default: true}},
		}
		h := &PlaybackHandler{codec: NewResourceIDCodec(), PlaybackConfig: func() config.PlaybackConfig {
			return config.PlaybackConfig{Routing: config.DefaultPlaybackRoutingPolicy()}
		}}
		for _, hevc := range []bool{true, false} {
			for _, max := range []int64{0, 4_000_000} {
				req := playbackInfoRequest{AudioStreamIndex: compatIntValueOf(1), MaxStreamingBitrate: max}
				s := h.buildPlaybackSource("item", "play", titleD, profile, req, true, hevc)
				t.Logf("%s | hevcEncoding=%v max=%d -> direct=%v stream=%v transcode=%v target=%q hlsRemux=%v",
					file, hevc, max, s.SupportsDirectPlay, s.SupportsDirectStream, s.SupportsTranscoding, compatSourceTargetVideoCodec(s), s.HLSRemux)
			}
		}
	}
}
