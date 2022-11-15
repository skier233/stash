package manager

import (
	"fmt"
	"net/url"

	"github.com/stashapp/stash/internal/manager/config"
	"github.com/stashapp/stash/pkg/ffmpeg"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/fsutil"
	"github.com/stashapp/stash/pkg/models"
)

func GetVideoFileContainer(file *file.VideoFile) (ffmpeg.Container, error) {
	var container ffmpeg.Container
	format := file.Format
	if format != "" {
		container = ffmpeg.Container(format)
	} else { // container isn't in the DB
		// shouldn't happen, fallback to ffprobe
		ffprobe := GetInstance().FFProbe
		tmpVideoFile, err := ffprobe.NewVideoFile(file.Path)
		if err != nil {
			return ffmpeg.Container(""), fmt.Errorf("error reading video file: %v", err)
		}

		return ffmpeg.MatchContainer(tmpVideoFile.Container, file.Path)
	}

	return container, nil
}

func includeSceneStreamPath(f *file.VideoFile, streamingResolution models.StreamingResolutionEnum, maxStreamingTranscodeSize models.StreamingResolutionEnum) bool {
	// convert StreamingResolutionEnum to ResolutionEnum so we can get the min
	// resolution
	convertedRes := models.ResolutionEnum(streamingResolution)

	minResolution := convertedRes.GetMinResolution()
	sceneResolution := f.GetMinResolution()

	// don't include if scene resolution is smaller than the streamingResolution
	if sceneResolution != 0 && sceneResolution < minResolution {
		return false
	}

	// if we always allow everything, then return true
	if maxStreamingTranscodeSize == models.StreamingResolutionEnumOriginal {
		return true
	}

	// convert StreamingResolutionEnum to ResolutionEnum
	maxStreamingResolution := models.ResolutionEnum(maxStreamingTranscodeSize)
	return maxStreamingResolution.GetMinResolution() >= minResolution
}

type SceneStreamEndpoint struct {
	URL      string  `json:"url"`
	MimeType *string `json:"mime_type"`
	Label    *string `json:"label"`
}

func makeStreamEndpoint(streamURL *url.URL, streamingResolution models.StreamingResolutionEnum, mimeType, label string) *SceneStreamEndpoint {
	urlCopy := *streamURL
	v := urlCopy.Query()
	v.Set("resolution", streamingResolution.String())
	urlCopy.RawQuery = v.Encode()

	return &SceneStreamEndpoint{
		URL:      urlCopy.String(),
		MimeType: &mimeType,
		Label:    &label,
	}
}

