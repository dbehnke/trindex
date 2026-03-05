# TTS Integration Options for Trindex Slides

This guide explains how to add Text-to-Speech (TTS) narration to your presentation.

## Quick Comparison

| Option | Difficulty | Quality | Best For | Cost |
|--------|-----------|---------|----------|------|
| **1. reveal.js Audio Plugin** | Easy | Medium | Self-running web presentation | Free |
| **2. Browser Web Speech API** | Easy | Medium | Live presentation aid | Free |
| **3. Python Video Generator** | Medium | High | YouTube/video export | Free/Paid |
| **4. ffmpeg + TTS** | Medium | High | Automated video pipeline | Free |
| **5. Descript/Descript-like** | Easy | Very High | Professional video editing | Paid |

---

## Option 1: reveal.js Audio Plugin (Recommended for Web)

Add automatic narration to your HTML slides.

### Setup

1. Download the reveal.js audio plugin:
```bash
cd slides
git clone https://github.com/rajgoel/reveal.js-plugins.git audio-plugins
```

2. Add to your `index.html`:

```html
<!-- In the <head> section -->
<link rel="stylesheet" href="audio-plugins/audio-slideshow/audio-slideshow.css">

<!-- Before closing </body> -->
<script src="audio-plugins/audio-slideshow/audio-slideshow.js"></script>
<script src="audio-plugins/audio-slideshow/recorder.js"></script>
<script>
Reveal.initialize({
    // ... existing config ...
    audio: {
        prefix: 'audio/',
        suffix: '.ogg',
        defaultDuration: 5,
        defaultAudios: true,
        playerOpacity: 0.5,
        playerStyle: 'position: fixed; bottom: 90px; right: 10px;'
    },
    plugins: [RevealHighlight, RevealAudioSlideshow]
});
</script>
```

3. Create audio files from your script:

```bash
# Using macOS say command (free, built-in)
mkdir -p slides/audio
say -v Samantha -o slides/audio/slide-01.aiff "Trindex: Persistent Semantic Memory for AI Agents"
say -v Samantha -o slides/audio/slide-02.aiff "What is Trindex? Trindex is a standalone Go binary..."
# ... etc for each slide

# Convert to ogg
for f in slides/audio/*.aiff; do
    ffmpeg -i "$f" "${f%.aiff}.ogg"
done
```

4. Add data-audio-src attributes to slides:

```html
<section data-audio-src="audio/slide-01.ogg">
    <h1>Trindex</h1>
    <p>Persistent Semantic Memory for AI Agents</p>
</section>
```

### Pros
- Native reveal.js integration
- Auto-advances slides with audio
- Play/pause controls
- Works offline

### Cons
- Need to generate audio files manually
- Audio files increase repo size

---

## Option 2: Browser Web Speech API (Live Presentation Aid)

Use browser's built-in TTS to read speaker notes during live presentation.

### Setup

Add this script to `index.html`:

```html
<script>
// Text-to-Speech helper for reveal.js
class SlideNarrator {
    constructor() {
        this.synth = window.speechSynthesis;
        this.utterance = null;
        this.isEnabled = false;
        
        // Define narration for each slide index
        this.scripts = {
            0: "Trindex: Persistent Semantic Memory for AI Agents",
            1: "What is Trindex? Trindex is a standalone Go binary that provides persistent semantic memory for AI agents via MCP.",
            2: "The problem: AI agents have no memory. Every conversation starts fresh.",
            // ... add all 15 slides
        };
        
        // Listen for slide changes
        Reveal.on('slidechanged', (event) => {
            if (this.isEnabled && this.scripts[event.indexh]) {
                this.speak(this.scripts[event.indexh]);
            }
        });
        
        // Add toggle button
        this.addToggleButton();
    }
    
    speak(text) {
        // Cancel current speech
        this.synth.cancel();
        
        this.utterance = new SpeechSynthesisUtterance(text);
        this.utterance.rate = 0.9; // Slightly slower
        this.utterance.pitch = 1.0;
        this.utterance.volume = 1.0;
        
        // Try to use a good voice
        const voices = this.synth.getVoices();
        const preferredVoice = voices.find(v => 
            v.name.includes('Samantha') || 
            v.name.includes('Google US English') ||
            v.name.includes('Microsoft David')
        );
        if (preferredVoice) {
            this.utterance.voice = preferredVoice;
        }
        
        this.synth.speak(this.utterance);
    }
    
    addToggleButton() {
        const btn = document.createElement('button');
        btn.innerHTML = '🔊 TTS';
        btn.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            z-index: 1000;
            padding: 10px 20px;
            background: #bb9af7;
            color: #1a1b26;
            border: none;
            border-radius: 5px;
            cursor: pointer;
            font-family: sans-serif;
            font-weight: bold;
        `;
        btn.onclick = () => {
            this.isEnabled = !this.isEnabled;
            btn.style.background = this.isEnabled ? '#9ece6a' : '#bb9af7';
            btn.innerHTML = this.isEnabled ? '🔊 TTS ON' : '🔊 TTS';
            if (!this.isEnabled) {
                this.synth.cancel();
            }
        };
        document.body.appendChild(btn);
    }
}

