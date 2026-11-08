package body

import (
	"testing"
)

func TestParseEmbed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		provider string
		url      string
		videoID  string
	}{
		{
			name:     "figma file",
			raw:      "https://www.figma.com/file/AbC123/My-Design?node-id=1-2",
			provider: "figma",
			url:      "https://www.figma.com/file/AbC123",
		},
		{
			name:     "figma design",
			raw:      "https://figma.com/design/AbC123/Title",
			provider: "figma",
			url:      "https://www.figma.com/design/AbC123",
		},
		{
			name:     "figma proto",
			raw:      "https://www.figma.com/proto/AbC123/Prototype",
			provider: "figma",
			url:      "https://www.figma.com/proto/AbC123",
		},
		{
			name:     "figma board",
			raw:      "https://www.figma.com/board/AbC123/Board",
			provider: "figma",
			url:      "https://www.figma.com/board/AbC123",
		},
		{
			name:     "canva view",
			raw:      "https://www.canva.com/design/DAF123/view?utm_content=abc",
			provider: "canva",
			url:      "https://www.canva.com/design/DAF123/view",
		},
		{
			name:     "canva slugged view",
			raw:      "https://www.canva.com/design/DAF123/My-Design/view",
			provider: "canva",
			url:      "https://www.canva.com/design/DAF123/view",
		},
		{
			name:     "google slides",
			raw:      "https://docs.google.com/presentation/d/abc_123_X/edit#slide=id.p1",
			provider: "google_slides",
			url:      "https://docs.google.com/presentation/d/abc_123_X/embed",
		},
		{
			name:     "google docs",
			raw:      "https://docs.google.com/document/d/abc_123_X/edit?usp=sharing",
			provider: "google_docs",
			url:      "https://docs.google.com/document/d/abc_123_X/preview",
		},
		{
			name:     "google drive",
			raw:      "https://drive.google.com/file/d/abc_123_X/view?usp=sharing",
			provider: "google_drive",
			url:      "https://drive.google.com/file/d/abc_123_X/preview",
		},
		{
			name:     "sketchfab",
			raw:      "https://sketchfab.com/3d-models/my-model-abc123?utm_source=x",
			provider: "sketchfab",
			url:      "https://sketchfab.com/3d-models/my-model-abc123",
		},
		{
			name:     "sketchfab models path",
			raw:      "https://sketchfab.com/models/abc123",
			provider: "sketchfab",
			url:      "https://sketchfab.com/3d-models/abc123",
		},
		{
			name:     "codepen",
			raw:      "https://codepen.io/alice/pen/AbC123",
			provider: "codepen",
			url:      "https://codepen.io/alice/pen/AbC123",
		},
		{
			name:     "codepen embed",
			raw:      "https://codepen.io/alice/embed/AbC123",
			provider: "codepen",
			url:      "https://codepen.io/alice/pen/AbC123",
		},
		{
			name:     "codesandbox s",
			raw:      "https://codesandbox.io/s/abc-123",
			provider: "codesandbox",
			url:      "https://codesandbox.io/s/abc-123",
		},
		{
			name:     "codesandbox p",
			raw:      "https://codesandbox.io/p/sandbox/abc-123",
			provider: "codesandbox",
			url:      "https://codesandbox.io/p/sandbox/abc-123",
		},
		{
			name:     "spotify track",
			raw:      "https://open.spotify.com/track/abc123?si=xyz",
			provider: "spotify",
			url:      "https://open.spotify.com/track/abc123",
		},
		{
			name:     "spotify episode",
			raw:      "https://open.spotify.com/episode/ep123",
			provider: "spotify",
			url:      "https://open.spotify.com/episode/ep123",
		},
		{
			name:     "spotify show",
			raw:      "https://open.spotify.com/show/show123",
			provider: "spotify",
			url:      "https://open.spotify.com/show/show123",
		},
		{
			name:     "spotify embed",
			raw:      "https://open.spotify.com/embed/track/abc123",
			provider: "spotify",
			url:      "https://open.spotify.com/track/abc123",
		},
		{
			name:     "youtube watch",
			raw:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=30",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
		{
			name:     "youtube short",
			raw:      "https://youtube.com/shorts/dQw4w9WgXcQ",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
		{
			name:     "youtube embed",
			raw:      "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
		{
			name:     "youtu.be",
			raw:      "https://youtu.be/dQw4w9WgXcQ",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
		{
			name:     "youtube live",
			raw:      "https://www.youtube.com/live/dQw4w9WgXcQ",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
		{
			name:     "vimeo",
			raw:      "https://vimeo.com/123456789",
			provider: "vimeo",
			url:      "https://vimeo.com/123456789",
			videoID:  "123456789",
		},
		{
			name:     "vimeo video path",
			raw:      "https://vimeo.com/video/123456789",
			provider: "vimeo",
			url:      "https://vimeo.com/123456789",
			videoID:  "123456789",
		},
		{
			name:     "vimeo player",
			raw:      "https://player.vimeo.com/video/123456789",
			provider: "vimeo",
			url:      "https://vimeo.com/123456789",
			videoID:  "123456789",
		},
		{
			name:     "uppercase scheme and host",
			raw:      "HTTPS://WWW.YOUTUBE.COM/watch?v=dQw4w9WgXcQ",
			provider: "youtube",
			url:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			videoID:  "dQw4w9WgXcQ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ParseEmbed(tc.raw)
			if !ok {
				t.Fatalf("ParseEmbed(%q) rejected", tc.raw)
			}
			if got.Provider != tc.provider {
				t.Fatalf("provider = %q, want %q", got.Provider, tc.provider)
			}
			if got.URL != tc.url {
				t.Fatalf("url = %q, want %q", got.URL, tc.url)
			}
			if got.VideoID != tc.videoID {
				t.Fatalf("videoID = %q, want %q", got.VideoID, tc.videoID)
			}
		})
	}
}

func TestParseEmbed_Rejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{"empty", ""},
		{"not a url", "not a url"},
		{"unknown host", "https://evil.com/file/abc"},
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<h1>x</h1>"},
		{"http scheme", "http://www.youtube.com/watch?v=dQw4w9WgXcQ"},
		{"protocol relative", "//www.youtube.com/watch?v=dQw4w9WgXcQ"},
		{"userinfo", "https://user:pass@youtube.com/watch?v=dQw4w9WgXcQ"},
		{"figma unsupported path", "https://www.figma.com/pricing"},
		{"figma missing key", "https://www.figma.com/file/"},
		{"figma invalid key", "https://www.figma.com/file/abc-123"},
		{"canva no view", "https://www.canva.com/design/DAF123"},
		{"canva wrong kind", "https://www.canva.com/folder/DAF123/view"},
		{"slides wrong kind", "https://docs.google.com/spreadsheets/d/abc_123"},
		{"drive missing path", "https://drive.google.com/open?id=abc"},
		{"sketchfab unsupported path", "https://sketchfab.com/features"},
		{"codepen unsupported path", "https://codepen.io/alice/pen"},
		{"codesandbox unsupported kind", "https://codesandbox.io/embed/abc"},
		{"spotify album", "https://open.spotify.com/album/abc123"},
		{"youtube missing v", "https://www.youtube.com/watch"},
		{"youtube bad id", "https://www.youtube.com/watch?v=short"},
		{"youtube bad short", "https://youtube.com/shorts/abc"},
		{"vimeo bad id", "https://vimeo.com/not-a-number"},
		{"vimeo missing id", "https://vimeo.com/"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got, ok := ParseEmbed(tc.raw); ok {
				t.Fatalf("ParseEmbed(%q) accepted: %+v", tc.raw, got)
			}
		})
	}
}