func GetSceneStreamPaths(scene *models.Scene, directStreamURL *url.URL, maxStreamingTranscodeSize models.StreamingResolutionEnum) ([]*SceneStreamEndpoint, error) {
	if scene == nil {
		return nil, fmt.Errorf("nil scene")
	}

	pf := scene.Files.Primary()
	if pf == nil {
		return nil, nil
	}

	var ret []*SceneStreamEndpoint
	mimeWebm := ffmpeg.MimeWebm
	mimeHLS := ffmpeg.MimeHLS
	mimeMp4 := ffmpeg.MimeMp4

	labelWebm := "webm"
	labelHLS := "HLS"

	// direct stream should only apply when the audio codec is supported
	audioCodec := ffmpeg.MissingUnsupported
	if pf.AudioCodec != "" {
		audioCodec = ffmpeg.ProbeAudioCodec(pf.AudioCodec)
	}

	// don't care if we can't get the container
	container, _ := GetVideoFileContainer(pf)

	replaceSuffix := func(suffix string) *url.URL {
		urlCopy := *directStreamURL
		urlCopy.Path += suffix
		return &urlCopy
	}

	if HasTranscode(scene, config.GetInstance().GetVideoFileNamingAlgorithm()) || ffmpeg.IsValidAudioForContainer(audioCodec, container) {
		label := "Direct stream"
		ret = append(ret, &SceneStreamEndpoint{
			URL:      directStreamURL.String(),
			MimeType: &mimeMp4,
			Label:    &label,
		})
	}

	// only add mkv stream endpoint if the scene container is an mkv already
	if container == ffmpeg.Matroska {
		label := "mkv"
		ret = append(ret, &SceneStreamEndpoint{
			URL: replaceSuffix(".mkv").String(),
			// set mkv to mp4 to trick the client, since many clients won't try mkv
			MimeType: &mimeMp4,
			Label:    &label,
		})
	}

	// WEBM quality transcoding options
	// Note: These have the wrong mime type intentionally to allow jwplayer to selection between mp4/webm
	webmLabelFourK := "WEBM 4K (2160p)"         // "FOUR_K"
	webmLabelFullHD := "WEBM Full HD (1080p)"   // "FULL_HD"
	webmLabelStandardHD := "WEBM HD (720p)"     // "STANDARD_HD"
	webmLabelStandard := "WEBM Standard (480p)" // "STANDARD"
	webmLabelLow := "WEBM Low (240p)"           // "LOW"

	// Setup up lower quality transcoding options (MP4)
	mp4LabelFourK := "MP4 4K (2160p)"         // "FOUR_K"
	mp4LabelFullHD := "MP4 Full HD (1080p)"   // "FULL_HD"
	mp4LabelStandardHD := "MP4 HD (720p)"     // "STANDARD_HD"
	mp4LabelStandard := "MP4 Standard (480p)" // "STANDARD"
	mp4LabelLow := "MP4 Low (240p)"           // "LOW"

	var webmStreams []*SceneStreamEndpoint
	var mp4Streams []*SceneStreamEndpoint

	webmURL := replaceSuffix(".webm")
	mp4URL := replaceSuffix(".mp4")

	if includeSceneStreamPath(pf, models.StreamingResolutionEnumFourK, maxStreamingTranscodeSize) {
		webmStreams = append(webmStreams, makeStreamEndpoint(webmURL, models.StreamingResolutionEnumFourK, mimeMp4, webmLabelFourK))
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4URL, models.StreamingResolutionEnumFourK, mimeMp4, mp4LabelFourK))
	}

	if includeSceneStreamPath(pf, models.StreamingResolutionEnumFullHd, maxStreamingTranscodeSize) {
		webmStreams = append(webmStreams, makeStreamEndpoint(webmURL, models.StreamingResolutionEnumFullHd, mimeMp4, webmLabelFullHD))
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4URL, models.StreamingResolutionEnumFullHd, mimeMp4, mp4LabelFullHD))
	}

	if includeSceneStreamPath(pf, models.StreamingResolutionEnumStandardHd, maxStreamingTranscodeSize) {
		webmStreams = append(webmStreams, makeStreamEndpoint(webmURL, models.StreamingResolutionEnumStandardHd, mimeMp4, webmLabelStandardHD))
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4URL, models.StreamingResolutionEnumStandardHd, mimeMp4, mp4LabelStandardHD))
	}

	if includeSceneStreamPath(pf, models.StreamingResolutionEnumStandard, maxStreamingTranscodeSize) {
		webmStreams = append(webmStreams, makeStreamEndpoint(webmURL, models.StreamingResolutionEnumStandard, mimeMp4, webmLabelStandard))
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4URL, models.StreamingResolutionEnumStandard, mimeMp4, mp4LabelStandard))
	}

	if includeSceneStreamPath(pf, models.StreamingResolutionEnumLow, maxStreamingTranscodeSize) {
		webmStreams = append(webmStreams, makeStreamEndpoint(webmURL, models.StreamingResolutionEnumLow, mimeMp4, webmLabelLow))
		mp4Streams = append(mp4Streams, makeStreamEndpoint(mp4URL, models.StreamingResolutionEnumLow, mimeMp4, mp4LabelLow))
	}

	ret = append(ret, webmStreams...)
	ret = append(ret, mp4Streams...)

	defaultStreams := []*SceneStreamEndpoint{
		{
			URL:      replaceSuffix(".webm").String(),
			MimeType: &mimeWebm,
			Label:    &labelWebm,
		},
	}

	ret = append(ret, defaultStreams...)

	hls := SceneStreamEndpoint{
		URL:      replaceSuffix(".m3u8").String(),
		MimeType: &mimeHLS,
		Label:    &labelHLS,
	}
	ret = append(ret, &hls)

	return ret, nil
}

// HasTranscode returns true if a transcoded video exists for the provided
// scene. It will check using the OSHash of the scene first, then fall back
// to the checksum.
func HasTranscode(scene *models.Scene, fileNamingAlgo models.HashAlgorithm) bool {
	if scene == nil {
		return false
	}

	sceneHash := scene.GetHash(fileNamingAlgo)
	if sceneHash == "" {
		return false
	}

	transcodePath := instance.Paths.Scene.GetTranscodePath(sceneHash)
	ret, _ := fsutil.FileExists(transcodePath)
	return ret
}
