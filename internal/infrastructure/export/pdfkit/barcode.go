package pdfkit

import "fmt"

// Code 128 patterns are expressed as alternating bar/space module widths.
// Values 0..105 have 11 modules; the stop symbol (106) has 13 modules.
var code128Patterns = [...]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213",
	"221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132",
	"221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211",
	"212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331",
	"231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111",
	"314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214",
	"112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141",
	"214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141",
	"114131", "311141", "411131", "211412", "211214", "211232", "2331112",
}

// Code128B returns Code 128 subset B symbols, including start, checksum and stop.
// Subset B covers printable ASCII, which includes the opaque scanner token format.
func Code128B(value string) ([]int, error) {
	if value == "" {
		return nil, fmt.Errorf("valor do codigo de barras obrigatorio")
	}
	symbols := make([]int, 0, len(value)+3)
	symbols = append(symbols, 104) // Start B.
	checksum := 104
	for position, r := range value {
		if r < 32 || r > 126 {
			return nil, fmt.Errorf("codigo de barras aceita somente ASCII imprimivel")
		}
		code := int(r) - 32
		symbols = append(symbols, code)
		checksum += code * (position + 1)
	}
	symbols = append(symbols, checksum%103, 106)
	return symbols, nil
}

// DrawCode128B draws a scanner-friendly vector barcode inside the requested box.
// Quiet zones are included and the bars preserve an integer module ratio.
func (p *Page) DrawCode128B(value string, x, top, w, h float64, color Color) error {
	symbols, err := Code128B(value)
	if err != nil {
		return err
	}
	modules := 20 // ten-module quiet zone on each side.
	for _, symbol := range symbols {
		for _, width := range code128Patterns[symbol] {
			modules += int(width - '0')
		}
	}
	moduleW := w / float64(modules)
	if moduleW < 0.75 {
		return fmt.Errorf("area insuficiente para leitura confiavel do codigo de barras")
	}
	cursor := x + 10*moduleW
	for _, symbol := range symbols {
		pattern := code128Patterns[symbol]
		for i, width := range pattern {
			segmentW := float64(width-'0') * moduleW
			if i%2 == 0 {
				p.FillRect(cursor, top, segmentW, h, color)
			}
			cursor += segmentW
		}
	}
	return nil
}
