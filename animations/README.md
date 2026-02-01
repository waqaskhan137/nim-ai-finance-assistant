# Nim Onboarding Animations

Animated explainers for the Nim product flow using [Manim](https://www.manim.community/).

## Installation

```bash
# Install Manim Community Edition
pip install manim

# On macOS, you may also need:
brew install py3cairo ffmpeg pango
```

## Available Animations

| Scene | Description | Duration |
|-------|-------------|----------|
| `NimOnboarding` | Full onboarding animation (12 scenes) | ~3 min |
| `NimFlowDiagram` | Simple flow diagram | ~30 sec |
| `EmergencyFundLadder` | Emergency fund progress visualization | ~20 sec |

## Render Commands

### Preview (Low Quality, Fast)
```bash
manim -pql nim_onboarding.py NimOnboarding
```

### Production (High Quality)
```bash
manim -pqh nim_onboarding.py NimOnboarding
```

### 4K Quality
```bash
manim -pqk nim_onboarding.py NimOnboarding
```

### GIF Output
```bash
manim -pql --format=gif nim_onboarding.py NimFlowDiagram
```

### Just the Flow Diagram
```bash
manim -pqh nim_onboarding.py NimFlowDiagram
```

## Output Location

Rendered videos are saved to:
```
media/videos/nim_onboarding/
├── 480p15/    # Low quality (-ql)
├── 720p30/    # Medium quality (-qm)
├── 1080p60/   # High quality (-qh)
└── 2160p60/   # 4K quality (-qk)
```

## Scenes Breakdown

### NimOnboarding (Full Animation)

1. **Title** - Nim logo and tagline
2. **Problem** - Shows why "Trading First" creates anxiety
3. **Solution** - Introduces Stabilize → Save → Invest
4. **Journey Overview** - Circular diagram of all 6 steps
5. **Step 1: Chat** - Conversational goal setting
6. **Step 2: Budget** - Envelope system with guardrails
7. **Step 3: Subscriptions** - Kill silent drains
8. **Step 4: Smart Savings** - Rules + Emergency Fund Ladder
9. **Step 5: Trading** - Surplus-gated investing
10. **Step 6: Autopilot** - Daily/Weekly/Monthly cadence
11. **Flywheel** - The virtuous cycle
12. **CTA** - Call to action

## Customization

### Color Palette
```python
NIM_BLUE = "#007AFF"    # Primary brand
NIM_GREEN = "#34C759"   # Positive/success
NIM_RED = "#FF3B30"     # Warning/negative
NIM_ORANGE = "#FF9500"  # Attention
NIM_PURPLE = "#AF52DE"  # Investing
NIM_TEAL = "#5AC8FA"    # Budget
NIM_GRAY = "#8E8E93"    # Secondary text
NIM_DARK = "#1C1C1E"    # Background
```

### Modify Scenes
Edit `nim_onboarding.py` and change:
- Text content in each `play_step_*` method
- Colors by updating the color constants
- Timing by adjusting `run_time` and `wait()` calls
- Layout by modifying positions and arrangements

## Embedding in App

### React/Web
1. Render as MP4 or WebM
2. Use `<video>` tag with autoplay
3. Or convert to Lottie JSON for interactive control

### Mobile (React Native)
1. Render as MP4
2. Use `expo-av` or `react-native-video`

### Export Frames for Custom Player
```bash
manim -pqh --save_last_frame nim_onboarding.py NimOnboarding
```

## Troubleshooting

### "LaTeX not found"
Manim uses LaTeX for math. For text-only animations (like these), it's not required.

### Slow rendering
Use `-ql` (low quality) for previewing, `-qh` only for final render.

### Missing fonts
The animation uses system fonts. On Linux, install:
```bash
sudo apt install fonts-dejavu
```
