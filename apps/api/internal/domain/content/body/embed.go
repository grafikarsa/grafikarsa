package body

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	youtubeIDPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	vimeoIDPattern     = regexp.MustCompile(`^[0-9]+$`)
	googleIDPattern    = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	figmaKeyPattern    = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	canvaIDPattern     = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	sketchfabSlug      = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	codepenUserPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	codepenPenPattern  = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	sandboxPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	spotifyIDPattern   = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

// ParseEmbed recognizes an allowlisted embed URL and returns its provider plus
// a canonical URL (and video id for YouTube/Vimeo). Unknown hosts, non-HTTPS
// URLs, and unsupported paths return ok=false.
func ParseEmbed(rawURL string) (Embed, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return Embed{}, false
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.User != nil {
		return Embed{}, false
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return Embed{}, false
	}

	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	segments := splitPath(u.Path)

	switch host {
	case "youtu.be":
		return parseYouTube(segments, u.Query())
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtube-nocookie.com", "www.youtube-nocookie.com":
		return parseYouTube(segments, u.Query())
	case "vimeo.com", "www.vimeo.com", "player.vimeo.com":
		return parseVimeo(segments)
	case "figma.com", "www.figma.com":
		return parseFigma(segments)
	case "canva.com", "www.canva.com":
		return parseCanva(segments)
	case "docs.google.com":
		return parseGoogle(segments)
	case "drive.google.com":
		return parseGoogleDrive(segments)
	case "sketchfab.com", "www.sketchfab.com":
		return parseSketchfab(segments)
	case "codepen.io", "www.codepen.io":
		return parseCodePen(segments)
	case "codesandbox.io", "www.codesandbox.io":
		return parseCodeSandbox(segments)
	case "open.spotify.com":
		return parseSpotify(segments)
	default:
		return Embed{}, false
	}
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseYouTube(segments []string, query url.Values) (Embed, bool) {
	var id string
	switch {
	case len(segments) >= 2 && (segments[0] == "embed" || segments[0] == "shorts" || segments[0] == "live" || segments[0] == "v"):
		id = segments[1]
	case len(segments) >= 1 && segments[0] == "watch":
		id = query.Get("v")
	case len(segments) >= 1:
		// youtu.be/<id> and youtube.com/<id> share this branch.
		id = segments[0]
	}
	if !youtubeIDPattern.MatchString(id) {
		return Embed{}, false
	}
	return Embed{
		Provider: "youtube",
		URL:      "https://www.youtube.com/watch?v=" + id,
		VideoID:  id,
	}, true
}

func parseVimeo(segments []string) (Embed, bool) {
	if len(segments) == 0 {
		return Embed{}, false
	}
	id := segments[0]
	if segments[0] == "video" {
		if len(segments) < 2 {
			return Embed{}, false
		}
		id = segments[1]
	}
	if !vimeoIDPattern.MatchString(id) {
		return Embed{}, false
	}
	return Embed{
		Provider: "vimeo",
		URL:      "https://vimeo.com/" + id,
		VideoID:  id,
	}, true
}

func parseFigma(segments []string) (Embed, bool) {
	if len(segments) < 2 {
		return Embed{}, false
	}
	kind := segments[0]
	switch kind {
	case "file", "design", "proto", "board":
	default:
		return Embed{}, false
	}
	key := segments[1]
	if !figmaKeyPattern.MatchString(key) {
		return Embed{}, false
	}
	return Embed{
		Provider: "figma",
		URL:      "https://www.figma.com/" + kind + "/" + key,
	}, true
}

func parseCanva(segments []string) (Embed, bool) {
	if len(segments) < 3 || segments[0] != "design" {
		return Embed{}, false
	}
	if segments[len(segments)-1] != "view" {
		return Embed{}, false
	}
	id := segments[1]
	if !canvaIDPattern.MatchString(id) {
		return Embed{}, false
	}
	return Embed{
		Provider: "canva",
		URL:      "https://www.canva.com/design/" + id + "/view",
	}, true
}

func parseGoogle(segments []string) (Embed, bool) {
	if len(segments) < 3 || segments[1] != "d" {
		return Embed{}, false
	}
	id := segments[2]
	if !googleIDPattern.MatchString(id) {
		return Embed{}, false
	}
	switch segments[0] {
	case "presentation":
		return Embed{
			Provider: "google_slides",
			URL:      "https://docs.google.com/presentation/d/" + id + "/embed",
		}, true
	case "document":
		return Embed{
			Provider: "google_docs",
			URL:      "https://docs.google.com/document/d/" + id + "/preview",
		}, true
	default:
		return Embed{}, false
	}
}

func parseGoogleDrive(segments []string) (Embed, bool) {
	if len(segments) < 3 || segments[0] != "file" || segments[1] != "d" {
		return Embed{}, false
	}
	id := segments[2]
	if !googleIDPattern.MatchString(id) {
		return Embed{}, false
	}
	return Embed{
		Provider: "google_drive",
		URL:      "https://drive.google.com/file/d/" + id + "/preview",
	}, true
}

func parseSketchfab(segments []string) (Embed, bool) {
	if len(segments) < 2 {
		return Embed{}, false
	}
	switch segments[0] {
	case "3d-models", "models":
	default:
		return Embed{}, false
	}
	slug := segments[len(segments)-1]
	if !sketchfabSlug.MatchString(slug) {
		return Embed{}, false
	}
	return Embed{
		Provider: "sketchfab",
		URL:      "https://sketchfab.com/3d-models/" + slug,
	}, true
}

func parseCodePen(segments []string) (Embed, bool) {
	if len(segments) < 3 || (segments[1] != "pen" && segments[1] != "embed") {
		return Embed{}, false
	}
	user, pen := segments[0], segments[2]
	if !codepenUserPattern.MatchString(user) || !codepenPenPattern.MatchString(pen) {
		return Embed{}, false
	}
	return Embed{
		Provider: "codepen",
		URL:      "https://codepen.io/" + user + "/pen/" + pen,
	}, true
}

func parseCodeSandbox(segments []string) (Embed, bool) {
	if len(segments) < 2 {
		return Embed{}, false
	}
	kind := segments[0]
	if kind != "s" && kind != "p" {
		return Embed{}, false
	}
	rest := strings.Join(segments[1:], "/")
	if rest == "" || !sandboxPathPattern.MatchString(rest) {
		return Embed{}, false
	}
	return Embed{
		Provider: "codesandbox",
		URL:      "https://codesandbox.io/" + kind + "/" + rest,
	}, true
}

func parseSpotify(segments []string) (Embed, bool) {
	if len(segments) < 2 {
		return Embed{}, false
	}
	if segments[0] == "embed" {
		segments = segments[1:]
	}
	if len(segments) < 2 {
		return Embed{}, false
	}
	kind := segments[0]
	switch kind {
	case "track", "episode", "show":
	default:
		return Embed{}, false
	}
	id := segments[1]
	if !spotifyIDPattern.MatchString(id) {
		return Embed{}, false
	}
	return Embed{
		Provider: "spotify",
		URL:      "https://open.spotify.com/" + kind + "/" + id,
	}, true
}
