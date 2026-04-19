// internal/upload/safepath_test.go
package upload

import "testing"

func TestSafePath(t *testing.T) {
	tests := []struct {
		root, rel string
		wantErr   bool
	}{
		{"/photos", "2024/a.jpg", false},
		{"/photos", "../etc/passwd", true},
		{"/photos", "2024/../2023/x.jpg", false}, // cleans to 2023/x.jpg, still inside root
		{"/photos", "/absolute", true},
		{"/photos", "2024/./b.jpg", false},
		{"/photos", "", true},
	}
	for _, tc := range tests {
		_, err := SafeJoin(tc.root, tc.rel)
		if (err != nil) != tc.wantErr {
			t.Errorf("%q+%q: err=%v want %v", tc.root, tc.rel, err, tc.wantErr)
		}
	}
}
