package synth

import "testing"

func TestPCM16ToWAV(t *testing.T) {
	samples := []int16{0, 1000, -1000, 0}
	data := PCM16ToWAV(samples, 22050)
	if len(data) <= 44 {
		t.Fatalf("expected wav data > 44 bytes, got %d", len(data))
	}
	if string(data[:4]) != "RIFF" {
		t.Fatalf("expected RIFF header")
	}
	if string(data[8:12]) != "WAVE" {
		t.Fatalf("expected WAVE header")
	}
}
