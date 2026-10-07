package sped

// CadastroDoItem é o que o cadastro diz do item para classificá-lo no 0200.
type CadastroDoItem struct {
	TipoEngenharia *int // 0 fabricado, 1 comprado, 2 de terceiro, 3 serviço
	TipoUso        *int // 0 industrialização (padrão), 1 consumo, 2 imobilizado
	Revenda        bool // tipo de venda REVENDA
	Embalagem      bool // item de embalagem
}

// TipoItem (0200/TIPO_ITEM). O 0200 é um por item no arquivo, então a
// classificação vem primeiro do cadastro:
//  1. uso e consumo / imobilizado declarados no item (07 / 08);
//  2. CFOP de entrada conclusivo (revenda 00, consumo 07, ativo 08, serviço 09)
//     — o tipo de uso "industrialização" é o padrão do cadastro e não distingue;
//  3. embalagem 02, serviço 09, fabricado 04 (produto acabado), comprado para
//     revenda 00, comprado 01 (matéria-prima); sem nada, 99.
func TipoItem(c CadastroDoItem, cfopEntrada string) string {
	if c.TipoUso != nil {
		switch *c.TipoUso {
		case 1:
			return "07"
		case 2:
			return "08"
		}
	}
	if len(cfopEntrada) == 4 {
		switch cfopEntrada[1:] {
		case "102", "403", "117", "118", "121":
			return "00"
		case "556", "407", "653":
			return "07"
		case "551", "406", "552", "553", "554", "555":
			return "08"
		case "933", "352", "353", "354", "355", "356", "357", "358", "359", "360":
			return "09"
		}
	}
	if c.Embalagem {
		return "02"
	}
	if c.TipoEngenharia != nil {
		switch *c.TipoEngenharia {
		case 3:
			return "09"
		case 0:
			return "04"
		case 1:
			if c.Revenda {
				return "00"
			}
			return "01"
		}
	}
	if c.Revenda {
		return "00"
	}
	return "99"
}

// CFOPEntrada converte o CFOP do emitente (5xxx/6xxx/7xxx) no de entrada
// (1xxx/2xxx/3xxx); um CFOP que já é de entrada fica como está.
func CFOPEntrada(cfop string) string {
	if len(cfop) != 4 {
		return cfop
	}
	switch cfop[0] {
	case '5':
		return "1" + cfop[1:]
	case '6':
		return "2" + cfop[1:]
	case '7':
		return "3" + cfop[1:]
	}
	return cfop
}

// cstTabelaB são as tributações do ICMS (tabela B do CST).
var cstTabelaB = map[string]bool{"00": true, "02": true, "10": true, "15": true, "20": true, "30": true, "40": true, "41": true,
	"50": true, "51": true, "53": true, "60": true, "61": true, "70": true, "90": true}

// CSTICMS monta o CST de 3 dígitos (origem + tributação). Um código de 3
// dígitos já é origem + CST quando começa pela origem do item e termina numa
// tributação válida ("500" com origem 5 é CST 00; com origem 1 é CSOSN 500).
// O CSOSN do fornecedor do Simples não existe na EFD e vira origem + 90.
func CSTICMS(origem, cst string) string {
	if len(cst) == 3 && (origem == "" || cst[:1] == origem) && cstTabelaB[cst[1:]] {
		return cst
	}
	if origem == "" {
		origem = "0"
	}
	if len(cst) == 2 && cstTabelaB[cst] {
		return origem + cst
	}
	return origem + "90"
}
