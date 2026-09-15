package config

import "testing"

func TestParseMemory(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int64
		wantErr bool
	}{
		{name: "plain bytes", in: "512", want: 512},
		{name: "megabytes", in: "256M", want: 256 << 20},
		{name: "megabytes lowercase", in: "256m", want: 256 << 20},
		{name: "gigabytes", in: "1G", want: 1 << 30},
		{name: "gib suffix", in: "2GiB", want: 2 << 30},
		{name: "terabytes", in: "3TB", want: 3 << 40},
		{name: "kilobytes with spaces", in: " 512K ", want: 512 << 10},
		{name: "kib suffix", in: "4KiB", want: 4 << 10},
		{name: "zero", in: "0", want: 0},
		{name: "empty", in: "", wantErr: true},
		{name: "letters only", in: "abc", wantErr: true},
		{name: "fractional", in: "1.5T", wantErr: true},
		{name: "negative", in: "-4G", wantErr: true},
		{name: "unknown suffix", in: "256XB", wantErr: true},
		{name: "suffix only", in: "M", wantErr: true},
		{name: "overflow", in: "99999999999999999999G", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMemory(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseMemory(%q) = %d, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMemory(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseMemory(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
