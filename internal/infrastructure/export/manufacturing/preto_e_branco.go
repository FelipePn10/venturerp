package manufacturing

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg" // decodifica logo JPEG
	"image/png"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit"
)

// A ordem de produção é impressa em preto e branco: vai para impressora de
// chão de fábrica, é copiada e rabiscada. Cor da marca e logo colorida somem;
// os cinzas claros ficam só como fundo de cabeçalho e campo.
func temaPretoEBranco() pdfkit.Theme {
	th := pdfkit.DefaultTheme()
	th.Brand = pdfkit.Black
	th.BrandText = pdfkit.White
	th.Title = pdfkit.Black
	th.Text = pdfkit.Black
	th.Muted = pdfkit.Color{R: 80, G: 80, B: 80}
	th.Zebra = pdfkit.Color{R: 242, G: 242, B: 242}
	th.Rule = pdfkit.Color{R: 160, G: 160, B: 160}
	th.TotalBg = pdfkit.Color{R: 235, G: 235, B: 235}
	return th
}

// logoEmCinza converte a logo para tons de cinza mantendo a transparência.
// Imagem que não decodifica volta como veio (o PDF a ignora se for inválida).
func logoEmCinza(dados []byte) []byte {
	if len(dados) == 0 {
		return nil
	}
	src, _, err := image.Decode(bytes.NewReader(dados))
	if err != nil {
		return dados
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			l := uint8((299*uint32(c.R) + 587*uint32(c.G) + 114*uint32(c.B)) / 1000)
			dst.SetNRGBA(x, y, color.NRGBA{R: l, G: l, B: l, A: c.A})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return dados
	}
	return out.Bytes()
}
