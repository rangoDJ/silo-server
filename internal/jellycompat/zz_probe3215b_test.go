package jellycompat

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Silo-Server/silo-server/internal/catalog"
	"github.com/Silo-Server/silo-server/internal/config"
	"github.com/Silo-Server/silo-server/internal/models"
)

// Streamyfin 0.55.1 buildMpvSubtitleProfiles: image formats Embed+Encode,
// text formats Embed (+External except the webvtt alias).
func streamyfinMPVSubtitleProfiles() []SubtitleProfile {
	var out []SubtitleProfile
	for _, f := range []string{"dvdsub", "idx", "pgs", "pgssub", "teletext", "vobsub", "xsub"} {
		out = append(out, SubtitleProfile{Format: f, Method: "Embed"}, SubtitleProfile{Format: f, Method: "Encode"})
	}
	for _, f := range []string{"webvtt", "vtt", "srt", "subrip", "ttml", "ass", "ssa", "microdvd", "mov_text", "mpl2", "pjs", "realtext", "scc", "smi", "stl", "sub", "subviewer", "text", "vplayer"} {
		out = append(out, SubtitleProfile{Format: f, Method: "Embed"})
		if f != "webvtt" {
			out = append(out, SubtitleProfile{Format: f, Method: "External"})
		}
	}
	return out
}

func TestZZProbe3215b(t *testing.T) {
	data, err := os.ReadFile("testdata/streamyfin_mpv_android_device_profile.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile DeviceProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	profile.SubtitleProfiles = streamyfinMPVSubtitleProfiles()

	titleD := catalog.FileVersion{
		FileID: 4, Container: "mkv", CodecVideo: "h264", CodecAudio: "eac3", Resolution: "1080p", Bitrate: 9816,
		VideoTracks:    []models.VideoTrack{{Codec: "h264", Profile: "High", Level: 41, Width: 1920, Height: 1080, BitDepth: 8}},
		AudioTracks:    []models.AudioTrack{{Codec: "eac3", Channels: 6, Default: true}},
		SubtitleTracks: []catalog.VersionSubtitleTrack{{Index: 2, Codec: "subrip", Language: "eng"}},
	}
	titleDExternal := titleD
	titleDExternal.SubtitleTracks = []catalog.VersionSubtitleTrack{{Index: 2, Codec: "srt", Language: "eng", External: true, FileName: "D.en.srt"}}
	titleA := catalog.FileVersion{
		FileID: 1, Container: "mkv", CodecVideo: "hevc", CodecAudio: "truehd", Resolution: "2160p", Bitrate: 61173, HDR: true,
		VideoTracks:    []models.VideoTrack{{Codec: "hevc", Profile: "Main 10", Level: 153, Width: 3840, Height: 2160, BitDepth: 10, ColorTransfer: "smpte2084", ColorPrimaries: "bt2020"}},
		AudioTracks:    []models.AudioTrack{{Codec: "truehd", Channels: 8, Default: true}, {Codec: "eac3", Channels: 6}, {Codec: "ac3", Channels: 2}},
		SubtitleTracks: []catalog.VersionSubtitleTrack{{Index: 4, Codec: "hdmv_pgs_subtitle", Language: "eng"}, {Index: 5, Codec: "subrip", Language: "eng"}},
	}
	h := &PlaybackHandler{codec: NewResourceIDCodec(), PlaybackConfig: func() config.PlaybackConfig {
		return config.PlaybackConfig{Routing: config.DefaultPlaybackRoutingPolicy()}
	}}
	for _, v := range []struct {
		name    string
		version catalog.FileVersion
		subs    []int
	}{{"D embedded srt", titleD, []int{-1, 2}}, {"D external srt", titleDExternal, []int{-1, 2}}, {"A pgs+srt", titleA, []int{-1, 4, 5}}} {
		for _, audio := range []int{0, 1} {
			for _, sub := range v.subs {
				for _, max := range []int64{0, 4_000_000} {
					req := playbackInfoRequest{AudioStreamIndex: compatIntValueOf(audio), SubtitleStreamIndex: compatIntValueOf(sub), MaxStreamingBitrate: max}
					s := h.buildPlaybackSource("item", "play", v.version, profile, req, true, true)
					s.SelectedSubtitleStreamIndex = resolveSelectedSubtitleStreamIndex(s.Version, 0, true, intPtr(sub), s.DefaultSubtitleStreamIndex)
					afterBuild := [3]bool{s.SupportsDirectPlay, s.SupportsDirectStream, s.SupportsTranscoding}
					applyCompatSubtitleDelivery(&s, profile, false)
					applyCompatDownloadedSubtitleDelivery(&s, profile, nil)
					t.Logf("%s | audio=%d sub=%d max=%d | build d/s/t=%v -> after subs d/s/t=[%v %v %v] burn=%v canBurn=%v",
						v.name, audio, sub, max, afterBuild, s.SupportsDirectPlay, s.SupportsDirectStream, s.SupportsTranscoding, s.SubtitleBurnIn, s.CanBurnSubtitle)
				}
			}
		}
	}
}
