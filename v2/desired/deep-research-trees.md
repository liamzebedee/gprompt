```
# ---- Prompt definitions ----

quantify(question) -> {metric, baseline, target, unknowns}:
  Take this research question and rewrite it as a precise, 
  measurable claim.
  
  Question: [question]
  
  Return:
  - metric: what's measured, with units
  - baseline: current number with source
  - target: goal number
  - unknowns: what numbers you don't have

decompose(question) -> list:
  Identify the 3-5 independent physical processes that 
  determine [question.metric].
  
  For each, state:
  - name
  - description (one sentence)
  - bottleneck (what limits its rate)
  - rate (order of magnitude number with units)
  - near_max (true/false)

anchor(subproblem) -> {claim, value, units, confidence, source}:
  Find the specific measured quantity for this:
  
  [subproblem.name]: [subproblem.bottleneck]
  
  Return:
  - claim: what you're measuring
  - value: the number
  - units: the units
  - confidence: MEASURED, ESTIMATED, or GUESSED
  - source: where this number comes from
  
  If you cannot find a reliable number, confidence = GUESSED.
  Do not fabricate precision.

sub_decompose(subproblem) -> list:
  This subproblem has unknowns that need deeper investigation:
  
  [subproblem.name]: [subproblem.bottleneck]
  
  Break it into 2-4 more specific sub-questions that, 
  if answered, would resolve the uncertainty.
  
  For each, state:
  - name
  - description
  - bottleneck
  - rate (if known)
  - near_max (if known)

synthesize(question, tree) -> report:
  You are compiling a research analysis.
  
  Original question: [question]
  
  Here is the full knowledge tree with anchored measurements 
  at every leaf:
  [tree]
  
  Synthesize this into:
  - A 2-paragraph summary of the binding constraints
  - Ranked list of intervention points (best first)
  - The research frontier: what remains GUESSED or hit depth limit

# ---- The engine ----

investigate(question) -> report:
  q = quantify(question)
  
  deep_decompose(q, depth=3) -> tree:
    divide:
      decompose(q) if depth == 3
      sub_decompose(subproblem) if depth < 3
    conquer(subproblem):
      a = anchor(subproblem)
      if a.confidence == "GUESSED" and depth > 0:
        deep_decompose(subproblem, depth=depth-1)
      else:
        -> a
    combine(results):
      -> results

  report = synthesize(q, tree)
  -> report

# ---- Run ----

@investigate("How do we grow a tree 100x faster")
```


The output tree is just the combine results rendered. Every leaf is either an anchored measurement or a depth-limit marker. The runtime can print it as that tree diagram natively because it knows the recursive structure — each deep_decompose call is a node, each anchor result is a leaf.

```
investigate("How do we grow a tree 100x faster")
│
├─ quantify(...)                                    # 1 LLM call
│  -> {metric: "dry biomass kg/yr", baseline: 20, target: 2000}
│
├─ deep_decompose(q, depth=3)
│  │
│  ├─ divide: decompose(q)                          # 1 LLM call
│  │  -> [light_capture, carbon_fixation, cambial_division, ...]
│  │
│  ├─ conquer (all in parallel):                    # N parallel LLM calls
│  │  │
│  │  ├─ anchor(light_capture)
│  │  │  -> {confidence: GUESSED}                   # GUESSED → recurse
│  │  │  │
│  │  │  └─ deep_decompose(light_capture, depth=2)
│  │  │     ├─ divide: sub_decompose(...)           # 1 LLM call
│  │  │     │  -> [leaf_area, chlorophyll_eff, canopy_arch]
│  │  │     ├─ conquer (parallel):                  # 3 parallel LLM calls
│  │  │     │  ├─ anchor(leaf_area)       -> MEASURED, done
│  │  │     │  ├─ anchor(chlorophyll_eff) -> MEASURED, done
│  │  │     │  └─ anchor(canopy_arch)     -> GUESSED → recurse
│  │  │     │     │
│  │  │     │     └─ deep_decompose(canopy_arch, depth=1)
│  │  │     │        ├─ divide: sub_decompose(...)  # 1 LLM call
│  │  │     │        ├─ conquer (parallel):         # 2 parallel LLM calls
│  │  │     │        │  ├─ anchor(leaf_angle)    -> MEASURED, done
│  │  │     │        │  └─ anchor(self_shading)  -> MEASURED, done
│  │  │     │        └─ combine -> results
│  │  │     └─ combine -> results
│  │  │
│  │  ├─ anchor(carbon_fixation)
│  │  │  -> {confidence: GUESSED}                   # GUESSED → recurse
│  │  │  │
│  │  │  └─ deep_decompose(carbon_fixation, depth=2)
│  │  │     ├─ divide: sub_decompose(...)
│  │  │     ├─ conquer (parallel):
│  │  │     │  ├─ anchor(rubisco_rate)    -> MEASURED, done
│  │  │     │  └─ anchor(c4_vs_c3)       -> GUESSED → recurse
│  │  │     │     │
│  │  │     │     └─ deep_decompose(c4_vs_c3, depth=1)
│  │  │     │        ├─ conquer (parallel):
│  │  │     │        │  ├─ anchor(kranz_anatomy)         -> GUESSED, depth limit, stop
│  │  │     │        │  └─ anchor(synthetic_fixation)    -> MEASURED, done
│  │  │     │        └─ combine -> results
│  │  │     └─ combine -> results
│  │  │
│  │  ├─ anchor(cambial_division) -> ...
│  │  └── ...
│  │
│  └─ combine -> tree
│
├─ synthesize(q, tree)                              # 1 LLM call
│  -> final report
```

Ideally this looks simpler to the user, something like:

The runtime builds an execution tree that might look like:

```
"How to grow trees 100x faster"
├── light capture [cap: 8x, not binding]
│   ├── leaf area limits (MEASURED: 5-8 m² typical canopy)
│   ├── chlorophyll efficiency (MEASURED: 1-2% of incident)
│   └── canopy architecture (ESTIMATED)
│       ├── optimal leaf angle distribution (MEASURED)
│       └── self-shading models (MEASURED)
├── carbon fixation [cap: 5x, BINDING]          ← the constraint
│   ├── RuBisCO rate (MEASURED: 3-10 s⁻¹)
│   └── C4 vs C3 pathways (GUESSED)              ← research frontier
│       ├── Kranz anatomy in wood (GUESSED)       ← frontier
│       └── synthetic carbon fixation (MEASURED)
├── sugar transport [cap: 50x, not binding]
│   └── phloem flow rate (MEASURED: 0.5-1.0 m/hr)
├── cambial cell division [cap: 15x, not binding]
│   ├── cell cycle time (ESTIMATED: 12-72 hrs)    ← wide range, dig?
│   └── hormonal regulation (GUESSED)             ← frontier
├── cell wall synthesis [cap: 30x, not binding]
│   └── cellulose synthase rate (MEASURED: 300-1000 glucose/hr)
│
├── CONJECTURES
│   ├── ✓ engineer C4 into trees (survived 2 rounds)
│   ├── ✓ bioreactor wood (damaged: no crystalline cellulose)
│   ├── ✗ CO₂ enrichment (killed: math caps at 1.5x)
│   └── ✓ replace RuBisCO (survived 1 round)
│
└── FRONTIER (what to investigate next)
    1. Can Kranz anatomy develop in woody tissue?
    2. What controls cambial cell cycle rate?
    3. Has anyone achieved crystalline cellulose in culture?
```