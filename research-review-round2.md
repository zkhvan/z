# Research Review: Round 2 -- Mermaid Diagrams as ASCII Art in Go

**Reviewer:** Technical Reviewer (Iteration 2)
**Date:** 2026-03-26

---

## 1. Assessment: Did Round 2 Address the Gaps from Round 1?

### Q1: GitHub Metrics (stars, last commits, contributors)
**Status: ADEQUATELY ADDRESSED**

Round 2 provided concrete numbers:
- AlexanderGrooff/mermaid-ascii: 1,291 stars, 53 forks, MIT license, last push Feb 21 2026, regular release cadence (v0.5.1 through v1.1.0). Actively maintained.
- pgavlin/mermaid-ascii: 0 stars, created March 2026, no releases, ~31 commits beyond upstream. Single-developer fork, very new.

This is a critical finding -- round 1 reported "~291 stars" for the original (likely stale data) while round 2 found 1,291 stars. The pgavlin fork being brand-new with zero community adoption is an important risk factor that round 1 did not convey.

### Q2: pgavlin Fork -- Does It Actually Render All 22 Types?
**Status: PARTIALLY ADDRESSED**

Round 2 found strong structural evidence: 22 diagram-type-specific packages under /pkg, each with its own rendering pipeline (Detect -> Parse -> Layout -> Render). The API surface (render.Render, render.Detect) is confirmed.

However, a key gap remains: **no verification of rendering quality for inherently non-graph diagram types** (pie charts, Sankey diagrams, Gantt charts, mindmaps as ASCII). The agent correctly flagged this uncertainty but did not attempt to run the tool or find example outputs. The claim "22 types" could mean anything from polished output to minimal/placeholder renderers.

### Q3: beautiful-mermaid Extensions After Porting from Go
**Status: ADEQUATELY ADDRESSED**

Round 2 enumerated specific extensions: dual SVG+ASCII output, 15-theme system with CSS color-mix(), Shiki integration, synchronous rendering, zero DOM dependencies. It also correctly identified that theming is CSS-specific and would need reimagining for Go/ANSI terminals.

The back-portability analysis was useful: layout algorithm improvements are the most portable piece; theming/SVG are web-specific.

### Q4: Parser Compatibility (sammcj/go-mermaid + mermaid-ascii)
**Status: ADEQUATELY ADDRESSED**

Round 2 confirmed the two projects use incompatible internal representations. It correctly assessed that adapter code would be needed and that the benefit is marginal given pgavlin's fork already has a complete parser. The note about sammcj's validation/linting capabilities being the unique value-add is insightful.

### Q5: A* Pathfinding Limitations for Edge Routing
**Status: ADEQUATELY ADDRESSED**

