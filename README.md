# Riscrithm

### A macro-assembly compiler with slightly fewer bare-metal-related inconveniences.

> [!CAUTION]
> **Riscrithm v2.0 is currently under active development.**
>
> The core compiler is being rebuilt from the ground up to provide major performance and developer-experience improvements.
>
> **Backwards compatibility is intentionally broken in v2.0.** Most v1.1 scripts will **not** be compatible with v2.0 without modification.
>
> If you have active Riscrithm projects, keep them pinned to **v1.1** for now.
>
> The v2.0 syntax is now finalized and active compiler development has officially started.

Welcome to **Riscrithm**, a high-level macro-assembly dialect that compiles directly to pure RISC-V assembly.

Riscrithm is built to act as a bridge between the readability of a high-level language and the raw, deterministic control of bare-metal hardware.

Riscrithm provides several conveniences:

- **File modularity** for reusable packages and utility libraries.
- **Cleaner control flow** with inline conditionals and subroutine returns.
- **Strict compile-time validation** for structural safety.
- **Optimization passes** for dead code and identity math elimination.
- **Expressive system and interrupt controls** replacing raw opcodes.
- **Automatic stack pointer management** and heap memory operations.
- **Readable assignment, compound math, and bitwise mutators.**
- **Raw assembly bypassing** for specialized hardware instructions.

Riscrithm intentionally keeps its abstractions transparent while making common bare-metal operations easier to express.

It is designed for programmers who want to write efficient hardware instructions without repeatedly typing manual XOR swaps and tracking exact stack pointer offsets.

Because apparently writing `xor x1, x1, x2` three times in a row was considered a highly productive way to spend an afternoon.

---

## 1. Installation and CLI

Riscrithm is used through its command-line compiler.

To compile a Riscrithm source file, use:

```text
riscrithm "source_code_file" "assembly_target_file" [-o/--optimize]
```

The arguments are:

| Argument | Description |
|---|---|
| `source_code_file` | The Riscrithm input file. |
| `assembly_target_file` | The generated `.s` assembly file. If the file does not exist, the compiler creates it automatically. |
| `-o`, `--optimize` | Enables the comprehensive optimization pass. |

For example:

```text
riscrithm "main.txt" "main.s"
```

Or with optimization enabled:

```text
riscrithm "main.txt" "main.s" --optimize
```

The compiler produces standard RISC-V assembly that can then be passed to the appropriate assembler, linker, simulator, or hardware toolchain.

---

## 2. File Structure and Entrypoints

Every main Riscrithm source file must declare its target section and entrypoint at the top of the file.

These directives, together with macro definitions and import statements, are the only lines allowed to exist completely unindented outside of a label block.

### Header

The `header` directive selects the target assembly section.

```text
header default
```

For example:

```text
header default
```

translates to:

```asm
.section .text
```

### Entrypoint

The `entrypoint` directive defines where program execution begins.

```text
entrypoint main
```

This translates to:

```asm
.globl main
```

A minimal Riscrithm program therefore begins with:

```text
header default
entrypoint main
```

---

## 3. Imports and Modular Files

Riscrithm supports modular source files, making it possible to split projects into reusable packages and utility libraries.

### 3.1 Global Imports

Use `import` to include an entire source file in the current compilation unit.

```text
import "packages/display_utilities.txt"
```

The imported file becomes part of the unified compilation unit.

### 3.2 Selective Imports

Use `from ... import` to import only specific label symbols from another source file.

```text
from "libraries/math_helpers.txt" import qux, quux
from "libraries/math_helpers.txt" import corge
```

Selective imports allow individual labels to be reused without importing every symbol from the source file.

### Important Rule

Imported modules and secondary source files must **not** contain a `header` or `entrypoint` directive.

These files function strictly as modular components.

Only the main compilation unit should define the target section and program entrypoint.

---

## 4. Definitions and Comments

### 4.1 Macros

Text-replacement macros are declared using the `define` keyword.

Macros are useful for aliasing registers, creating constants, or defining short inline code fragments.

```text
define foo = x1
define bar = x2
define baz = x3
define qux = x4
define quux = 10
define corge = 20
define clearFoo = foo ^^
```

Whenever the parser encounters `foo`, it replaces it with `x1` before processing logical expressions.

