package storage

import "testing"

func TestWithPathPrefix(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		raw    string
		want   string
	}{
		{
			name:   "empty prefix returns raw",
			prefix: "",
			raw:    "https://example.com/grafikarsa/a.jpg?X-Amz-Expires=900",
			want:   "https://example.com/grafikarsa/a.jpg?X-Amz-Expires=900",
		},
		{
			name:   "prefix inserted before bucket",
			prefix: "/storage",
			raw:    "https://example.com/grafikarsa/a.jpg?X-Amz-Expires=900",
			want:   "https://example.com/storage/grafikarsa/a.jpg?X-Amz-Expires=900",
		},
		{
			name:   "prefix without leading slash normalized",
			prefix: "storage",
			raw:    "https://example.com/grafikarsa/a.jpg",
			want:   "https://example.com/storage/grafikarsa/a.jpg",
		},
		{
			name:   "already prefixed not doubled",
			prefix: "/storage",
			raw:    "https://example.com/storage/grafikarsa/a.jpg?x=1",
			want:   "https://example.com/storage/grafikarsa/a.jpg?x=1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MinIOClient{presignPathPrefix: tc.prefix}
			if got := m.withPathPrefix(tc.raw); got != tc.want {
				t.Fatalf("withPathPrefix(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
