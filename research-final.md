# Final Research Report: Rendering Mermaid Diagrams as ASCII Art in Go

**Date:** 2026-03-26
**Methodology:** 3-round iterative research with review agents identifying gaps between rounds.

---

## Executive Summary

**The Go ecosystem has a clear winner for Mermaid-to-ASCII rendering: pgavlin/mermaid-ascii.**

Three rounds of research converged on this conclusion. The landscape is narrow -- exactly one lineage of tools exists (AlexanderGrooff's original and pgavlin's fork). Round 3 resolved the key uncertainty from earlier rounds: pgavlin's fork is not a speculative side project but a serious engineering effort by a highly credible developer, with 87.6% test coverage, 63+ test cases, a clean library API, and active development.

---

## The Landscape

| Project | Diagram Types | Stars | API Quality | Test Coverage | Status |
|---------|--------------|-------|-------------|---------------|--------|
| **pgavlin/mermaid-ascii** | 22 | 0 (new) | Clean: `render.Render(input, config)` | 87.6%, 63+ cases | Active (Mar 2026) |
| **AlexanderGrooff/mermaid-ascii** | 2 (flowchart, sequence) | 1,291 | Awkward: lives in `cmd/` package | Unknown | Active (Feb 2026) |
| **beautiful-mermaid** (TS) | 6 | ~8,400 | npm library | Unknown | Active |
| **sammcj/go-mermaid** | 21+ (parser only) | Low | Clean AST | Unknown | Active |

No other Go library renders Mermaid syntax to ASCII. This was confirmed across all three rounds.

### Lineage

Go (AlexanderGrooff/mermaid-ascii) --> TypeScript (beautiful-mermaid) --> Go (pgavlin/mermaid-ascii, extended)

The foundational algorithms originated in Go. beautiful-mermaid ported them to TypeScript and added diagram types. pgavlin's fork extends the Go original with its own parser unification and 22 diagram types.

---

## Key Findings by Round

### Round 1: Landscape Mapping
- Identified the two Go projects and the Go-to-TypeScript lineage
- Cataloged 6 implementation approaches (direct import, fork, WASM bridge, shell out, etc.)
- Mapped the broader Go ASCII/graph library ecosystem
- Found no competing Go solutions

