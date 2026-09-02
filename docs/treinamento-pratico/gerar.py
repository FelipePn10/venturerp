#!/usr/bin/env python3
"""
Gera dia-um.pdf e dia-dois.pdf a partir dos fontes HTML + estilo.css.

Uso:
    pip install weasyprint
    python3 gerar.py            # gera os dois
    python3 gerar.py dia-um     # gera apenas um
"""
import sys
from pathlib import Path

from weasyprint import HTML, CSS

BASE = Path(__file__).resolve().parent

CABECALHO = """<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>{titulo}</title></head><body>
"""

# O rótulo do dia vira o cabeçalho corrente de todas as páginas internas.
RODAPE_DIA = """
@page {{
  @top-left {{
    content: "{dia}";
    font-family: "DejaVu Sans", sans-serif;
    font-size: 7.5pt;
    color: #7c8a80;
    letter-spacing: .04em;
    text-transform: uppercase;
  }}
}}
@page capa {{ @top-left {{ content: none; }} }}
"""

DOCS = {
    "dia-um": ("Treinamento Prático VentureERP — Dia 1",
               "Dia 1 · Do cadastro à saída do acabado"),
    "dia-dois": ("Treinamento Prático VentureERP — Dia 2",
                 "Dia 2 · Fiscal, financeiro, custos e contabilidade"),
}


def gerar(nome: str) -> None:
    titulo, dia = DOCS[nome]
    corpo = (BASE / f"{nome}.html").read_text(encoding="utf-8")
    html = CABECALHO.format(titulo=titulo) + corpo + "\n</body></html>"
    saida = BASE / f"{nome}.pdf"
    HTML(string=html, base_url=str(BASE)).write_pdf(
        saida,
        stylesheets=[
            CSS(filename=str(BASE / "estilo.css")),
            CSS(string=RODAPE_DIA.format(dia=dia)),
        ],
    )
    kb = saida.stat().st_size / 1024
    print(f"  {saida.name}  ({kb:,.0f} KB)")


if __name__ == "__main__":
    alvos = sys.argv[1:] or list(DOCS)
    print("Gerando PDFs:")
    for alvo in alvos:
        if alvo not in DOCS:
            sys.exit(f"documento desconhecido: {alvo} (use {' ou '.join(DOCS)})")
        gerar(alvo)
