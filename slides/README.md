# Trindex Presentation Slides

This folder contains presentation materials for explaining the Trindex project.

## Files

### 1. `presentation.md`
A markdown version of the slides suitable for:
- Viewing in any markdown viewer
- Converting to PDF via pandoc
- Importing into other presentation tools

### 2. `index.html`
A reveal.js-based HTML presentation with:
- Dracula color theme
- Syntax highlighting
- Keyboard navigation (arrow keys, space, enter)
- Responsive design

### 3. `speaker-script.md`
Detailed speaker notes including:
- What to say for each slide
- Timing guidelines
- Key points to emphasize
- Potential Q&A responses

## How to Present

### Option 1: HTML Slides (Recommended)
```bash
# Open in browser
open slides/index.html

# Or serve via Python
python3 -m http.server 8000
# Then open http://localhost:8000/slides/
```

Features:
- Arrow keys or space to navigate
- `f` for fullscreen
- `s` for speaker notes (if added)
- Works offline after first load

### Option 2: Markdown
```bash
# Convert to PDF with pandoc
pandoc slides/presentation.md -o trindex-slides.pdf

# Or view in any markdown previewer
```

## Presentation Structure

1. **Title** (30 sec) - Introduction
2. **What is Trindex?** (2 min) - Overview of features
3. **The Problem** (2 min) - Why this exists
4. **Key Features** (2 min) - MCP-native, semantic search, namespaces
5. **Architecture** (2 min) - Technical overview
6. **CLI Redesign** (3 min) - Major recent improvement
7. **CLI Demo** (3 min) - Commands and usage
8. **Technical Stack** (1 min) - Technologies used
9. **Database Schema** (1 min) - Data model
10. **Why Trindex?** (1 min) - Benefits for different audiences
11. **Roadmap** (1 min) - Future plans
12. **Getting Started** (1 min) - Quick start guide
13. **Demo** (3-5 min) - Live demonstration
14. **Questions** (2-5 min) - Q&A
15. **Thank You** (30 sec) - Closing

**Total time: ~25-30 minutes**

## Customization

### Changing the Theme
Edit `index.html` and change the reveal.js theme in the CSS import:
- `dracula.css` (current)
- `black.css`
- `white.css`
- `league.css`
- `sky.css`
- etc.

### Adding Speaker Notes
Add notes to any slide in `index.html`:
```html
<aside class="notes">
    Your speaker notes here
</aside>
```

Then press `s` during presentation to open speaker view.

## Tips

1. **Practice the CLI demo** - Have a working Trindex instance ready
2. **Emphasize the CLI redesign** - This is the recent major improvement
3. **Show the Web UI** - People love seeing the actual interface
4. **Have the repo ready** - Show the code if technical questions arise

## Text-to-Speech (TTS) Integration

The slides include several TTS options:

### Quick Start: Web Speech API (Easiest)
Add narration to your presentation with one line:

```html
<!-- Add to index.html before closing </body> -->
<script src="tts-narrator.js"></script>
```

Then open in browser and click the 🔊 TTS button. Press 'T' to toggle.

### Generate Video with TTS
Create a complete video with narration:

```bash
cd slides
python3 generate-video.py
# Output: trindex-presentation.mp4
```

See `TTS-GUIDE.md` for detailed options including:
- Cloud TTS (OpenAI, ElevenLabs) for professional quality
- Browser-based narration for live presentations
- Export to YouTube with automated narration

## Additional Files

### 4. `TTS-GUIDE.md`
Complete guide for 5 different TTS integration methods:
1. reveal.js Audio Plugin (offline, pre-recorded)
2. Browser Web Speech API (live, free)
3. Python Video Generator (YouTube export)
4. ffmpeg + Cloud TTS (professional quality)
5. Descript (video editing with Overdub)

### 5. `tts-narrator.js`
Ready-to-use Web Speech API integration. Just include in HTML.

### 6. `generate-video.py`
Python script to generate complete video with TTS narration.

### 7. `generate-pdf.sh`
Convert markdown slides to PDF using pandoc.

## Resources

- **GitHub**: https://github.com/dbehnke/trindex
- **MCP Spec**: https://modelcontextprotocol.io
- **reveal.js docs**: https://revealjs.com