For example:

```text
define foo = x1

main:
    load foo = 10
```

is processed as though the register alias had been written directly.

### 4.2 Comments

Comments are written using the `#` symbol.

The compiler strips everything following a `#` on a line.

```text
load foo = 10 # Initialize foo
```

Comments can therefore be placed inline without affecting generated assembly.

---

## 5. Compiler Validation

Writing bare-metal assembly can be error-prone and tedious to debug.

Riscrithm performs a compile-time validation pass before generating assembly.

The compiler checks for several structural problems and stops compilation when invalid code is detected.

### Missing Header

The main source file must contain a `header` directive.

Without a header, the compiler cannot initialize the target assembly section.

### Invalid Entrypoint

The symbol specified by `entrypoint` must resolve to a valid, defined label.

### Global Duplicate Labels

All unified modules are scanned to ensure that no label is declared more than once.

### Undefined Jumps and Branches

Any jump or branch targeting a label that does not exist causes compilation to fail.

### Unreachable Code

Basic control-flow analysis detects dead code following terminating instructions such as `return`.

For example:

```text
example:
    return
    foo ++
```

The instruction after `return` is unreachable.

### Duplicate File Imports

Importing the exact same file path multiple times is detected and reported.

### Duplicate Label Imports

Importing the same specific label more than once causes a syntax validation failure.

These validation checks are performed before assembly output is generated.

---

## 6. Labels, Indentation, and Raw Blocks

Riscrithm uses indentation to define execution blocks.

### 6.1 Standard Labels

Labels define execution blocks and must end with a colon.

Labels must not be indented.

Instructions inside a label block must be indented using spaces or tabs.

```text
main:
    load foo = quux
    move bar = foo
```

Leaving an instruction unindented outside of a valid block results in a `SyntaxError`.

### 6.2 Raw Assembly Labels

To completely bypass the Riscrithm preprocessor and write raw RISC-V assembly, prefix a label with `!!`.

```text
!!raw_block:
    li x1, 10
    variable ^^ # This stays exactly as written!
```

The compiler removes the `!!` prefix from the label and passes the contents of the block through without processing.

Macros and Riscrithm shorthands are not expanded inside raw blocks.

### 6.3 Inline Raw Assembly

If only a single instruction needs to bypass the compiler, use the `!!` prefix directly on an indented instruction.

```text
process_data:
    load foo = 5
    !!addi x1, x1, 10
    foo ++
```

The prefixed instruction is emitted directly as raw assembly.

This is useful for specialized instructions that are not represented by the Riscrithm syntax.

---

## 7. System and Interrupt Controls

Riscrithm provides readable keywords for common system and interrupt instructions.

| Riscrithm | RISC-V Assembly | Description |
|---|---|---|
| `interrupt.u` | `uret` | User-mode trap return |
| `interrupt.s` | `sret` | Supervisor-mode trap return |
| `interrupt.m` | `mret` | Machine-mode trap return |
| `wait` | `wfi` | Wait for interrupt |
| `trap` | `ebreak` | Debugger trap |
| `halt` | `ecall` | System environment call / halt |
| `...` | `nop` | No-operation |

### Example

```text
handle_system_events:
    wait
    interrupt.u
```

These keywords provide readable names for common low-level system operations without requiring raw opcode syntax.

---

## 8. Branching and Conditionals

Riscrithm handles common branch mappings automatically.

### 8.1 Unconditional Jumps

To unconditionally jump to a label, use the `@` prefix.

```text
execute_jump:
    @some_label
```

This compiles to:

```asm
j some_label
```

### 8.2 Conditional Branches

Conditional branches use an inline ternary-like syntax.

```text
compare_registers:
    if foo == bar @true_block else @false_block
    if foo > baz @greater_block else @lesser_block
```

The compiler maps supported comparisons to the appropriate RISC-V branch instructions such as:

- `beq`
- `bne`
- `blt`
- `bge`

The compiler also handles register ordering for asymmetric comparisons such as `>` and `<=`.

### 8.3 Else-less If Statements

Conditional branches do not require an `else` target.

```text
guard_check:
    if foo == bar @true_block
```

This is useful for guard clauses and simple conditional jumps.

---