// Initialize when reveal is ready
Reveal.on('ready', () => {
    window.narrator = new SlideNarrator();
});
</script>
```

### Usage
1. Open `index.html` in browser
2. Click "🔊 TTS" button to enable
3. Navigate slides - TTS will read narration automatically
4. Press 's' for speaker view to see script while TTS speaks

### Pros
- No audio files needed
- Real-time narration
- Toggle on/off during presentation
- Uses local browser voices (free)

### Cons
- Quality varies by browser/OS
- Requires live browser
- Not suitable for video export

---

## Option 3: Python Video Generator (Best for YouTube)

Generate a complete video with slides and narration.

### Setup

Create `generate-video.py`:

```python
#!/usr/bin/env python3
"""Generate video from slides with TTS narration."""

import os
import subprocess
import tempfile
from pathlib import Path

# Configuration
SLIDES_DIR = Path("slides")
OUTPUT_FILE = "trindex-presentation.mp4"
SLIDE_DURATION = 30  # seconds per slide (adjust based on your script)

# Speaker script extracted from speaker-script.md
SCRIPTS = {
    1: "Trindex: Persistent Semantic Memory for AI Agents.",
    2: "Trindex uses a client server MCP model: trindex mcp runs as a thin proxy client, and trindex server runs the shared backend on port 9636.",
    3: "The server provides MCP-over-HTTP endpoints plus REST API, backed by PostgreSQL with pgvector for hybrid semantic search.",
    # Add remaining slides...
}

def text_to_speech(text, output_path):
    """Convert text to speech using macOS say or gTTS."""
    
    # Option A: macOS say (free, built-in)
    if os.system("which say > /dev/null 2>&1") == 0:
        aiff_path = output_path.with_suffix('.aiff')
        subprocess.run(['say', '-v', 'Samantha', '-o', str(aiff_path), text], check=True)
        subprocess.run(['ffmpeg', '-y', '-i', str(aiff_path), '-c:a', 'libmp3lame', 
                       '-q:a', '2', str(output_path)], check=True, 
                       capture_output=True)
        aiff_path.unlink()
        return
    
    # Option B: gTTS (Google TTS, requires internet)
    try:
        from gtts import gTTS
        tts = gTTS(text=text, lang='en', slow=False)
        tts.save(str(output_path))
        return
    except ImportError:
        pass
    
    # Option C: pyttsx3 (offline, cross-platform)
    try:
        import pyttsx3
        engine = pyttsx3.init()
        engine.save_to_file(text, str(output_path))
        engine.runAndWait()
        return
    except ImportError:
        pass
    
    raise RuntimeError("No TTS engine available. Install gTTS or pyttsx3.")

def capture_slides():
    """Capture slides as images using playwright."""
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("Installing playwright...")
        subprocess.run(['pip', 'install', 'playwright'], check=True)
        subprocess.run(['playwright', 'install', 'chromium'], check=True)
        from playwright.sync_api import sync_playwright
    
    slide_images = []
    
    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={'width': 1920, 'height': 1080})
        page.goto(f'file://{os.path.abspath("slides/index.html")}')
        
        # Wait for reveal.js to load
        page.wait_for_selector('.reveal')
        
        total_slides = 15  # Update based on your slides
        
        for i in range(total_slides):
            # Navigate to slide
            page.evaluate(f'Reveal.slide({i})')
            page.wait_for_timeout(500)  # Wait for transition
            
            # Capture screenshot
            screenshot_path = f'/tmp/slide_{i+1:02d}.png'
            page.screenshot(path=screenshot_path, full_page=False)
            slide_images.append(screenshot_path)
            
        browser.close()
    
    return slide_images