### Round 2: Verification and Gap Analysis
- Corrected star count (1,291, not ~291)
- Confirmed pgavlin's fork has 22 separate rendering packages (not just detection)
- Found D2 (terrastruct) as alternative architectural approach (ELK layout downscaling)
- Identified concrete edge routing limitations via GitHub issues (#30, #56, #25, #46)
- Confirmed sammcj/go-mermaid parser is incompatible with mermaid-ascii internals

### Round 3: Quality and Credibility Assessment
- **pgavlin/mermaid-ascii code quality is strong**: 87.6% test coverage, 63+ test cases sourced from official Mermaid.js docs, unified recursive descent parser, width constraints, multi-line labels
- **AlexanderGrooff library API confirmed awkward**: rendering lives in `cmd/` package (`cmd.RenderDiagram(mermaid, nil)`), issue #50 still open
- **pgavlin's fork specifically fixed the API problem**: moved to `pkg/render/` with clean `Render(input, config)` signature
- **Developer credibility is high**: Pat Gavlin is a Staff Software Engineer at Pulumi (6 years), ex-Microsoft .NET CLR team, 130 GitHub repos, 84 followers, Arctic Code Vault Contributor, author of performance-critical Go libraries (aho-corasick 20x faster than Cloudflare's)
- **Cursor CLI likely uses beautiful-mermaid (TS)**, not the Go tool -- Cursor is Electron/Node.js
- **34+ commits ahead of upstream** with substantial architectural improvements

---

## Definitive Recommendation

### Primary: pgavlin/mermaid-ascii

**Use pgavlin/mermaid-ascii as your Go library.**

Reasoning:
1. **Clean library API** -- `render.Render(input, config)` is exactly what you want for embedding. The original's `cmd.RenderDiagram` API is a known pain point with no fix planned.
2. **22 diagram types** with dedicated rendering packages, not stubs.
3. **87.6% test coverage** with 63+ test cases from official Mermaid.js documentation -- this is not toy code.
4. **Credible maintainer** -- a senior systems engineer with compiler/runtime background and a track record of high-quality Go libraries.
5. **Active development** -- unified parser architecture, width constraints, and multi-line labels added in March 2026.
6. **Pure Go, 7 dependencies** -- lightweight and embeddable.
7. **MIT license** (inherited from upstream).

The main risk is that this is a new fork with zero community adoption. However, round 3 findings significantly mitigate this: the test coverage, developer credibility, and code quality indicators are all strong.

### Fallback: AlexanderGrooff/mermaid-ascii

If you only need flowcharts and sequence diagrams and want maximum community validation, use the original. Be aware the library API is not clean -- you will either import `cmd.RenderDiagram` or shell out to the CLI binary.

### Not Recommended

- **WASM bridge to beautiful-mermaid**: Adds enormous complexity for marginal benefit over pgavlin's fork.
- **sammcj/go-mermaid + custom renderer**: Parser-only; you would need to build all rendering from scratch. Layout algorithms are the hard part.
- **D2 transpiler approach**: Requires Mermaid-to-D2 syntax conversion. Interesting architecturally (ELK layout) but impractical as a primary path.
- **Shell out to beautiful-mermaid**: Adds Node.js runtime dependency. Only justified if you need the TypeScript library's theming/SVG features.

---

## Remaining Unknowns

1. **Rendering quality for non-graph types**: No one has verified ASCII output quality for pie charts, Sankey diagrams, mindmaps, or git graphs in pgavlin's fork. These are inherently visual and may produce mediocre ASCII. Test with your actual inputs.

2. **Performance benchmarks**: No rendering speed data exists for any Go tool on large diagrams. beautiful-mermaid claims 100+ diagrams in <500ms; no Go equivalent benchmark exists.

3. **Output quality comparison**: No side-by-side comparison of beautiful-mermaid vs. pgavlin/mermaid-ascii for the same input diagrams. This would be valuable for diagram types both support.

4. **Upstream merge potential**: Whether pgavlin's improvements will merge back into AlexanderGrooff's repo is unknown. The fork may diverge permanently.

5. **Edge routing quality at scale**: The A* pathfinding approach has known limitations (no edge bundling, fixed grid granularity). Behavior on graphs with 15+ nodes and complex edge patterns is untested.

---

## Next Steps

1. **Hands-on evaluation** (30 min): Clone pgavlin/mermaid-ascii. Run `render.Render()` against 5-10 representative Mermaid diagrams from your actual use case. Assess output quality visually.

2. **Integration prototype** (1-2 hours): Write a small Go program that imports `github.com/pgavlin/mermaid-ascii/pkg/render` and renders flowcharts and sequence diagrams. Confirm the API works as documented and the output meets your quality bar.

3. **Test non-graph types** (30 min): If you need Gantt, class, ER, or state diagrams, test those specifically. Deprioritize pie/Sankey/mindmap unless you actually need them -- these are inherently hard to render as ASCII.

4. **Pin the dependency**: Since there are no releases yet, pin to a specific commit hash in your go.mod. Monitor the repo for tagged releases.

5. **Contribute upstream**: If you find bugs or want improvements, contribute back. The maintainer has a track record of active development and the codebase is well-structured.

---

## Research Quality Rating: 8/10

**Strengths:**
- Comprehensive landscape mapping across all 3 rounds with no competing Go solution missed
- Critical correction of star count data between rounds
- Round 3 credibility and code quality analysis resolved the key "is pgavlin's fork trustworthy?" question
- Concrete GitHub issue numbers, API signatures, and dependency counts
- Clear lineage tracing (Go -> TS -> Go)

**Weaknesses:**
- No hands-on output quality verification in any round (the single biggest gap)
- No performance benchmarks
- D2 alternative was identified but not deeply explored
- Round 3 Cursor CLI finding is "likely" not "confirmed"

The research is strong enough to make a confident recommendation. The remaining unknowns are all resolvable in under 2 hours of hands-on testing, which is the appropriate next step.
