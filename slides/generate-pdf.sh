#!/bin/bash
# Generate PDF from presentation markdown

if ! command -v pandoc &> /dev/null; then
    echo "pandoc not found. Install with:"
    echo "  brew install pandoc"
    echo "  or visit https://pandoc.org/installing.html"
    exit 1
fi

if ! command -v xelatex &> /dev/null; then
    echo "LaTeX not found. Install with:"
    echo "  brew install --cask mactex"
    echo "  or visit https://www.latex-project.org/get/"
    exit 1
fi

echo "Generating PDF..."

pandoc presentation.md \
    -o trindex-presentation.pdf \
    --pdf-engine=xelatex \
    -V geometry:margin=1in \
    -V fontsize=11pt \
    -V colorlinks=true \
    -V linkcolor=blue \
    -V urlcolor=blue \
    -V toccolor=blue \
    --toc \
    --toc-depth=2 \
    -f markdown+autolink_bare_uris

echo "PDF generated: trindex-presentation.pdf"