## 9. Subroutines and Loops

Riscrithm avoids high-level `while` and `for` constructs in order to preserve bare-metal transparency.

Instead, loops are constructed using labels, jumps, and conditional branches.

### 9.1 Loops

#### Infinite Loop

```text
infinite_loop:
    foo ++
    @infinite_loop
```

#### Conditional Loop

```text
loop_setup:
    load foo = 0
    load bar = 10

loop_start:
    if foo == bar @loop_end else @loop_body

loop_body:
    foo ++
    @loop_start

loop_end:
    halt
```

This keeps the generated control flow explicit and predictable.

### 9.2 Subroutine Returns

Labels can act as reusable subroutines using the `return` statement.

```text
multiply_logic:
    foo *= bar
    return
```

The `return` statement compiles directly to the native RISC-V `ret` instruction.

---

## 10. Operations and Mutators

Riscrithm supports immediate assignments and compound mathematical expressions.

When an integer literal is used where an immediate instruction is available, the compiler automatically selects the appropriate `i`-suffixed RISC-V instruction, such as `addi` or `xori`.

### 10.1 Load and Move

```text
assign_values:
    load foo = 100
    move bar = foo
```

These correspond to:

```asm
li foo, 100
mv bar, foo
```

### 10.2 Compound Math

```text
apply_math:
    foo += 5
    bar *= baz
    foo <<= 2
```

The compiler maps these expressions to the corresponding RISC-V instructions.

### 10.3 Increments and Decrements

```text
adjust_counters:
    foo ++
    bar --
```

For example:

```text
foo ++
```

compiles to:

```asm
addi foo, foo, 1
```

and:

```text
bar --
```

compiles to:

```asm
addi bar, bar, -1
```

### 10.4 The Clear Shorthand (`^^`)

The `^^` operator clears a register by XORing it with itself.

```text
reset_state:
    foo ^^
```

This translates to:

```asm
xor foo, foo, foo
```

The resulting register value is zero.

### 10.5 Swapping Variables

The `swap` command exchanges the contents of two registers without requiring a third temporary register.

```text
perform_swap:
    foo swap bar
```

This translates to:

```asm
xor foo, foo, bar
xor bar, foo, bar
xor foo, foo, bar
```

No temporary register required.

Because apparently three XOR instructions are preferable to admitting that temporary storage exists.

---

## 11. Memory Operations

Riscrithm provides explicit stack and heap expressions for memory operations.

Memory operations use data-width indicators:

- `.b` for byte / 8-bit.
- `.w` for word / 32-bit.
- `.d` for double-word / 64-bit.

### 11.1 Stack Operations

Stack expressions automatically manage the hardware stack pointer (`sp`) according to the selected data width.

#### Push (`->`)

Push operations decrement `sp` by the relative width before storing the register value.

```text
save_context:
    foo -> stack.w
```

For a word-sized operation, the stack pointer is decremented by 4 bytes before storing the value.

#### Pop (`<-`)

Pop operations load from the current stack address and then increment `sp` by the relative width.

```text
restore_context:
    bar <- stack.d
```

A double-word operation therefore increments `sp` by 8 bytes after loading.

#### Peek (`=`)

The `=` stack expression reads from the current stack address without modifying `sp`.

```text
check_top:
    baz = stack.b
```

This loads a byte from the top of the stack without moving the stack pointer.

### 11.2 Heap Operations

Heap operations require an explicit base address register using `&` pointer notation.

#### Store (`->`)

```text
write_memory:
    foo -> heap.w from &bar
```

This stores the value in `foo` at the address contained in `bar`.

#### Load (`<-`)

```text
read_memory:
    baz <- heap.b from &foo
```

This loads a byte from the address contained in `foo` into `baz`.

---

## 12. The Optimizer (`-o`)

Riscrithm uses a lightweight two-pass compiler pipeline.

### Pass 1: Sanitization and Validation

The first pass:

1. Evaluates module imports.
2. Resolves source layout.
3. Performs structural validation.
4. Checks labels and control-flow targets.
5. Prepares the unified compilation unit.

### Pass 2: Parsing and Optimization

The second pass:

1. Replaces macros.
2. Processes Riscrithm shorthands.
3. Applies enabled optimizations.
4. Generates RISC-V assembly.

