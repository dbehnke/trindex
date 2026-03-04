#!/usr/bin/env python3
"""
Generate video from slides with TTS narration.
Usage: python3 generate-video.py

Note: This script handles optional imports gracefully.
gTTS, pyttsx3, and playwright are installed automatically if needed.
"""

import os
import subprocess
import tempfile
from pathlib import Path

# Configuration
OUTPUT_FILE = "trindex-presentation.mp4"

# Speaker scripts extracted from speaker-script.md
# Format: {slide_number: "narration text"}
SCRIPTS = {
    1: "Trindex: Persistent Semantic Memory for AI Agents. A standalone Go binary for AI agent memory via the Model Context Protocol.",
    2: "What is Trindex? It stores memories as vector embeddings, retrieves via semantic similarity search, uses hybrid search combining vector and full-text with R-R-F fusion, supports namespace scoping with global fallback, and provides REST API, Web UI, and CLI interfaces.",
    3: "The problem: AI agents have no memory. Every conversation starts fresh. Context windows are limited. There is no standard way to persist knowledge across sessions. Existing solutions like LangChain are Python-only with heavy dependencies. Vector databases are powerful but low level. Trindex is standalone, language-agnostic, and MCP-native.",
    4: "Key features: First, MCP-native integration. Works with Claude Code, opencode, Cursor, and any MCP client. Second, semantic search using cosine similarity via pgvector combined with PostgreSQL full-text search, merged with reciprocal rank fusion. Third, namespace organization with isolated memory spaces for different projects, but global namespace always included.",
    5: "Architecture: MCP clients communicate via standard input-output. Trindex has two modes: MCP mode for agent integration, and server mode for HTTP API and Web UI. Everything stores in PostgreSQL with the pgvector extension. The embedding service is pluggable: Ollama for local development, OpenAI for production, or any OpenAI-compatible endpoint.",
    6: "CLI redesign: Before, Trindex was monolithic. Running the binary started everything at once: MCP server, HTTP server, and database connection. The new CLI uses explicit subcommands: MCP starts just the MCP server. Server starts just the HTTP server. Doctor runs diagnostics. Memories provides CRUD operations. Search performs semantic search from the command line. This enables standalone deployment and scripting.",
    7: "CLI demo: Run doctor to verify configuration, database connectivity, and embedding endpoint. Start the server on a custom port. The memory commands use the REST API under the hood, so they work with local or remote servers via the API URL flag.",
    8: "Technical stack: Go 1.26 plus for the implementation. PostgreSQL 17 with pgvector for storage. HNSW index for fast approximate nearest neighbor search. Hybrid search combining pgvector and tsvector. OpenAI-compatible embedding API. Chi router for HTTP. Vue 3 with Tailwind for the web UI. Testcontainers-go for integration testing.",
    9: "Database schema: The memories table has an ID, namespace, content, vector embedding, JSONB metadata, and a generated tsvector column for full-text search. The key index is the HNSW index on the embedding column using vector cosine operations for fast similarity search.",
    10: "Why Trindex? For AI agent developers, it is drop-in MCP memory with no Python dependencies. For DevOps, it is a single binary with Docker Compose support using existing PostgreSQL infrastructure. For end users, there is a web UI for browsing, CLI for scripting, and import-export for backups with no vendor lock-in.",
    11: "Future roadmap: Phase 3 includes enterprise features like authentication, role-based access control, multi-tenancy, and audit logging. Phase 4 adds advanced search with reranking, query expansion, automatic namespace detection, and memory decay. Phase 5 expands the ecosystem with LangChain integration, Python client, webhooks, and memory sharing between agents.",
    12: "Getting started: Clone the repository, copy the environment example file, and edit it with your embedding endpoint configuration. Then either run with Docker Compose, which is the easiest way, or build locally with Go. The doctor command verifies your setup before starting services.",
    13: "Demo time: Let me show you Trindex in action. First, run diagnostics to verify everything is configured correctly. Then start the server and create some memories via CLI. Next, search for them to show semantic retrieval. Then show the web UI for browsing. Finally, demonstrate export and import for backups.",
    14: "Questions? You can find the code on GitHub at github.com slash dbehncke slash trindex. The documentation includes a complete CLI reference. The MCP specification is available at modelcontextprotocol dot io. The project is under Business Source License 1.1, which means it is free for non-production use.",
    15: "Thank you. Trindex is about giving AI agents persistent memory: one brain for every agent. The goal is to make AI agents truly useful by letting them remember what matters. I hope you will check it out and let me know what you think.",
}


