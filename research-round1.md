# Research: Rendering Mermaid Diagrams as ASCII Art in Go

**Date:** 2026-03-26
**Methodology:** 3-round iterative research with review agents identifying gaps and follow-up questions between rounds.

---

## Executive Summary

**Yes, this is possible in Go -- and the foundational work already exists.**

The JavaScript library `beautiful-mermaid` was actually **ported FROM a Go project** (`AlexanderGrooff/mermaid-ascii`), not the other way around. The lineage is Go → TypeScript. This means the core algorithms for Mermaid-to-ASCII rendering originated in Go.

**Key finding:** Two Go projects already solve this problem:
1. **AlexanderGrooff/mermaid-ascii** (1,291 stars, MIT, active) -- battle-tested, 2 diagram types, CLI-focused
2. **pgavlin/mermaid-ascii** (new fork, Mar 2026) -- 22 diagram types, library API, unproven

---

## 1. beautiful-mermaid (JavaScript/TypeScript)

**Repository:** [github.com/lukilabs/beautiful-mermaid](https://github.com/lukilabs/beautiful-mermaid)
**npm:** [beautiful-mermaid](https://www.npmjs.com/package/beautiful-mermaid)
**Stars:** ~8,400 | **License:** MIT

A pure TypeScript library by Craft (lukilabs) that renders Mermaid DSL into SVGs or terminal-friendly ASCII art.

### Features
- **6 diagram types:** Flowcharts, State, Sequence, Class, ER, XY Charts
- **Dual output:** SVG + ASCII/Unicode
- **Zero DOM dependencies:** Pure TS, works in Node/browser/CLI
- **15 built-in themes**, Shiki integration, CSS custom properties
- **Performance:** 100+ diagrams in under 500ms

### ASCII Rendering
- **Unicode mode** (default): Box-drawing characters (`┌───┐`, `│`, `└───┘`)
- **Pure ASCII mode**: Maximum compatibility (`+---+`, `|`, `+---+`)
- Config: `paddingX`, `paddingY`, `boxBorderPadding`, `colorMode` (none/ansi16/ansi256/truecolor)

### Origin
The ASCII engine was **ported from Go** (`mermaid-ascii` by Alexander Grooff), then extended with:
- Additional diagram types (State, Class, ER, XY Charts)
- SVG rendering alongside ASCII
- Theming system (CSS-based, 15 themes)
- Shiki integration for syntax highlighting

---

## 2. Core Go Projects

### 2a. AlexanderGrooff/mermaid-ascii -- THE ORIGINAL

| Metric | Value |
|--------|-------|
| **Repository** | [github.com/AlexanderGrooff/mermaid-ascii](https://github.com/AlexanderGrooff/mermaid-ascii) |
| **Stars** | 1,291 |
| **Forks** | 53 |
| **License** | MIT |
| **Language** | Go (96.6%) |
| **Created** | February 2023 |
| **Last Push** | February 21, 2026 |
| **Releases** | 10+ (v0.5.1 → v1.1.0) |
| **Open Issues** | 16 |
| **Web Demo** | [mermaid-ascii.art](https://mermaid-ascii.art/) |

**Supported Diagram Types:** Flowcharts (LR/TD/BT/RL) + Sequence diagrams

**Features:**
- Node shapes: round, stadium, subroutine, cylinder, circle, diamond, hexagon, flag
- Edge types: solid, dotted, thick, bidirectional, cross, circle
- Edge labels, subgraphs (including nested), classDef styling
- Colored output (ANSI), Unicode and ASCII rendering modes

**Architecture (4-stage pipeline):**
1. **Detect** -- auto-detect diagram type
2. **Parse** -- Mermaid syntax → diagram model
3. **Layout** -- grid placement + A* pathfinding for edge routing
4. **Render** -- draw to 2D character canvas → string

Nodes occupy a 3x3 area of grid points. Edge routing uses A* pathfinding to avoid overlaps.

**Release History:**
- v0.6.0: Added Unicode box-drawing
- v0.8.0: Added subgraphs
- v1.0.0: Added sequence diagrams
- v1.1.0: Added autonumber for sequence diagrams

**Library API Status:** Primarily a CLI tool. Issue #50 requests a proper library API. Can be imported via `pkg/diagram` and `pkg/render` packages, but the API may not be considered stable.

**Known Issues:**
- #30: Unexpected arrow rendering (v0.7.0+)
- #56: No support for multiple arrows between same nodes
- #25: Arrow direction limitations
- #46: Node labels treated as part of node ID

**Verdict:** Actively maintained, battle-tested, industry-adopted (Cursor CLI, Python wrappers). Limited to 2 diagram types.

### 2b. pgavlin/mermaid-ascii -- EXTENDED FORK

| Metric | Value |
|--------|-------|
| **Repository** | [github.com/pgavlin/mermaid-ascii](https://github.com/pgavlin/mermaid-ascii) |
| **Stars** | 0 (brand new) |
| **Last Push** | March 6, 2026 |
| **Commits** | 236 (31 beyond upstream's 205) |
| **Releases** | None |

**Supported Diagram Types:** 22 -- with dedicated packages for each:

```
architecture, blockdiagram, c4, classdiagram, erdiagram, gantt,
gitgraph, graph, journey, kanban, mindmap, packet, piechart,
quadrant, requirement, sankey, sequence, statediagram, timeline,
xychart, zenuml
```

Plus supporting packages: `canvas`, `diagram`, `layout`, `parser`, `render`

**Library API:**
```go
import (
    "github.com/pgavlin/mermaid-ascii/pkg/render"
    "github.com/pgavlin/mermaid-ascii/pkg/diagram"
)

output, err := render.Render(input, diagram.DefaultConfig())
diagramType := render.Detect(input)
```

**Verification:** The `/pkg` directory contains 22 diagram-type-specific packages, each with its own rendering implementation using a shared `canvas` package. This is NOT just detection/parsing -- each package has dedicated rendering code. Gantt charts confirmed working. Quality for inherently-visual types (pie, Sankey, mindmap) is unverified.

**Verdict:** Most complete Go solution. Brand new, zero community validation, single developer. High potential but risky as a dependency.

### 2c. sammcj/go-mermaid -- PARSER ONLY

**Package:** [github.com/sammcj/go-mermaid](https://pkg.go.dev/github.com/sammcj/go-mermaid)
**License:** Apache 2.0 | **Published:** November 2025

Pure Go parser, validator, and linter. 21+ diagram type ASTs via `ast.Diagram` interface. No rendering.
- Performance: ~6.3μs per flowchart parse, ~220ns per validation
- Uses different internal representations than mermaid-ascii (not directly compatible)
- Main value over mermaid-ascii's parser: validation/linting capabilities

---

## 3. Go ASCII/Text Diagram Libraries

### Diagram/Chart Rendering

| Library | Stars | Description |
|---------|-------|-------------|
| [gizak/termui](https://github.com/gizak/termui) | ~13.5k | Terminal dashboards, charts, grid layout |
| [guptarohit/asciigraph](https://github.com/guptarohit/asciigraph) | ~3k | ASCII line graphs (time series only) |
| [NimbleMarkets/ntcharts](https://github.com/NimbleMarkets/ntcharts) | -- | Terminal charts for BubbleTea TUI |
| [blampe/goat](https://github.com/blampe/goat) | ~773 | ASCII art → SVG (reverse direction, used in Hugo) |
| [tombrk/asciimatrix](https://github.com/tombrk/asciimatrix) | -- | Place strings at x/y on 2D character canvas |
| [thediveo/go-asciitree](https://github.com/thediveo/go-asciitree) | -- | Tree structures as ASCII |

### Graph Data Structures & Layout

| Library | Stars | Description |
|---------|-------|-------------|
| [dominikbraun/graph](https://github.com/dominikbraun/graph) | ~2k | Generic graph + DOT export (no layout) |
| [nikolaydubina/go-graph-layout](https://github.com/nikolaydubina/go-graph-layout) | ~95 | Sugiyama-style DAG layout (WIP) |
| [goccy/go-graphviz](https://github.com/goccy/go-graphviz) | -- | Graphviz via WASM (SVG/PNG, no ASCII) |
| [gonum/graph](https://pkg.go.dev/gonum.org/v1/gonum/graph) | -- | Graph algorithms + DOT marshaling |
| [terrastruct/d2](https://github.com/terrastruct/d2) | High | Modern diagram language, own DSL. Has alpha ASCII renderer using ELK layout downscaling -- alternative architectural approach worth investigating |

### Non-Go Reference

| Tool | Description |
|------|-------------|
| [Graph::Easy](https://metacpan.org/pod/Graph::Easy) (Perl) | Gold standard for ASCII graph rendering. Manhattan grid layout. No Go equivalent. |

---

## 4. Implementation Approaches

### Approach A: Use pgavlin/mermaid-ascii (Best Coverage)

Use the pgavlin fork as a Go library dependency. It already has 22 diagram types with a clean `render.Render()` API.

| | |
|---|---|
| **Pros** | 22 diagram types, library API, pure Go, builds on proven architecture |
| **Cons** | Brand new (0 stars), single developer, no releases, unverified quality for some types |
| **Effort** | Low (import and use) |
| **Risk** | Medium-high (dependency on unproven fork) |

### Approach B: Use AlexanderGrooff/mermaid-ascii (Most Stable)

Use the original as a Go library. Battle-tested for flowcharts + sequence diagrams.

| | |
|---|---|
| **Pros** | 1,291 stars, MIT, actively maintained, industry-adopted (Cursor CLI), proven quality |
| **Cons** | Only 2 diagram types, CLI-focused (library API not first-class) |
| **Effort** | Low for flowcharts/sequence, high to extend |
| **Risk** | Low |

### Approach C: Fork + Extend mermaid-ascii

Fork AlexanderGrooff's original and add diagram types, potentially cherry-picking from pgavlin's fork or back-porting from beautiful-mermaid's TypeScript.

| | |
|---|---|
| **Pros** | Start from proven base, add what you need, control quality |
| **Cons** | Significant effort per diagram type (parser + layout + renderer) |
| **Effort** | Medium-high |
| **Risk** | Low-medium |

### Approach D: sammcj/go-mermaid Parser + Custom Renderer

Use the most comprehensive parser and build ASCII renderers on top.

| | |
|---|---|
| **Pros** | Best parser coverage (21+ types), clean AST, validation/linting |
| **Cons** | Must build ALL rendering from scratch, layout algorithms are the hard part |
| **Effort** | Very high |
| **Risk** | Medium |

### Approach E: WASM Bridge to beautiful-mermaid

Compile the TypeScript library to WASM, invoke from Go via wazero/wasmer-go.

| | |
|---|---|
| **Pros** | Full feature parity with beautiful-mermaid immediately |
| **Cons** | Large WASM binary, runtime overhead, complex build, JS ecosystem dependency |
| **Effort** | Medium |
| **Risk** | Medium (complexity) |

### Approach F: Shell Out to beautiful-mermaid CLI

Call the npm package as a subprocess.

| | |
|---|---|
| **Pros** | Simplest implementation, full feature support |
| **Cons** | Requires Node.js runtime, slower (process spawn), not embeddable |
| **Effort** | Very low |
| **Risk** | Low (but adds runtime dependency) |

---

## 5. Industry Adoption

- **Cursor CLI** (Feb 2026): Renders Mermaid inline as ASCII in terminal. Ctrl+O toggles rendered/raw views.
- **Gemini CLI**: Feature request for inline Mermaid rendering with Sixel/iTerm2/Kitty support + ASCII fallback.
- **Python wrappers**: `osl-packages/mermaid-ascii` (ships Go binary in pip wheel), `mermaid-ascii-diagrams` (pure Python)
- **LobeHub Skills**: `beautiful-mermaid-ascii` wraps the npm package

---

## 6. Technical Challenges

### Layout (The Hardest Problem)
- **Sugiyama/layered layout**: Standard for DAGs (cycle removal → layer assignment → crossing minimization [NP-hard] → coordinate assignment)
- mermaid-ascii uses a simpler grid-based approach with A* pathfinding
- Character cells are ~2:1 aspect ratio, distorting layouts
- Converting continuous coordinates to discrete character positions adds quantization challenges

### Edge Routing
- A* pathfinding on character grid works but has known limitations:
  - No edge bundling (can't draw parallel edges between same nodes)
  - Fixed 3-grid-point granularity limits routing resolution
  - No iterative refinement (node positions fixed after placement)
  - Complex nested subgraphs stress the pathfinder

### Mermaid Syntax
- 22+ diagram types, each with unique syntax and rendering logic
- Official parser uses Langium-based PEG grammar (since Mermaid v11)
- Syntax is a moving target -- keeping up with upstream changes is ongoing work
- AI-generated Mermaid has ~23% error rate

### Per-Diagram Challenges

| Diagram Type | ASCII Rendering Challenge |
|-------------|--------------------------|
| Flowcharts | Subgraph nesting, complex edge routing, node shape variety |
| Sequence | Activation boxes, loops/alt blocks, message ordering |
| Class | Inheritance arrows, method/field lists, relationship labels |
| ER | Cardinality notation, relationship diamonds |
| State | Nested states, transitions, start/end markers |
| Gantt/Timeline | Time axis scaling, bar alignment |
| Pie/Sankey | Inherently visual -- hard to represent meaningfully in ASCII |
| XY Charts | Axis scaling, multi-series in character cells |

---

## 7. Recommendation

### For Immediate Use: AlexanderGrooff/mermaid-ascii

If you only need flowcharts and sequence diagrams, this is the proven choice. 1,291 stars, MIT license, actively maintained, used by Cursor CLI.

### For Broader Coverage: Evaluate pgavlin/mermaid-ascii

If you need more diagram types, evaluate the pgavlin fork. Test the specific diagram types you need and assess output quality. It has a proper library API and 22 diagram types, but is unproven.

### For Maximum Control: Fork + Extend

Fork mermaid-ascii and selectively add diagram types. Cherry-pick from pgavlin's fork or back-port from beautiful-mermaid's TypeScript. This gives full control over quality and scope.

### Key Insight

The lineage is **Go (mermaid-ascii) → TypeScript (beautiful-mermaid)**. The foundational algorithms originated in Go. Bringing the TypeScript extensions back to Go would be completing a round trip, not starting from scratch.

---

## Appendix: All Libraries Referenced

| Library | Language | Purpose | URL |
|---------|----------|---------|-----|
| AlexanderGrooff/mermaid-ascii | Go | Mermaid → ASCII (flowchart, sequence) | [GitHub](https://github.com/AlexanderGrooff/mermaid-ascii) |
| pgavlin/mermaid-ascii | Go | Mermaid → ASCII (22 types) | [GitHub](https://github.com/pgavlin/mermaid-ascii) |
| sammcj/go-mermaid | Go | Mermaid parser (21+ types) | [pkg.go.dev](https://pkg.go.dev/github.com/sammcj/go-mermaid) |
| beautiful-mermaid | TypeScript | Mermaid → SVG + ASCII (6 types) | [GitHub](https://github.com/lukilabs/beautiful-mermaid) |
| TyphonHill/go-mermaid | Go | Mermaid generator | [GitHub](https://github.com/TyphonHill/go-mermaid) |
| goldmark-mermaid | Go | Markdown extension | [GitHub](https://github.com/abhinav/goldmark-mermaid) |
| dominikbraun/graph | Go | Graph data structures | [GitHub](https://github.com/dominikbraun/graph) |
| nikolaydubina/go-graph-layout | Go | Sugiyama layout (WIP) | [GitHub](https://github.com/nikolaydubina/go-graph-layout) |
| goccy/go-graphviz | Go | Graphviz via WASM | [GitHub](https://github.com/goccy/go-graphviz) |
| terrastruct/d2 | Go | Modern diagram language | [GitHub](https://github.com/terrastruct/d2) |
| guptarohit/asciigraph | Go | ASCII line graphs | [GitHub](https://github.com/guptarohit/asciigraph) |
| blampe/goat | Go | ASCII → SVG | [GitHub](https://github.com/blampe/goat) |
| Graph::Easy | Perl | ASCII graph rendering | [CPAN](https://metacpan.org/pod/Graph::Easy) |
