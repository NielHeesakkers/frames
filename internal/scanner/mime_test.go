// internal/scanner/mime_test.go
package scanner

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		name, wantKind, wantMime string
	}{
		{"IMG.JPG", "image", "image/jpeg"},
		{"img.heic", "image", "image/heic"},
		{"clip.MP4", "video", "video/mp4"},
		{"dsc.arw", "raw", "image/x-sony-arw"},
		{"nikon.NEF", "raw", "image/x-nikon-nef"},
		{"canon.cr2", "raw", "image/x-canon-cr2"},
		{"adobe.dng", "raw", "image/x-adobe-dng"},
		{"readme.pdf", "other", "application/pdf"},
		{"song.flac", "other", "audio/flac"},
		{"unknown.xyz", "other", "application/octet-stream"},
	}
	for _, c := range cases {
		k, m := Classify(c.name)
		if k != c.wantKind || m != c.wantMime {
			t.Errorf("%s: got (%s,%s) want (%s,%s)", c.name, k, m, c.wantKind, c.wantMime)
		}
	}
}