Round 2 found concrete GitHub issues (#30, #56, #25, #46) and enumerated structural limitations: no edge bundling, fixed grid granularity, no iterative refinement, limited parallel edge support. This is actionable information for evaluating the tool's suitability.

### Q6: Alternative Go Libraries Not Found in Round 1
**Status: ADEQUATELY ADDRESSED**

Round 2 confirmed a critical finding: **there is no competing Go library for Mermaid-to-ASCII rendering**. New discoveries include:
- **terrastruct/D2**: Go diagram tool with dagre layout, but its own DSL (not Mermaid) and no ASCII output. Has an alpha-stage ASCII renderer using ELK layout downscaling.
- **gverger/go-graph-layout**: Native Go Sugiyama layout (WIP), could serve as a foundation for custom rendering.
- **goccy/go-graphviz**: Go bindings for Graphviz via WASM, SVG/PNG output only.
- **graph-easy** (Perl): Gold standard for DOT-to-ASCII but not Go.

The D2 finding is particularly interesting -- round 1 missed it entirely. D2's alpha ASCII renderer represents an alternative architectural approach (downscale from high-res layout rather than render directly to grid).

---

## 2. Remaining Gaps

1. **pgavlin rendering quality verification**: Nobody has actually tested or shown ASCII output for the non-trivial diagram types (pie, Gantt, mindmap, Sankey, git graph). "Has a package" does not mean "produces good output."

2. **AlexanderGrooff/mermaid-ascii library API status**: Round 2 mentions issue #50 requesting library usage as a go-get dependency. It is unclear whether the tool can currently be imported as a Go library or only used as a CLI. Round 1 claimed `render.Render(input, config)` works, but round 2 suggests this may only be true for the pgavlin fork.

3. **D2 ASCII renderer depth**: D2 was surfaced as a discovery but not deeply investigated. Its ASCII renderer uses ELK layout (a sophisticated layered layout engine) rather than simple grid-based A* -- this could be architecturally superior for complex diagrams. The trade-off is that D2 uses its own DSL, so a Mermaid-to-D2 transpiler would be needed.

4. **License compatibility**: pgavlin's fork license was not confirmed. If it inherits MIT from upstream, that is fine, but this was not verified.

5. **Performance benchmarks**: No data on rendering speed for large diagrams in any of the Go tools. Round 1's beautiful-mermaid benchmark (100+ diagrams in <500ms) has no Go equivalent.

---

## 3. Follow-Up Questions for Round 3

1. **pgavlin/mermaid-ascii output quality**: Can you find or generate actual ASCII output examples from pgavlin's fork for at least 3 non-graph diagram types (e.g., Gantt, pie chart, mindmap)? Check the repo's test fixtures, README examples, or issue discussions.

2. **D2 ASCII renderer evaluation**: What is the actual output quality of D2's ASCII renderer for flowchart-like diagrams? Could a Mermaid-to-D2 transpiler approach be viable -- how different are the two DSLs syntactically? What is D2's star count and maintenance status?

3. **AlexanderGrooff/mermaid-ascii as a Go library**: Does issue #50 have a resolution? Can the original mermaid-ascii be imported as a library today (go get), or does it require vendoring/forking? What is the actual Go import path and API?

4. **Cursor CLI integration details**: Round 1 mentioned Cursor CLI renders Mermaid inline as ASCII. Which library does it use (mermaid-ascii, beautiful-mermaid, or something else)? How many diagram types does it support? This would reveal real-world quality expectations.

5. **Edge routing: grid-based A* vs. ELK/dagre downscaling**: What are the concrete trade-offs between mermaid-ascii's direct-to-grid approach and D2's approach of computing a high-res layout then downscaling to ASCII? Which produces better results for graphs with 10+ nodes and complex edge patterns?

---

## 4. Combined Research Quality Rating

**Rating: 7/10**

Strengths:
- Comprehensive landscape mapping -- the field is now well-characterized
- The pgavlin fork discovery and structural analysis is valuable
- D2 as an alternative approach was a good find
- Concrete GitHub issue numbers for edge routing problems
- The "no competition exists" finding is definitive and important
- Star count correction (1,291 vs ~291) fixes a significant round 1 error

Weaknesses:
- No hands-on verification of any tool's output quality
- The pgavlin fork's 22-type claim remains structurally plausible but unverified
- D2 was mentioned but not deeply explored
- Missing library-vs-CLI distinction for the original mermaid-ascii
- No performance data for any Go tool

---

## 5. Synthesized Recommendation

### The Landscape

The Go ecosystem for Mermaid-to-ASCII rendering is extremely narrow. There is exactly one lineage of tools: AlexanderGrooff/mermaid-ascii (the original, 1.3k stars, actively maintained, 2 diagram types) and pgavlin's recent fork (22 diagram types claimed, zero community validation). Everything else either parses without rendering, renders without Mermaid input, or is not in Go.

### Recommended Path

**For production use today:**
Use AlexanderGrooff/mermaid-ascii for flowcharts and sequence diagrams. These are the two most commonly needed diagram types and the tool is battle-tested (1.3k stars, used by Cursor CLI and the Python ecosystem). Verify whether it works as a Go library import or only as a CLI subprocess.

**For broader diagram type coverage:**
Evaluate pgavlin/mermaid-ascii carefully. Clone it, run it against your actual diagram inputs, and assess rendering quality. The 22-type coverage is structurally plausible (separate packages exist) but the fork is 3 weeks old with zero community adoption. Treat it as promising but unproven. Its library API (render.Render) is a significant advantage over the original.

**For the highest quality output:**
Shell out to beautiful-mermaid (npm/TypeScript). It supports 6 diagram types with superior theming and polish. The cost is a Node.js runtime dependency.

**Alternative architectural approach worth investigating:**
D2 (terrastruct) has an alpha ASCII renderer with ELK-based layout that may produce better results for complex graphs. However, it requires Mermaid-to-D2 syntax conversion, which adds a translation layer. Worth investigating if edge routing quality in mermaid-ascii proves insufficient.

### Key Insight Updated from Round 1

The lineage remains Go (mermaid-ascii) -> TypeScript (beautiful-mermaid), but round 2 revealed that the landscape has recently shifted. pgavlin's fork (March 2026) potentially solves the diagram-type-coverage gap that was the original's main limitation. The critical unknown is whether its rendering quality matches its structural ambition. A round 3 focused on hands-on testing would resolve this.