def text_to_speech(text, output_path):
    """Convert text to speech using available TTS engine."""

    # Option A: macOS say (free, built-in, best quality for free)
    if os.system("which say > /dev/null 2>&1") == 0:
        aiff_path = output_path.with_suffix(".aiff")
        try:
            subprocess.run(
                ["say", "-v", "Samantha", "-o", str(aiff_path), text],
                check=True,
                capture_output=True,
            )
            subprocess.run(
                [
                    "ffmpeg",
                    "-y",
                    "-i",
                    str(aiff_path),
                    "-c:a",
                    "libmp3lame",
                    "-q:a",
                    "2",
                    str(output_path),
                ],
                check=True,
                capture_output=True,
            )
            aiff_path.unlink()
            return True
        except subprocess.CalledProcessError as e:
            print(f"  Warning: macOS say failed: {e}")
            # Fall through to next option

    # Option B: gTTS (Google TTS, requires internet)
    try:
        from gtts import gTTS

        tts = gTTS(text=text, lang="en", slow=False)
        tts.save(str(output_path))
        return True
    except ImportError:
        pass
    except Exception as e:
        print(f"  Warning: gTTS failed: {e}")

    # Option C: pyttsx3 (offline, cross-platform)
    try:
        import pyttsx3

        engine = pyttsx3.init()
        engine.save_to_file(text, str(output_path))
        engine.runAndWait()
        return True
    except ImportError:
        pass
    except Exception as e:
        print(f"  Warning: pyttsx3 failed: {e}")

    return False


def check_ffmpeg():
    """Check if ffmpeg is installed."""
    if os.system("which ffmpeg > /dev/null 2>&1") != 0:
        print("Error: ffmpeg not found. Install with:")
        print("  macOS: brew install ffmpeg")
        print("  Ubuntu: sudo apt-get install ffmpeg")
        print("  Windows: https://ffmpeg.org/download.html")
        return False
    return True


def install_dependencies():
    """Install required Python packages."""
    print("Checking dependencies...")

    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        print("Installing playwright...")
        subprocess.run(["pip3", "install", "playwright"], check=True)
        subprocess.run(["playwright", "install", "chromium"], check=True)

    # Check for TTS options
    if os.system("which say > /dev/null 2>&1") != 0:
        try:
            import gtts
        except ImportError:
            print("Installing gTTS (Google TTS)...")
            subprocess.run(["pip3", "install", "gtts"], check=True)


def capture_slides():
    """Capture slides as images using playwright."""
    from playwright.sync_api import sync_playwright

    slide_images = []

    with sync_playwright() as p:
        browser = p.chromium.launch()
        page = browser.new_page(viewport={"width": 1920, "height": 1080})

        slides_html = Path(__file__).parent / "index.html"
        page.goto(f"file://{slides_html.absolute()}")

        # Wait for reveal.js to load
        page.wait_for_selector(".reveal")
        page.wait_for_timeout(1000)

        total_slides = len(SCRIPTS)

        for i in range(total_slides):
            print(f"  Capturing slide {i + 1}...")
            page.evaluate(f"Reveal.slide({i})")
            page.wait_for_timeout(500)

            screenshot_path = f"/tmp/slide_{i + 1:02d}.png"
            page.screenshot(path=screenshot_path, full_page=False)
            slide_images.append(screenshot_path)

        browser.close()

    return slide_images


