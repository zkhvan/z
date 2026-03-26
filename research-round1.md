# Research: Mermaid Diagrams as ASCII/Terminal-Compatible Diagrams in Go

**Date:** 2026-03-26

---

## 1. beautiful-mermaid (JavaScript/TypeScript Library)

**Repository:** [github.com/lukilabs/beautiful-mermaid](https://github.com/lukilabs/beautiful-mermaid)
**Package:** [npm: beautiful-mermaid](https://www.npmjs.com/package/beautiful-mermaid)
**Website:** [agents.craft.do/mermaid](https://agents.craft.do/mermaid)

### What It Does

beautiful-mermaid is a tiny, pure-TypeScript library that turns Mermaid DSL into beautiful SVGs or terminal-friendly ASCII art. It is the engine powering Craft Agents' diagram support.

### Key Features

- **6 diagram types:** Flowcharts, State, Sequence, Class, ER, and XY Charts (bar, line, combined)
- **Dual output:** SVG for rich UIs, ASCII/Unicode for terminals
- **Zero DOM dependencies:** Pure TS, works in Node, browser, or CLI
- **Rich theming:** 15 built-in themes, two-color mono mode, enriched mode, Shiki integration
- **Live theme switching:** CSS custom properties, no re-render
- **Performance:** 100+ diagrams in under 500ms

### ASCII Rendering Approach

The ASCII rendering engine supports two modes:
- **Unicode mode** (default): Uses box-drawing characters (`┌───┐`, `│`, `└───┘`)
- **Pure ASCII mode**: Maximum compatibility (`+---+`, `|`, `+---+`)

Options include `paddingX`, `paddingY`, `boxBorderPadding`, and `colorMode` settings (`'none'`, `'auto'`, `'ansi16'`, `'ansi256'`, `'truecolor'`, `'html'`).

### Origin

The ASCII rendering engine was **ported from Go to TypeScript** from Alexander Grooff's `mermaid-ascii` (a Go project), then extended with additional diagram types and theming. This means the foundational work already exists in Go.

---

## 2. Go Mermaid Parsing Libraries

### 2a. AlexanderGrooff/mermaid-ascii (Go CLI + Library) -- THE KEY PROJECT

**Repository:** [github.com/AlexanderGrooff/mermaid-ascii](https://github.com/AlexanderGrooff/mermaid-ascii)
**Stars:** ~291 | **Forks:** ~12 | **License:** Not specified
**Latest Release:** v0.6.0
**Web Demo:** [mermaid-ascii.art](https://mermaid-ascii.art/)

This is the most important project for our purposes. It is a **pure Go** tool that both parses Mermaid syntax AND renders it as ASCII/Unicode art in the terminal. It includes its own Mermaid parser.

**Supported Diagram Types:**
- Graph/Flowchart diagrams (LR, TD/TB, BT, RL directions)
- Sequence diagrams (participants, self-messages, aliases)
- Subgraphs (including nested)

**Supported Features:**
- Node shapes: round, stadium, subroutine, cylinder, circle, diamond, hexagon, flag
- Edge types: solid, dotted, thick, bidirectional, cross, circle
- Edge labels
- `classDef` styling
- Colored output (ANSI)
- Unicode and ASCII rendering modes

**Architecture:**
- Nodes are placed on a grid coordinate system (each node occupies a 3x3 area of grid points)
- Pathfinding is used to route edges between nodes without overlapping
- Library usage: import `pkg/diagram` and `pkg/render`, call `render.Render(input, diagram.DefaultConfig())`

**Limitations (compared to beautiful-mermaid's TS port):**
- No State diagrams
- No Class diagrams
- No ER diagrams
- No XY Charts
- No theming system

### 2b. pgavlin/mermaid-ascii (Extended Fork)

**Package:** [pkg.go.dev/github.com/pgavlin/mermaid-ascii](https://pkg.go.dev/github.com/pgavlin/mermaid-ascii)

A notable fork of the above that extends it with:
- **Architecture-beta diagrams** (`pkg/architecture` package)
- **C4 diagrams** (C4Context, C4Container, C4Component, C4Dynamic via `pkg/c4` package)
- Claims support for 22 diagram types total

This fork has a clean package structure with `Parse()` and `Render()` functions per diagram type.

### 2c. sammcj/go-mermaid (Parser, Validator, Linter)

**Package:** [pkg.go.dev/github.com/sammcj/go-mermaid](https://pkg.go.dev/github.com/sammcj/go-mermaid)
**License:** Apache 2.0 | **Published:** November 2025

A pure Go parser, validator, and linter for Mermaid diagrams. This is a **parse-only** library (no rendering).

- 21+ type-specific parsers producing a complete AST
- Strongly-typed diagram representations implementing an `ast.Diagram` interface
- All parsers meet <100ms target for diagrams up to 1000 lines
- Works with `.mmd` files and markdown files containing Mermaid code blocks

This could serve as the parser component if you wanted to build your own renderer.

### 2d. TyphonHill/go-mermaid (Diagram Generator)

**Repository:** [github.com/TyphonHill/go-mermaid](https://github.com/TyphonHill/go-mermaid)
**Stars:** ~38

A Go library for **generating** Mermaid diagram syntax programmatically (not parsing or rendering). Supports flowcharts and user journey diagrams. Goes the opposite direction from what we need.

### 2e. goldmark-mermaid (Markdown Extension)

**Package:** [go.abhg.dev/goldmark/mermaid](https://pkg.go.dev/go.abhg.dev/goldmark/mermaid)

A goldmark Markdown parser extension for Mermaid diagrams. Supports client-side rendering (JavaScript injection) and server-side rendering (MermaidJS CLI). Not a Mermaid parser itself -- it just identifies Mermaid code blocks in Markdown.

---

## 3. Go ASCII/Text Diagram Rendering Libraries

### General-Purpose Terminal Rendering

| Library | Stars | Description | URL |
|---------|-------|-------------|-----|
| `gizak/termui` | ~13.5k | Terminal dashboard with widgets, charts, and grid layout | [GitHub](https://github.com/gizak/termui) |
| `guptarohit/asciigraph` | -- | Lightweight ASCII line graphs with zero deps | [GitHub](https://github.com/guptarohit/asciigraph) |
| `NimbleMarkets/ntcharts` | -- | Terminal charts for BubbleTea TUI framework | [GitHub](https://github.com/NimbleMarkets/ntcharts) |
| `buger/goterm` | -- | Advanced terminal output: boxes, tables, line charts | [GitHub](https://github.com/buger/goterm) |
| `chriskim06/drawille-go` | -- | Braille character-based high-res terminal plotting | [GitHub](https://github.com/chriskim06/drawille-go) |

### Graph Layout Algorithms in Go

| Library | Description | URL |
|---------|-------------|-----|
| `nikolaydubina/go-graph-layout` | Sugiyama-style DAG layout algorithms (WIP) | [GitHub](https://github.com/nikolaydubina/go-graph-layout) |
| `dominikbraun/graph` | ~2k stars. Generic graph data structures + algorithms (DFS, BFS, shortest path, topo sort). DOT/Graphviz visualization. | [GitHub](https://github.com/dominikbraun/graph) |
| `heimdalr/dag` | Thread-safe DAG implementation | [GitHub](https://github.com/heimdalr/dag) |
| `goombaio/dag` | DAG implementation in Go | [GitHub](https://github.com/goombaio/dag) |

### ASCII Art From Diagrams (Reverse Direction)

| Library | Description | URL |
|---------|-------------|-----|
| `esimov/diagram` | Converts ASCII art diagrams to hand-drawn PNGs | [GitHub](https://github.com/esimov/diagram) |
| `akavel/ditaa` | Go port of DITAA (Diagrams Through ASCII Art) -- ASCII to bitmap | [GitHub](https://github.com/akavel/ditaa) |

---

## 4. Approaches to Achieve Mermaid-to-ASCII in Go

### Approach A: Use mermaid-ascii Directly (Recommended Starting Point)

**AlexanderGrooff/mermaid-ascii** (or the **pgavlin fork**) already does exactly what we want for flowcharts and sequence diagrams. It is pure Go, has its own parser and renderer, and is the original codebase that beautiful-mermaid was ported FROM.

- **Pros:** Already works, pure Go, battle-tested (used by beautiful-mermaid, Cursor CLI, Python wrappers), actively maintained
- **Cons:** Fewer diagram types than beautiful-mermaid's TS version, limited theming
- **Effort:** Can be used as a Go library today with `render.Render(input, config)`

### Approach B: Extend mermaid-ascii with More Diagram Types

Take the pgavlin fork (which already supports 22 diagram types) or extend the original with additional diagram types (State, Class, ER, XY Charts) to match beautiful-mermaid's feature set.

- **Pros:** Stays pure Go, builds on proven architecture
- **Cons:** Significant implementation effort per diagram type (each needs a parser + layout algorithm + renderer)
- **Effort:** Medium-to-high

### Approach C: Use sammcj/go-mermaid Parser + Custom ASCII Renderer

Use the comprehensive 21+ diagram type AST parser from `sammcj/go-mermaid` and build custom ASCII renderers for each diagram type.

- **Pros:** Best parser coverage (21+ types), clean AST, Apache 2.0 licensed
- **Cons:** Must build all rendering from scratch, layout algorithms are the hard part
- **Effort:** High

### Approach D: Compile beautiful-mermaid to WASM and Call from Go

Compile the TypeScript beautiful-mermaid library (or mermaid.js itself) to WASM and invoke it from Go.

- **Pros:** Gets full feature parity with beautiful-mermaid immediately
- **Cons:** Large WASM binary, runtime overhead, complex build pipeline, dependency on JS ecosystem
- **Effort:** Medium (integration), but adds complexity to distribution

### Approach E: Shell Out to Node.js / beautiful-mermaid CLI

Call the `beautiful-mermaid` npm package or `mermaid-ascii` Python package as a subprocess from Go.

- **Pros:** Simplest to implement, full feature support
- **Cons:** Requires Node.js/Python runtime, slower (process spawn), not embeddable as a library
- **Effort:** Low

---

## 5. Existing Projects / Real-World Usage

### Direct Implementations

1. **AlexanderGrooff/mermaid-ascii** -- The original Go implementation. This IS the existing project that does Mermaid-to-ASCII in Go. (291 stars)

2. **pgavlin/mermaid-ascii** -- Extended Go fork with architecture and C4 diagram support.

3. **beautiful-mermaid** -- TypeScript port and extension OF the Go project above. Proves the concept works and has been extended to 6 diagram types.

### Industry Adoption (2026)

- **Cursor CLI** (Feb 2026): Renders Mermaid code blocks inline as ASCII diagrams in terminal conversations. Supports flowcharts, sequence diagrams, state machines, class diagrams, and ER diagrams. Ctrl+O toggles between rendered and raw views.

- **Gemini CLI** (feature request): Proposal for a `visualize` tool that renders Mermaid inline in terminal with image protocol support (Sixel, iTerm2, Kitty) and ASCII box-drawing fallback.

- **LobeHub Skills**: A `beautiful-mermaid-ascii` skill wraps the npm package for CLI use.

### Python Ecosystem

- **osl-packages/mermaid-ascii** on PyPI: Ships the upstream Go binary inside a Python wheel.
- **mermaid-ascii-diagrams** on PyPI: Pure Python implementation supporting flowchart and sequence diagrams.

---

## 6. Key Technical Challenges

### Layout Algorithms (The Hardest Problem)

The core challenge in rendering diagrams as ASCII art is **layout**: placing nodes and routing edges in a 2D text grid.

- **Sugiyama/layered layout** is the standard algorithm for directed graphs (used by Graphviz `dot`). It involves: (1) cycle removal, (2) layer assignment, (3) crossing minimization (NP-hard), and (4) coordinate assignment.
- `nikolaydubina/go-graph-layout` is attempting a pure Go implementation but is still WIP (~80% complete on Brandes-Kopf coordinate assignment).
- `mermaid-ascii` uses a simpler grid-based approach with pathfinding for edge routing, which works well for flowcharts but may not scale to complex diagrams.

### Mermaid Syntax Complexity

Mermaid supports 15+ diagram types, each with its own syntax:
- Flowcharts, sequence, class, state, ER, Gantt, pie, journey, git graph, mindmap, timeline, sankey, xychart, block, architecture, C4, etc.
- Each type requires its own parser and dedicated layout/rendering logic.
- The official Mermaid.js parser uses a PEG grammar (Langium-based since Mermaid v11).

### Edge Routing

- ASCII art constrains edges to horizontal, vertical, and limited diagonal lines.
- Avoiding edge-node and edge-edge overlaps in a character grid is a constraint satisfaction problem.
- `mermaid-ascii` uses A*-like pathfinding on the grid, which works but can produce suboptimal routes for complex graphs.

### Text Measurement

- In graphical renderers, text width is measured in pixels. In ASCII, it is measured in characters.
- Unicode characters (especially CJK, emoji) may have different terminal widths.
- Box-drawing characters work well in most monospace fonts but can have alignment issues in some terminals.

### Diagram-Specific Challenges

| Diagram Type | Challenge |
|-------------|-----------|
| Flowcharts | Subgraph nesting, complex edge routing, node shape variety |
| Sequence | Activation boxes, loops/alt blocks, message ordering |
| Class | Inheritance arrows, method/field lists, relationship labels |
| ER | Cardinality notation, relationship diamonds |
| State | Nested states, transitions, start/end markers |
| Gantt/Timeline | Time axis scaling, bar alignment |
| XY Charts | Axis scaling, multi-series rendering in character cells |

---

## 7. Summary and Recommendation

**The answer is yes -- and it largely already exists.**

The Go project `AlexanderGrooff/mermaid-ascii` already renders Mermaid diagrams as ASCII/Unicode art in the terminal. It was, in fact, the **original inspiration** for the JavaScript library `beautiful-mermaid`, which ported its Go code to TypeScript and extended it.

### Recommended Path

1. **Start with `AlexanderGrooff/mermaid-ascii`** (or the `pgavlin` fork) as a Go library dependency. It already supports flowcharts, sequence diagrams, subgraphs, and multiple node/edge styles.

2. **For broader diagram type coverage**, either:
   - Extend the existing Go codebase with additional diagram type renderers (State, Class, ER, etc.)
   - Use `sammcj/go-mermaid` as the parser and build renderers on top of its AST
   - Port the additional renderers from beautiful-mermaid's TypeScript back to Go

3. **For immediate full coverage** with less effort, shell out to the `beautiful-mermaid` npm CLI or embed it via WASM -- but this sacrifices the pure-Go advantage.

### Key Insight

The lineage is: **Go (mermaid-ascii) -> TypeScript (beautiful-mermaid)**. The foundational algorithms and approach originated in Go. Bringing the TypeScript extensions back to Go would be completing a round trip, not starting from scratch.
