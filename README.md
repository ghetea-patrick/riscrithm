# Riscrithm v1.0.0 - Initial Release

Riscrithm is a lightweight, hardware-aware macro-assembler and optimizing compiler pipeline for RISC-V written in Go. It bridges high-level semantic conveniences with bare-metal control, featuring a clean text-substitution preprocessor and a multi-tier architectural optimization pass.

## Key Features

### 1. Directives & System Layouts
* `header <type>`: Emits raw section headers (e.g., `header default` maps directly to `.section .text`).
* `entrypoint <label>`: Explicitly defines the global execution vector.
* High-level architectural abstractions mapping straight to core pipeline routines:
  * `interrupt.u` / `interrupt.s` / `interrupt.m` -> `uret` / `sret` / `mret`
  * `wait` -> `wfi` (Wait for Interrupt low-power state)
  * `trap` -> `ebreak` (Debugger breakpoint)
  * `halt` -> `ecall` (System call environment trap)

### 2. High-Speed Macro Preprocessor
* `define <macro> = <expansion>`: Implements a single-pass text-substitution pipeline.
* Independent scopes omit nested evaluation, completely eliminating cyclic dependency loops and guaranteeing predictable, linear compile times.

### 3. Syntax Shorthands & Unary Mutations
* Translates compound expressions (e.g., `+=`, `-=`, `<<=`, `|=`, `%=`) into standard algebraic three-operand formats.
* Normalizes traditional unary loops (`++`, `--`) down to baseline addition/subtraction.
* Features a dedicated zeroing syntax: `reg ^^` compiles as a self-targeted bitwise XOR (`xor reg, reg, reg`) for rapid, zero-overhead register clearing.

### 4. Instruction Selection & Lowering Engine
* Intelligent immediate parsing automatically switches instructions based on type (e.g., shifting between `add` and `addi`, or mapping variable configurations straight to `li` / `mv`).
* `reg1 swap reg2`: Intercepts target swaps and lowers them into an optimized 3-step bitwise XOR sequence, preventing register spilling or stack thrashing without an intermediate temporary register.
* `if reg1 [op] reg2 @true else @false`: Evaluates relational conditions directly to conditional hardware branches (`beq`, `bne`, `blt`, `bge`). Unsupported directions (`>` and `<=`) are corrected via an automatic operand position flip.

### 5. Memory Architecture & Pointer Handling
* **Stack Pipeline:** Abstracted routines (`-> stack`, `<- stack`, `= stack`) automatically compute data sizes and handle explicit Stack Pointer (`sp`) tracking for Byte (`.b`), Word (`.w`), and Doubleword (`.d`) allocations.
* **Heap Accessors:** Simple base-offset reference operations (`<- heap` / `-> heap`) map straight to zero-offset memory index queries (e.g., `lw dest, 0(addressReg)`).

### 6. Code Blocks & Hardware Safety Escape Hatches
* **The Ellipsis (`...`) Token:** Explicitly shielded from the optimization engine. It forces an untouchable path straight to raw execution delays (`nop`) to respect physical bare-metal timing loops.
* **Inline Injection (`!!`):** Functions as a raw code escape hatch. Any label prefixed or wrapped with `!!` bypasses the parser entirely, passing straight through to the output stream untouched for fine-grained manual control.
* **Strict Labelling Constraints:** To ensure clean structural control flow, block labels require explicit tagging (`@label`), preventing branch targets from getting tangled up with standard register variables or system shorthand macros.

### 7. Pipeline Optimizations (`-o` / `--optimize`)
* **Dead-Store Elimination:** Scrubs out redundant, sequential load or move instructions targeting the exact same destination register.
* **Algebraic Identity Minimization:** Analyzes math blocks to eliminate zero-effect operations (adding/subtracting 0, or multiplying/dividing by 1).
* **Strength Reduction:** Monitors multiplication and division by power-of-two constants (2^n) and dynamically swaps heavy algebraic calculations out for ultra-fast, single-cycle logical bitwise shifts (`<<` / `>>`).
