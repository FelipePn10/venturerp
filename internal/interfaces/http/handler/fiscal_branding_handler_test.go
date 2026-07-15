package handler

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestValidHexColor(t *testing.T) {
	for _, value := range []string{"#1B5E36", "#abcdef", "#000000"} {
		if !validHexColor(value) {
			t.Fatalf("expected valid color %q", value)
		}
	}
	for _, value := range []string{"1B5E36", "#12345", "#GG0000", "", "#1234567"} {
		if validHexColor(value) {
			t.Fatalf("expected invalid color %q", value)
		}
	}
}

func TestSniffImageMime(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 27, G: 94, B: 54, A: 255})

	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	if mime, ok := sniffImageMime(pngData.Bytes()); !ok || mime != "image/png" {
		t.Fatalf("unexpected PNG result: %q %v", mime, ok)
	}

	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, img, nil); err != nil {
		t.Fatal(err)
	}
	if mime, ok := sniffImageMime(jpegData.Bytes()); !ok || mime != "image/jpeg" {
		t.Fatalf("unexpected JPEG result: %q %v", mime, ok)
	}

	for _, invalid := range [][]byte{[]byte("not an image"), {0x89, 'P', 'N', 'G'}, {0xff, 0xd8, 0xff, 0x00}} {
		if mime, ok := sniffImageMime(invalid); ok {
			t.Fatalf("invalid image accepted as %q", mime)
		}
	}
}