def create_video(slide_images, audio_files):
    """Combine images and audio into video."""
    
    # Create concat file for ffmpeg
    concat_file = '/tmp/concat.txt'
    with open(concat_file, 'w') as f:
        for img, audio in zip(slide_images, audio_files):
            # Get audio duration
            result = subprocess.run(['ffprobe', '-v', 'error', '-show_entries', 
                                   'format=duration', '-of', 
                                   'default=noprint_wrappers=1:nokey=1', str(audio)],
                                  capture_output=True, text=True)
            duration = float(result.stdout.strip())
            
            f.write(f"file '{img}'\n")
            f.write(f"duration {duration}\n")
        
        # Last image needs to be repeated
        f.write(f"file '{slide_images[-1]}'\n")
    
    # Generate video
    cmd = [
        'ffmpeg', '-y',
        '-f', 'concat',
        '-safe', '0',
        '-i', concat_file,
        '-vf', 'fps=30,format=yuv420p',
        '-c:v', 'libx264',
        '-preset', 'medium',
        '-crf', '23',
        '-pix_fmt', 'yuv420p',
        '-movflags', '+faststart',
        OUTPUT_FILE
    ]
    
    subprocess.run(cmd, check=True)
    print(f"Video created: {OUTPUT_FILE}")

def main():
    print("Generating audio files...")
    audio_files = []
    
    with tempfile.TemporaryDirectory() as tmpdir:
        for slide_num, script in SCRIPTS.items():
            print(f"  Slide {slide_num}...")
            audio_path = Path(tmpdir) / f'slide_{slide_num:02d}.mp3'
            text_to_speech(script, audio_path)
            audio_files.append(audio_path)
        
        print("\nCapturing slides...")
        slide_images = capture_slides()
        
        print("\nCreating video...")
        create_video(slide_images, audio_files)
        
        # Cleanup
        for img in slide_images:
            os.unlink(img)

if __name__ == '__main__':
    main()
```

### Usage

```bash
# Install dependencies
pip install gtts playwright
playwright install chromium

# Generate video
python3 slides/generate-video.py

# Output: trindex-presentation.mp4
```

### Pros
- Full video export (YouTube-ready)
- High quality TTS options
- Automated pipeline
- Can add background music

### Cons
- Requires Python setup
- Generation takes time
- Large output file

---

## Option 4: ffmpeg + Cloud TTS (Best Quality)

Use cloud TTS APIs for professional narration.

### Setup

Create `generate-video-cloud.sh`:

```bash
#!/bin/bash
# Generate video using OpenAI or ElevenLabs TTS

set -e

# Configuration
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
ELEVENLABS_API_KEY="${ELEVENLABS_API_KEY:-}"
OUTPUT="trindex-presentation-cloud.mp4"

# Create temp directory
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# Speaker scripts (from speaker-script.md)
declare -A SCRIPTS=(
    [1]="Trindex: Persistent Semantic Memory for AI Agents"
    [2]="Trindex uses a client server MCP architecture: trindex mcp is the local proxy client, and trindex server is the shared backend at port 9636."
    # ... add all slides
)

echo "Generating audio with OpenAI TTS..."

for i in {1..15}; do
    script="${SCRIPTS[$i]}"
    if [ -n "$script" ]; then
        echo "  Slide $i..."
        
        # OpenAI TTS API
        curl -s -X POST https://api.openai.com/v1/audio/speech \
            -H "Authorization: Bearer $OPENAI_API_KEY" \
            -H "Content-Type: application/json" \
            -d "{
                \"model\": \"tts-1\",
                \"voice\": \"alloy\",
                \"input\": \"$script\"
            }" \
            --output "$TMPDIR/slide_$(printf "%02d" $i).mp3"
    fi
done