When `-o` or `--optimize` is supplied, the optimizer applies several transformations.

### 12.1 Dead Assignment Elimination

Consecutive redundant assignments or useless load/move operations targeting the same register are removed.

#### Source Input

```text
load foo = 128
load foo = 128
```

#### Optimized Output

```asm
li x1, 128
```

### 12.2 Identity Math Elimination and Transformation

Mathematical expressions that do not change a value are optimized according to the destination context.

#### Self-Identity Elimination

If a register performs an identity operation on itself, the instruction can be removed.

#### Source Input

```text
foo = foo + 0
bar = bar * 1
```

#### Optimized Output

```text
# Instructions deleted entirely
```

#### Cross-Register Identity Transformation

When the source and destination registers differ, the expression can be converted into a register copy using `mv`.

#### Source Input

```text
foo = bar + 0
foo = bar * 1
```

#### Optimized Output

```asm
mv x1, x2
mv x1, x2
```

### 12.3 Strength Reduction

Multiplication or division by a constant power of two can be rewritten as a logical bit shift.

#### Source Input

```text
foo = bar * 2
baz = foo / 8
```

#### Optimized Output

```asm
slli x1, x2, 1
srli x3, x1, 3
```

This replaces multiplication and division with equivalent shift operations where the transformation is applicable.

---

## 13. Clean, Ready-to-Use Output

The assembly generated by Riscrithm is automatically formatted for readability.

The resulting `.s` file is pretty-printed so that:

- Instructions inside execution blocks are consistently indented.
- Labels remain flush against the left margin.
- Generated instructions remain easy to read.
- The output can be passed directly to standard RISC-V development tools.

The generated assembly can therefore be used with hardware simulators, assemblers, binary toolchains, linkers, and desktop debuggers without requiring manual formatting.

---

## 14. Naming Conventions

Riscrithm encourages consistent naming conventions to keep projects easy to scan.

### Variables and Registers

Use **camelCase** for register aliases and dynamic variables.

Examples:

```text
firstNum
addressRegister
stackOffset
```

### Labels and Code Blocks

Use **snake_case** for jump locations, execution blocks, loop boundaries, and subroutines.

Examples:

```text
loop_start
on_true
error_handler
```

### Constants and Literals

Use **SCREAMING_SNAKE_CASE** for static configuration values, macro constants, and invariant boundaries.

Examples:

```text
DEFAULT_HEADER
MAX_BUFFER_SIZE
IMM_VALUE
```

These conventions provide a visual distinction between registers, control-flow targets, and constants.

---

## 15. Complete Operator and Expression Reference

### 15.1 Core Expressions and Memory Operators

| Riscrithm Syntax | Category | Internal Expansion / Behavior | Target RISC-V Assembly |
|---|---|---|---|
| `load <reg> = <imm>` | Assignment | Direct immediate assignment | `li reg, imm` |
| `move <reg1> = <reg2>` | Assignment | Register-to-register copy | `mv reg1, reg2` |
| `<reg1> swap <reg2>` | Value Exchange | Triple-XOR non-destructive swap | `xor reg1, reg1, reg2`<br>`xor reg2, reg1, reg2`<br>`xor reg1, reg1, reg2` |
| `<reg> -> stack.[b/w/d]` | Stack Memory | Decrement pointer, store byte/word/double | `addi sp, sp, -offset`<br>`s[b/w/d] reg, 0(sp)` |
| `<reg> <- stack.[b/w/d]` | Stack Memory | Load byte/word/double, increment pointer | `l[b/w/d] reg, 0(sp)`<br>`addi sp, sp, offset` |
| `<reg> = stack.[b/w/d]` | Stack Memory | Peek value from top of stack | `l[b/w/d] reg, 0(sp)` |
| `<reg1> <- heap.[b/w/d] from &<reg2>` | Heap Memory | Base-register memory read | `l[b/w/d] reg1, 0(reg2)` |
| `<reg1> -> heap.[b/w/d] from &<reg2>` | Heap Memory | Base-register memory write | `s[b/w/d] reg1, 0(reg2)` |

### 15.2 Math and Bitwise Operators