def create_video(slide_images, audio_files):
    """Combine images and audio into video."""

    # Create concat file for ffmpeg
    concat_file = "/tmp/video_concat.txt"
    audio_concat = "/tmp/audio_concat.txt"

    with open(concat_file, "w") as vf, open(audio_concat, "w") as af:
        for img, audio in zip(slide_images, audio_files):
            if audio.exists():
                # Get audio duration
                result = subprocess.run(
                    [
                        "ffprobe",
                        "-v",
                        "error",
                        "-show_entries",
                        "format=duration",
                        "-of",
                        "default=noprint_wrappers=1:nokey=1",
                        str(audio),
                    ],
                    capture_output=True,
                    text=True,
                )
                try:
                    duration = float(result.stdout.strip())
                except ValueError:
                    duration = 5.0  # default duration

                vf.write(f"file '{img}'\n")
                vf.write(f"duration {duration}\n")

                af.write(f"file '{audio}'\n")

        # Last image
        if slide_images:
            vf.write(f"file '{slide_images[-1]}'\n")

    # Build video
    print("Rendering video...")

    # Step 1: Create video from images
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            concat_file,
            "-vf",
            "fps=30,format=yuv420p",
            "-c:v",
            "libx264",
            "-preset",
            "fast",
            "-crf",
            "23",
            "-pix_fmt",
            "yuv420p",
            "/tmp/video_only.mp4",
        ],
        check=True,
        capture_output=True,
    )

    # Step 2: Concatenate audio
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-f",
            "concat",
            "-safe",
            "0",
            "-i",
            audio_concat,
            "-c:a",
            "libmp3lame",
            "-q:a",
            "2",
            "/tmp/audio_only.mp3",
        ],
        check=True,
        capture_output=True,
    )

    # Step 3: Combine video and audio
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-i",
            "/tmp/video_only.mp4",
            "-i",
            "/tmp/audio_only.mp3",
            "-c:v",
            "copy",
            "-c:a",
            "aac",
            "-b:a",
            "192k",
            "-shortest",
            "-movflags",
            "+faststart",
            OUTPUT_FILE,
        ],
        check=True,
        capture_output=True,
    )

    # Cleanup temp files
    for f in ["/tmp/video_only.mp4", "/tmp/audio_only.mp3", concat_file, audio_concat]:
        if os.path.exists(f):
            os.unlink(f)

    print(f"\n✓ Video created: {OUTPUT_FILE}")
    print(f"  Duration: ~{sum(len(SCRIPTS[i]) for i in SCRIPTS) / 15:.0f} seconds")


def main():
    print("=" * 60)
    print("Trindex Presentation Video Generator")
    print("=" * 60)

    if not check_ffmpeg():
        return 1

    try:
        install_dependencies()
    except subprocess.CalledProcessError as e:
        print(f"Error installing dependencies: {e}")
        return 1

    with tempfile.TemporaryDirectory() as tmpdir:
        tmpdir = Path(tmpdir)

        # Generate audio files
        print("\nGenerating audio narration...")
        audio_files = []

        for slide_num, script in SCRIPTS.items():
            audio_path = tmpdir / f"slide_{slide_num:02d}.mp3"
            if text_to_speech(script, audio_path):
                audio_files.append(audio_path)
                print(f"  ✓ Slide {slide_num}")
            else:
                print(f"  ✗ Slide {slide_num} - TTS failed")

        if not audio_files:
            print("Error: No audio files generated. Install a TTS engine:")
            print("  macOS: Built-in (say command)")
            print("  All: pip3 install gtts")
            print("  All: pip3 install pyttsx3")
            return 1

        # Capture slides
        print("\nCapturing slide screenshots...")
        try:
            slide_images = capture_slides()
        except Exception as e:
            print(f"Error capturing slides: {e}")
            return 1

        # Create video
        try:
            create_video(slide_images, audio_files)
        except subprocess.CalledProcessError as e:
            print(f"Error creating video: {e}")
            return 1

        # Cleanup screenshots
        for img in slide_images:
            if os.path.exists(img):
                os.unlink(img)

    print("\nDone! Open the video to see the result.")
    return 0


if __name__ == "__main__":
    exit(main())
