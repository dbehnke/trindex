// Text-to-Speech integration for reveal.js slides
// Add this script to index.html before the closing </body> tag

class SlideNarrator {
    constructor() {
        this.synth = window.speechSynthesis;
        this.utterance = null;
        this.isEnabled = false;
        this.currentSlide = 0;
        
        // Narration scripts for each slide (customize these!)
        this.scripts = {
            0: "Trindex: Persistent Semantic Memory for AI Agents.",
            1: "What is Trindex? It is a standalone Go binary that provides persistent, semantic memory for AI agents via the Model Context Protocol.",
            2: "The problem: AI agents have no memory. Every conversation starts fresh, context windows are limited, and building custom memory systems is complex.",
            3: "Key features: MCP-native integration works with Claude Code, opencode, and Cursor. Semantic search combines vector similarity with full-text search. Namespace organization with global fallback.",
            4: "Architecture: MCP clients communicate via standard IO. Trindex has MCP mode for agents and server mode for HTTP API and web UI. Everything stores in PostgreSQL with pgvector.",
            5: "CLI redesign: Before, Trindex was monolithic. The new CLI uses explicit subcommands: MCP for agent integration, server for HTTP only, doctor for diagnostics, and memories for CRUD operations.",
            6: "CLI demo: Run doctor to verify configuration. Start server with custom ports. Use memories commands for list, get, create, and delete operations.",
            7: "Technical stack: Go 1.26 plus, PostgreSQL 17 with pgvector, HNSW index, hybrid search, OpenAI-compatible embeddings, Chi router, Vue 3 with Tailwind.",
            8: "Database schema: The memories table stores ID, namespace, content, vector embedding, JSONB metadata, and generated tsvector for full-text search.",
            9: "Why Trindex? For developers: drop-in MCP memory. For DevOps: single binary with Docker Compose. For users: web UI, CLI scripting, and no vendor lock-in.",
            10: "Future roadmap: Phase 3 brings authentication and multi-tenancy. Phase 4 adds reranking and memory decay. Phase 5 includes LangChain integration and Python client.",
            11: "Getting started: Clone the repo, configure environment variables, and run with Docker Compose or build locally with Go.",
            12: "Demo time: Let me show you Trindex in action with diagnostics, creating memories, searching, and the web UI.",
            13: "Questions? Find the code on GitHub at dbehncke slash trindex. Documentation is in the docs folder. The project is under Business Source License 1.1.",
            14: "Thank you. Trindex gives AI agents persistent memory: one brain for every agent. Check it out and share your feedback."
        };
        
        // Listen for slide changes
        Reveal.on('slidechanged', (event) => {
            this.currentSlide = event.indexh;
            if (this.isEnabled && this.scripts[this.currentSlide]) {
                // Small delay to let slide transition finish
                setTimeout(() => this.speak(this.scripts[this.currentSlide]), 500);
            }
        });
        
        // Add toggle button
        this.addToggleButton();
        
        console.log('🎙️ Slide Narrator loaded. Click the TTS button to enable narration.');
    }
    
    speak(text) {
        // Cancel any ongoing speech
        this.synth.cancel();
        
        this.utterance = new SpeechSynthesisUtterance(text);
        this.utterance.rate = 0.9;  // Slightly slower for clarity
        this.utterance.pitch = 1.0;
        this.utterance.volume = 1.0;
        
        // Try to use a good voice
        this.setVoice();
        
        this.synth.speak(this.utterance);
    }
    
    setVoice() {
        const voices = this.synth.getVoices();
        
        // Preferred voices in order
        const preferredVoices = [
            'Samantha',           // macOS
            'Google US English',  // Chrome
            'Microsoft David',    // Windows
            'Microsoft Zira',
            'Alex'                // macOS fallback
        ];
        
        for (const preferred of preferredVoices) {
            const voice = voices.find(v => v.name.includes(preferred));
            if (voice) {
                this.utterance.voice = voice;
                return;
            }
        }
        
        // Fallback to first English voice
        const englishVoice = voices.find(v => v.lang.startsWith('en'));
        if (englishVoice) {
            this.utterance.voice = englishVoice;
        }
    }
    
    addToggleButton() {
        const btn = document.createElement('button');
        btn.id = 'tts-toggle';
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
            font-size: 14px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.3);
            transition: all 0.3s ease;
        `;
        
        btn.onmouseenter = () => {
            btn.style.transform = 'scale(1.05)';
        };
        btn.onmouseleave = () => {
            btn.style.transform = 'scale(1)';
        };
        
        btn.onclick = () => {
            this.isEnabled = !this.isEnabled;
            
            if (this.isEnabled) {
                btn.style.background = '#9ece6a';
                btn.innerHTML = '🔊 TTS ON';
                
                // Speak current slide
                if (this.scripts[this.currentSlide]) {
                    this.speak(this.scripts[this.currentSlide]);
                }
            } else {
                btn.style.background = '#bb9af7';
                btn.innerHTML = '🔊 TTS';
                this.synth.cancel();
            }
        };
        
        document.body.appendChild(btn);
        
        // Keyboard shortcut: Press 'T' to toggle
        document.addEventListener('keydown', (e) => {
            if (e.key === 't' && !e.ctrlKey && !e.metaKey && !e.altKey) {
                // Don't trigger if typing in an input
                if (e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
                    btn.click();
                }
            }
        });
    }
    
    // Public API
    enable() {
        this.isEnabled = true;
        document.getElementById('tts-toggle').style.background = '#9ece6a';
        document.getElementById('tts-toggle').innerHTML = '🔊 TTS ON';
    }
    
    disable() {
        this.isEnabled = false;
        this.synth.cancel();
        document.getElementById('tts-toggle').style.background = '#bb9af7';
        document.getElementById('tts-toggle').innerHTML = '🔊 TTS';
    }
}

// Initialize when reveal is ready
if (typeof Reveal !== 'undefined') {
    Reveal.on('ready', () => {
        window.narrator = new SlideNarrator();
    });
} else {
    console.error('Reveal.js not loaded. Make sure to include this script after reveal.js.');
}