| Riscrithm Syntax | Operator Type | Evaluated Expression |
|---|---|---|
| `<reg> ++` | Self Operator | `<reg> = <reg> + 1` |
| `<reg> --` | Self Operator | `<reg> = <reg> - 1` |
| `<reg> ^^` | Self Operator | `<reg> = <reg> ^ <reg>` (Fast Register Clear) |
| `<reg> += <val>` | Compound Tag | `<reg> = <reg> + <val>` |
| `<reg> -= <val>` | Compound Tag | `<reg> = <reg> - <val>` |
| `<reg> *= <val>` | Compound Tag | `<reg> = <reg> * <val>` |
| `<reg> /= <val>` | Compound Tag | `<reg> = <reg> / <val>` |
| `<reg> %= <val>` | Compound Tag | `<reg> = <reg> % <val>` |
| `<reg> <<= <val>` | Compound Tag | `<reg> = <reg> << <val>` |
| `<reg> >>= <val>` | Compound Tag | `<reg> = <reg> >> <val>` |
| `<reg1> = <reg2> + <val>` | Base Arithmetic | Addition with immediate realignment |
| `<reg1> = <reg2> - <val>` | Base Arithmetic | Subtraction with immediate realignment |
| `<reg1> = <reg2> & <val>` | Base Arithmetic | Bitwise AND with immediate realignment |
| `<reg1> = <reg2> \| <val>` | Base Arithmetic | Bitwise OR with immediate realignment |
| `<reg1> = <reg2> ^ <val>` | Base Arithmetic | Bitwise XOR with immediate realignment |
| `<reg1> = <reg2> << <val>` | Base Arithmetic | Logical shift left with immediate realignment |
| `<reg1> = <reg2> >> <val>` | Base Arithmetic | Logical shift right with immediate realignment |
| `<reg1> = <reg2> * <val>` | Base Arithmetic | Hardware multiplication (M-extension) |
| `<reg1> = <reg2> / <val>` | Base Arithmetic | Hardware division (M-extension) |
| `<reg1> = <reg2> % <val>` | Base Arithmetic | Hardware remainder (M-extension) |

---

## 16. Complete Examples

### 16.1 Source Code

#### Example One

```text
header default
entrypoint main

define foo = x1
define bar = x2
define baz = x3

main:
    load foo = 10
    load bar = 20

    foo += 5
    bar += 2

    baz = foo + bar

    halt
```

#### Example Two: Memory and Swapping

```text
header default
entrypoint main

define foo = x1
define bar = x2

main:
    load foo = 42
    load bar = 99

    foo -> stack.w
    bar -> stack.w

    foo <- stack.w
    bar <- stack.w

    foo swap bar

    halt
```

#### Example Three: Math and Conditionals

```text
header default
entrypoint main

define foo = x1
define bar = x2

main:
    load foo = 7
    load bar = 7

    foo = foo * 1
    bar = bar + 0

    foo = bar * 1

    foo *= 8
    bar /= 2

    if foo == bar @equal_block else @not_equal_block

equal_block:
    foo ++
    halt

not_equal_block:
    foo --
    trap
```

#### Example Four: Modularity

`orange_banana.txt`

```text
!!banana:
    addi x1, x1, 10
    ret

orange:
    bar >>= 1
    bar &= 10
    return

apple:
    if foo < bar @banana
```

`horse_battery.txt`

```text
qux:
    foo swap bar
    foo ^^

quux:
    !!li x1, 10
    bar swap foo
    bar --
    foo ++
```

`main.txt`

```text
header default
entrypoint main

import "libraries/orange_banana.txt"
from "libraries/horse_battery.txt" import qux, quux

define foo = x1
define bar = x2
define baz = x3

main:
    load foo = qux
    load bar = quux

    foo += bar
    foo -> heap.w from &baz

    halt

    if foo == bar @qux else @quux

    @orange
    ...
    @banana
```

---

### 16.2 Optimized RISC-V Assembly Output

#### Example One

```asm
.section .text
.globl main
main:
   li x1, 10
   li x2, 20
   addi x1, x1, 5
   addi x2, x2, 2
   add x3, x1, x2
   ecall
```

#### Example Two