echo "Capturing slides with Puppeteer..."

# Create capture script
cat > "$TMPDIR/capture.js" << 'EOF'
const puppeteer = require('puppeteer');
const fs = require('fs');

(async () => {
    const browser = await puppeteer.launch({
        defaultViewport: { width: 1920, height: 1080 }
    });
    const page = await browser.newPage();
    
    await page.goto('file://' + process.cwd() + '/slides/index.html', {
        waitUntil: 'networkidle0'
    });
    
    await page.waitForTimeout(2000);
    
    const totalSlides = 15;
    
    for (let i = 0; i < totalSlides; i++) {
        await page.evaluate((index) => {
            Reveal.slide(index);
        }, i);
        
        await page.waitForTimeout(1000);
        
        await page.screenshot({
            path: `/tmp/slide_${String(i + 1).padStart(2, '0')}.png`,
            type: 'png'
        });
    }
    
    await browser.close();
})();
EOF

node "$TMPDIR/capture.js"

echo "Creating final video..."

# Create ffmpeg concat script
CONCAT_FILE="$TMPDIR/concat.txt"
for i in {1..15}; do
    AUDIO="$TMPDIR/slide_$(printf "%02d" $i).mp3"
    IMG="/tmp/slide_$(printf "%02d" $i).png"
    
    if [ -f "$AUDIO" ]; then
        DURATION=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$AUDIO")
        echo "file '$IMG'" >> "$CONCAT_FILE"
        echo "duration $DURATION" >> "$CONCAT_FILE"
    fi
done

# Build video with audio
ffmpeg -y \
    -f concat -safe 0 -i "$CONCAT_FILE" \
    -f concat -safe 0 -i <(for f in $TMPDIR/slide_*.mp3; do echo "file '$f'"; done) \
    -shortest \
    -c:v libx264 -preset medium -crf 23 -pix_fmt yuv420p \
    -c:a aac -b:a 192k \
    -movflags +faststart \
    "$OUTPUT"

echo "Video created: $OUTPUT"
```

### Usage

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Generate
bash slides/generate-video-cloud.sh
```

### Pros
- Professional TTS quality (OpenAI, ElevenLabs)
- Natural-sounding voices
- Easy voice selection

### Cons
- Requires API key ($)
- API costs (~$1-5 for full presentation)
- Internet required

---

## Option 5: Descript (Easiest Professional)

Use Descript for video editing with built-in Overdub TTS.

### Steps

1. **Export slides as images:**
```bash
# Use the capture script from Option 3
python3 -c "
from playwright.sync_api import sync_playwright
import os

with sync_playwright() as p:
    browser = p.chromium.launch()
    page = browser.new_page(viewport={'width': 1920, 'height': 1080})
    page.goto(f'file://{os.path.abspath(\"slides/index.html\")}')
    page.wait_for_selector('.reveal')
    
    os.makedirs('slide-images', exist_ok=True)
    
    for i in range(15):
        page.evaluate(f'Reveal.slide({i})')
        page.wait_for_timeout(500)
        page.screenshot(path=f'slide-images/slide_{i+1:02d}.png')
    
    browser.close()
"
```

2. **Import to Descript:**
   - Create new project
   - Import slide images
   - Add to timeline

3. **Add narration:**
   - Use Descript's Overdub feature
   - Type your script from speaker-script.md
   - Or record your voice and use Overdub to clone it

4. **Export video**

### Pros
- Professional editing tools
- Overdub voice cloning
- Easy to edit/tweak
- Screen recording capability

### Cons
- $12-24/month subscription
- Requires learning Descript
- Cloud-based

---

## Recommendation

**For live presentations:** Use **Option 2** (Web Speech API) - add the toggle button to your HTML and enable TTS when presenting.

**For YouTube/video:** Use **Option 4** (Cloud TTS) with OpenAI's TTS - best quality, automated.

**For quick sharing:** Use **Option 1** (reveal.js audio) with macOS `say` command - simple, offline.

---

## Quick Start: Web Speech API

The fastest way to add TTS to your existing slides:

1. Add the script from Option 2 to `slides/index.html`
2. Open in browser
3. Click 🔊 TTS button
4. Present!

This gives you immediate narration without generating any files.
