Take a big output, too big for a context window, recursively divide it until one context window : 1 work item mapping.

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