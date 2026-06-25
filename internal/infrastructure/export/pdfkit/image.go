package pdfkit

import (
	"bytes"
	"compress/zlib"
	"errors"
	"image/jpeg"
	"image/png"
	"strconv"
)

// image is an embedded raster XObject.
type Image struct {
	alias  string
	w, h   int
	filter string // FlateDecode (PNG) or DCTDecode (JPEG)
	data   []byte
}

// AddImage registers a logo/raster from raw PNG or JPEG bytes and returns a
// handle to draw it with Page.DrawImage. The format is sniffed from the magic
// bytes. JPEG data is embedded verbatim (DCTDecode); PNG is decoded to RGB and
// re-compressed (FlateDecode), with any alpha composited over white so logos on
// transparent backgrounds print cleanly.
func (d *Doc) AddImage(data []byte) (*Image, error) {
	alias := "Im" + strconv.Itoa(len(d.images))
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		img, err := addJPEG(alias, data)
		if err != nil {
			return nil, err
		}
		d.images = append(d.images, img)
		return img, nil
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		img, err := addPNG(alias, data)
		if err != nil {
			return nil, err
		}
		d.images = append(d.images, img)
		return img, nil
	default:
		return nil, errors.New("pdfkit: unsupported image format (want PNG or JPEG)")
	}
}

func addJPEG(alias string, data []byte) (*Image, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &Image{alias: alias, w: cfg.Width, h: cfg.Height, filter: "DCTDecode", data: data}, nil
}

func addPNG(alias string, data []byte) (*Image, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	rgb := make([]byte, 0, w*h*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA() // 16-bit pre-multiplied
			// Composite over white using the alpha.
			rr := compositeWhite(r, a)
			gg := compositeWhite(g, a)
			bb := compositeWhite(bl, a)
			rgb = append(rgb, rr, gg, bb)
		}
	}

	var comp bytes.Buffer
	zw := zlib.NewWriter(&comp)
	if _, err := zw.Write(rgb); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return &Image{alias: alias, w: w, h: h, filter: "FlateDecode", data: comp.Bytes()}, nil
}

// compositeWhite blends a 16-bit channel value over a white background using a
// 16-bit alpha, returning an 8-bit result.
func compositeWhite(c, a uint32) byte {
	// c is already alpha-pre-multiplied (range 0..a). Add the white contribution
	// for the transparent part: out = c + white*(1-alpha).
	const max = 0xFFFF
	out := c + (max - a)
	if out > max {
		out = max
	}
	return byte(out >> 8)
}
