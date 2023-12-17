umls = $(wildcard *.plantuml)
svgs = $(umls:plantuml=svg)
# pngs = $(umls:plantuml=png)
# pdfs = $(umls:plantuml=pdf)

all: ${svgs} #${pdfs} ${pngs}

%.png: %.plantuml
	plantuml -tng $<

%.svg: %.plantuml
	plantuml -tsvg $<

%.pdf: %.plantuml
	plantuml -tpdf $<