```asm
.section .text
.globl main
main:
   li x1, 42
   li x2, 99
   addi sp, sp, -4
   sw x1, 0(sp)
   addi sp, sp, -4
   sw x2, 0(sp)
   lw x1, 0(sp)
   addi sp, sp, 4
   lw x2, 0(sp)
   addi sp, sp, 4
   xor x1, x1, x2
   xor x2, x1, x2
   xor x1, x1, x2
   ecall
```

#### Example Three: Strength Reduction and Identity Math Elimination

```asm
.section .text
.globl main
main:
   li x1, 7
   li x2, 7
   mv x1, x2
   slli x1, x1, 3
   srli x2, x2, 1
   beq x1, x2, equal_block
   j not_equal_block

equal_block:
   addi x1, x1, 1
   ecall

not_equal_block:
   addi x1, x1, -1
   ebreak
```

#### Example Four

```asm
.section .text
.globl main
main:
   load x1 = qux
   load x2 = quux
   add x1, x1, x2
   sw x1, 0(x3)
   ecall
   beq x1, x2, qux
   j quux
   j orange
   nop
   j banana

banana:
   addi x1, x1, 10
   ret

orange:
   srli x2, x2, 1
   andi x2, x2, 10
   ret

apple:
   blt x1, x2, banana

qux:
   xor x1, x1, x2
   xor x2, x1, x2
   xor x1, x1, x2
   xor x1, x1, x1

quux:
   li x1, 10
   xor x2, x2, x1
   xor x1, x2, x1
   xor x2, x2, x1
   addi x2, x2, -1
   addi x1, x1, 1
```

---

## 17. Design Philosophy

Riscrithm is designed to provide a readable, high-level interface over raw RISC-V assembly generation.

The compiler deliberately bridges readable expressions such as:

```text
if foo == bar
foo += 5
```

to their corresponding hardware instructions such as:

```asm
beq
addi
```

without introducing a magical runtime environment between the source code and the processor.

The architecture separates responsibilities into logical stages:

1. **Modularity and file unification.**
2. **Structural validation and safety checks.**
3. **Preprocessor definitions.**
4. **Instruction parsing and optimization.**

Riscrithm does not attempt to replace LLVM infrastructure, C compilers, or full-scale language toolchains.

It simply makes writing pure bare-metal assembly less tedious to construct by hand.

Because apparently memorizing manual branch destinations and manually performing bit-shift reductions was not enough of a headache.

---

## 18. Limitations

Riscrithm intentionally remains a lightweight abstraction over bare RISC-V assembly.

The current **v1.1 implementation** does not provide:

- Automatic register allocation. Registers must be aliased manually.
- Complex native floating-point math extensions.
- Custom linker script generation.
- Automated struct layouts or data-structure alignment.
- Arbitrary-depth nested loops without explicit labels.
- A stable v1-to-v2 compatibility layer.

The v2.0 compiler is currently being rebuilt and is intentionally breaking backwards compatibility with v1.1.

The goal is not to hide the processor.

The processor is exactly what we are here for.

---

## 19. Version 2.0 Development

Riscrithm v2.0 is currently under active development.

The v2.0 compiler is being rebuilt from the ground up with the goal of improving:

- Compiler performance.
- Developer experience.
- Syntax consistency.
- Compiler architecture.
- Error handling.
- Long-term maintainability.

The v2.0 syntax has been finalized, and implementation work is now underway.

### Backwards Compatibility

Compatibility with v1.1 is **not maintained**.

Existing v1.1 projects should remain pinned to v1.1 until they are migrated to the new compiler.

The breaking changes are intentional and are a consequence of removing accumulated technical debt from the previous compiler architecture.

In other words, the compiler has reached the traditional software-engineering milestone where the only reasonable solution is to rebuild the thing and pretend the previous version was a learning experience.

---

## 20. Final Example

A compact Riscrithm program can therefore look like this:

```text
header default
entrypoint main

define foo = x1
define bar = x2

main:
    load foo = 10
    load bar = 20

    foo += bar

    halt
```

With optimization enabled:

```text
riscrithm "main.txt" "main.s" --optimize
```

Riscrithm produces clean RISC-V assembly while retaining explicit control over registers, memory, branching, and hardware instructions.

A small language for doing the things bare-metal programming has spent decades making unnecessarily verbose.
