# Riscrithm

**Readable macro-assembly. Explicit hardware control. Fewer hand-written XOR rituals.**

Riscrithm is a high-level macro-assembly dialect that compiles directly to pure RISC-V assembly. It connects readable expressions with the deterministic control of bare-metal programming, while keeping registers, memory, branches, and hardware instructions visible.

You get conveniences for the repetitive parts without a hidden runtime between your source and the processor. The processor is very much still invited.

[What it does](#what-it-does) · [Get started](#get-started) · [A quick example](#a-quick-example) · [How it fits together](#how-it-fits-together) · [Reference](#reference) · [Examples](#examples) · [Good to know](#good-to-know) · [Design notes](#design-notes) · [Version 2.0 preview](#version-20-preview)

> [!CAUTION]
> **Riscrithm v2.0 is under active development and intentionally breaks v1.1 compatibility.** Most v1.1 scripts require changes for v2.0. Keep active projects pinned to **v1.1** until you migrate them.
>
> The v2.0 syntax is finalized, and the compiler rebuild has started. This README keeps the **v1.1 guide** separate from the complete [v2.0 syntax preview](#version-20-preview).

## What it does

- Combines source files into reusable packages and utility libraries through global and selective imports.
- Makes control flow readable with inline conditions, label-based loops, jumps, and subroutine returns.
- Validates source structure, labels, imports, and control-flow targets before assembly generation.
- Optimizes redundant assignments, identity arithmetic, and applicable power-of-two operations.
- Gives system and interrupt instructions readable names.
- Handles stack-pointer adjustments for stack operations and provides explicit heap-memory syntax.
- Supports assignments, compound math, increments, decrements, register clearing, swaps, and bitwise expressions.
- Lets you bypass preprocessing with raw assembly blocks or individual raw instructions.
- Produces consistently formatted `.s` output for the next stage of a RISC-V toolchain.

## Get started

### Compile from the command line

The v1.1 compiler interface is:

```text
riscrithm "source_code_file" "assembly_target_file" [-o/--optimize]
```

The arguments are:

| Argument | Description |
|---|---|
| `source_code_file` | The Riscrithm input file. |
| `assembly_target_file` | The generated `.s` assembly file. If the file does not exist, the compiler creates it automatically. |
| `-o`, `--optimize` | Enables the comprehensive optimization pass. |

For a normal compilation:

```text
riscrithm "main.txt" "main.s"
```

To enable optimization:

```text
riscrithm "main.txt" "main.s" --optimize
```

The output is standard RISC-V assembly, which you pass to an appropriate assembler, linker, simulator, hardware toolchain, or debugger. The compiler creates the target file when it does not already exist.

> [!NOTE]
> The command examples assume the `riscrithm` compiler is available. This guide documents its invocation and language behavior; the supplied project documentation does not specify a package-installation procedure.

### Choose the syntax version

The guide, reference, and examples below describe **v1.1**. The final section contains the established **v2.0 syntax**, including every form from the development showcase. Treat that section as a preview for the compiler rebuild, not as interchangeable v1.1 source.

## A quick example

Define two register aliases, load values, add them, and emit the environment-call instruction:

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

Compile it with optimization enabled:

```text
riscrithm "main.txt" "main.s" --optimize
```

`foo` and `bar` are explicit aliases; `foo += bar` becomes a hardware-level arithmetic operation. `halt` maps to `ecall`, whose effect depends on the execution environment.

## How it fits together

### Declare the main file's section and entrypoint

Every main source file declares its target section and entrypoint at the top. The `header` and `entrypoint` directives, macro definitions, and import statements belong outside label blocks. Labels themselves are also unindented; executable instructions belong inside their indented blocks.

The section-selection directive is:

```text
header default
```

For example, this source:

```text
header default
```

generates:

```asm
.section .text
```

The entrypoint directive identifies where program execution begins:

```text
entrypoint main
```

It generates the global-symbol declaration:

```asm
.globl main
```

A minimal main-file preamble therefore contains:

```text
header default
entrypoint main
```

The entrypoint symbol must also resolve to a defined label in the compilation unit.

### Import reusable source files

Use a global import to include an entire source file:

```text
import "packages/display_utilities.txt"
```

The imported source becomes part of the unified compilation unit.

Use selective `from ... import` statements when you only need particular label symbols:

```text
from "libraries/math_helpers.txt" import qux, quux
from "libraries/math_helpers.txt" import corge
```

That lets you reuse the requested labels without importing every symbol from the file.

> [!IMPORTANT]
> **Only the main source file declares `header` and `entrypoint`.** Imported modules and secondary source files must omit both directives; they serve as modular components.

### Define text-replacement macros

`define` creates aliases, constants, or short inline fragments:

```text
define foo = x1
define bar = x2
define baz = x3
define qux = x4
define quux = 10
define corge = 20
define clearFoo = foo ^^
```

When the parser encounters `foo`, it substitutes `x1` before processing logical expressions. For example:

```text
define foo = x1

main:
    load foo = 10
```

The register alias is handled as though you had written the register directly. Macros can name a value such as `10`, a register such as `x1`, or a short shorthand such as `foo ^^`.

### Add comments

In normal Riscrithm source, `#` starts a comment and the compiler strips the rest of that line:

```text
load foo = 10 # Initialize foo
```

Inline comments therefore explain the source without adding generated instructions. Raw assembly blocks have their own bypass behavior, described below.

### Let validation check the source structure

The compiler validates the unified source before generating assembly. It checks these structural problems:

| Check | What is required |
| --- | --- |
| Missing header | The main source must declare a target assembly section with `header`. |
| Invalid entrypoint | The entrypoint symbol must resolve to a valid, defined label. |
| Global duplicate labels | A label may not be declared more than once across unified modules. |
| Undefined jumps and branches | Every targeted label must exist. |
| Unreachable code | Basic control-flow analysis detects dead code after terminating instructions such as `return`. |
| Duplicate file imports | Repeated imports of the exact same file path are detected and reported. |
| Duplicate label imports | Importing the same specific label more than once causes a syntax-validation failure. |

For example:

```text
example:
    return
    foo ++
```

The increment after `return` is unreachable. These checks happen before assembly output is generated; invalid structure stops compilation.

### Use labels and indentation for execution blocks

Standard labels end with a colon and start at the left margin. Instructions inside them must be indented with spaces or tabs:

```text
main:
    load foo = quux
    move bar = foo
```

An executable instruction left unindented outside a valid block produces `SyntaxError`.

### Bypass preprocessing with raw assembly

Prefix a label with `!!` to make the entire block raw:

```text
!!raw_block:
    li x1, 10
    variable ^^ # This stays exactly as written!
```

The compiler removes `!!` from the label and emits the block's contents without processing them. Macros and Riscrithm shorthands are not expanded there. In the example, `variable ^^` deliberately stays as written; passing text through does not make it a valid RISC-V instruction.

For a single raw instruction, put `!!` directly before an indented instruction:

```text
process_data:
    load foo = 5
    !!addi x1, x1, 10
    foo ++
```

Only the prefixed instruction bypasses processing. This is useful for specialized hardware instructions that have no Riscrithm shorthand.

### Build loops and subroutines explicitly

Riscrithm keeps loops at the label-and-branch level rather than introducing high-level `while` or `for` constructs.

An infinite loop:

```text
infinite_loop:
    foo ++
    @infinite_loop
```

A conditional loop:

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

The generated control flow stays visible: initialize registers, compare them, branch to the body or end, and jump back when appropriate.

A label can also serve as a reusable subroutine:

```text
multiply_logic:
    foo *= bar
    return
```

`return` compiles to the native RISC-V `ret` instruction.

### Follow the two-pass pipeline

| Pass | Work performed |
| --- | --- |
| 1. Sanitization and validation | Evaluate module imports, resolve source layout, validate structure, check labels and control-flow targets, and prepare the unified compilation unit. |
| 2. Parsing and optimization | Replace macros, process shorthands, apply enabled optimizations, and generate RISC-V assembly. |

The `-o` and `--optimize` flags enable the transformations described in the optimizer reference below.

## Reference

### System and interrupt controls

| Riscrithm | RISC-V Assembly | Description |
|---|---|---|
| `interrupt.u` | `uret` | User-mode trap return |
| `interrupt.s` | `sret` | Supervisor-mode trap return |
| `interrupt.m` | `mret` | Machine-mode trap return |
| `wait` | `wfi` | Wait for interrupt |
| `trap` | `ebreak` | Debugger trap |
| `halt` | `ecall` | System environment call / halt |
| `...` | `nop` | No-operation |

For example:

```text
handle_system_events:
    wait
    interrupt.u
```

These keywords are readable names for the corresponding low-level instructions. In particular, `halt` emits `ecall`; the surrounding environment determines how that environment call is handled.

### Jumps and conditional branches

Use `@label` for an unconditional jump:

```text
execute_jump:
    @some_label
```

It compiles to:

```asm
j some_label
```

Conditions use an inline, ternary-like form:

```text
compare_registers:
    if foo == bar @true_block else @false_block
    if foo > baz @greater_block else @lesser_block
```

The compiler maps supported comparisons to RISC-V branches such as `beq`, `bne`, `blt`, and `bge`. It also arranges register order for asymmetric comparisons such as `>` and `<=`.

You can omit `else` for a guard-style conditional jump:

```text
guard_check:
    if foo == bar @true_block
```

### Load and move

Immediate assignment and register-to-register copying use separate forms:

```text
assign_values:
    load foo = 100
    move bar = foo
```

Corresponding assembly:

```asm
li foo, 100
mv bar, foo
```

When an integer literal can use an immediate instruction, the compiler selects the corresponding `i`-suffixed form, such as `addi` or `xori`.

### Compound math

```text
apply_math:
    foo += 5
    bar *= baz
    foo <<= 2
```

These expressions map to the corresponding RISC-V arithmetic or shift instructions. The complete operator table below includes the remaining arithmetic and bitwise forms.

### Increment and decrement

```text
adjust_counters:
    foo ++
    bar --
```

An increment:

```text
foo ++
```

becomes:

```asm
addi foo, foo, 1
```

A decrement:

```text
bar --
```

becomes:

```asm
addi bar, bar, -1
```

### Clear a register with `^^`

```text
reset_state:
    foo ^^
```

The shorthand XORs the register with itself:

```asm
xor foo, foo, foo
```

The resulting value is zero.

### Swap two registers

```text
perform_swap:
    foo swap bar
```

The compiler expands `swap` into the triple-XOR exchange:

```asm
xor foo, foo, bar
xor bar, foo, bar
xor foo, foo, bar
```

That exchanges the two register values without a third temporary register. The XOR ritual is still happening; at least you no longer have to type it every time.

### Memory widths

Stack and heap operations use explicit data-width suffixes:

| Suffix | Width | Size |
| --- | --- | --- |
| `.b` | Byte / 8-bit. | 1 byte. |
| `.w` | Word / 32-bit. | 4 bytes. |
| `.d` | Double-word / 64-bit. | 8 bytes. |

### Stack operations

Stack expressions adjust the hardware stack pointer, `sp`, according to the chosen width.

**Push with `->`:** decrement `sp` by the data width, then store the register value.

```text
save_context:
    foo -> stack.w
```

A word push subtracts 4 bytes before storing.

**Pop with `<-`:** load from the current stack address, then increment `sp` by the data width.

```text
restore_context:
    bar <- stack.d
```

A double-word pop adds 8 bytes after loading.

**Peek with `=`:** load from the current stack address without changing `sp`.

```text
check_top:
    baz = stack.b
```

This reads a byte from the top of the stack while leaving the pointer in place.

### Heap operations

Heap syntax requires an explicit base-address register, introduced with `&`.

Store a value with `->`:

```text
write_memory:
    foo -> heap.w from &bar
```

This stores the value in `foo` at the address held by `bar`.

Load a value with `<-`:

```text
read_memory:
    baz <- heap.b from &foo
```

This loads a byte from the address held by `foo` into `baz`.

### Optimizer: redundant assignments

With optimization enabled, consecutive redundant assignments and useless load/move operations targeting the same register are removed.

Source:

```text
load foo = 128
load foo = 128
```

Optimized output:

```asm
li x1, 128
```

The register aliases used in these optimizer examples are `foo = x1`, `bar = x2`, and `baz = x3`.

### Optimizer: identity math

Identity operations are handled according to whether the destination is also the source.

For a self-identity operation:

```text
foo = foo + 0
bar = bar * 1
```

the instructions can be removed entirely:

```text
# Instructions deleted entirely
```

When the source and destination differ:

```text
foo = bar + 0
foo = bar * 1
```

the expression becomes a register copy:

```asm
mv x1, x2
mv x1, x2
```

### Optimizer: strength reduction

Where applicable, multiplication or division by a constant power of two is rewritten as a logical shift.

Source:

```text
foo = bar * 2
baz = foo / 8
```

Optimized output:

```asm
slli x1, x2, 1
srli x3, x1, 3
```

These examples show the applicable shift transformation; they do not introduce a separate runtime arithmetic layer.

### Generated assembly formatting

Generated `.s` files keep labels at the left margin and indent instructions consistently. The output is pretty-printed for readability and can be handed to standard RISC-V development tools without manual reformatting.

That includes hardware simulators, assemblers, binary toolchains, linkers, and desktop debuggers. You still choose the toolchain and execution environment appropriate to your target.

### Naming conventions

Use **camelCase** for register aliases and dynamic variables:

```text
firstNum
addressRegister
stackOffset
```

Use **snake_case** for labels, jump targets, execution blocks, loop boundaries, and subroutines:

```text
loop_start
on_true
error_handler
```

Use **SCREAMING_SNAKE_CASE** for static configuration values, macro constants, and invariant boundaries:

```text
DEFAULT_HEADER
MAX_BUFFER_SIZE
IMM_VALUE
```

The different shapes make registers, control-flow targets, and constants easier to distinguish at a glance.

### Complete core-expression and memory reference

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

### Complete math and bitwise reference

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

## Examples

The examples below pair each v1.1 source with its documented optimized assembly output. Small source fragments elsewhere in the guide assume the surrounding declarations and target labels are present.

### Arithmetic and assignments

Source:

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

Optimized RISC-V assembly:

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

This shows aliases becoming hardware registers, immediate loads becoming `li`, compound additions becoming `addi`, and register addition becoming `add`.

### Memory operations and swapping

> [!CAUTION]
> This example uses the original illustrative aliases `foo = x1` and `bar = x2`. On RISC-V, **`x2` is `sp`**, so loading `bar` also changes the stack pointer. Choose appropriate working registers and initialize a valid stack for your target before executing a stack-based program. The source/output pair below documents the expansion as written.

Source:

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

Optimized RISC-V assembly:

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

The word pushes and pops adjust `sp` by 4 bytes per operation, and the final swap expands to three XOR instructions.

### Identity math, strength reduction, and branching

Source:

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

Optimized RISC-V assembly:

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

The self-identity instructions disappear, the cross-register identity becomes `mv`, and the power-of-two compound operations become shifts. The condition becomes a branch plus an explicit jump to the alternative block.

### Modularity and raw assembly

> [!WARNING]
> This modularity example is a **syntax and expansion sketch**. Its `load foo = qux` and `load bar = quux` use imported label symbols where the documented `load` form expects an immediate. The listed output leaves those `load` lines unresolved, and the main block includes instructions after `halt`. Review those choices against your compiler and execution environment before treating it as an assemblable program.

`libraries/orange_banana.txt`:

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

`libraries/horse_battery.txt`:

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

`main.txt`:

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

Documented assembly expansion:

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

The example retains global and selective imports, raw blocks and raw instructions, shifts, bitwise mutation, swapping, clearing, returns, heap storage, branches, and no-operation syntax in one multi-file walkthrough.

## Good to know

> [!IMPORTANT]
> This is a lightweight abstraction over RISC-V assembly. Register choices and the execution environment remain explicit responsibilities. The examples show syntax and translations; they do not establish a calling convention, initialize a general-purpose runtime, or choose a linker layout for you.

The v1.1 implementation does not provide:

- Automatic register allocation; aliases must be chosen manually.
- Complex native floating-point math extensions.
- Custom linker-script generation.
- Automated struct layouts or data-structure alignment.
- Arbitrary-depth nested loops without explicit labels.
- A stable v1-to-v2 compatibility layer.

The v2.0 compiler is being rebuilt with intentional breaking changes. Keep existing v1.1 projects pinned until you migrate them rather than mixing the two syntax versions in one source file.

## Design notes

Riscrithm connects readable source expressions such as:

```text
if foo == bar
foo += 5
```

with hardware instructions such as:

```asm
beq
addi
```

Its architecture separates file modularity and unification, structural validation, preprocessor definitions, and instruction parsing with optimization. Those stages translate the source into assembly without introducing a hidden runtime between your code and the processor.

The scope stays focused on making bare-metal assembly less tedious to write by hand. Full LLVM infrastructure, C compilers, and large language toolchains have different responsibilities. Here, the register offsets and branch destinations remain visible; you just get to stop repeating quite so much machinery.

## Version 2.0 preview

> [!CAUTION]
> **v2.0 intentionally does not maintain v1.1 compatibility.** The syntax is finalized and implementation work has started, but the compiler is under active development. Existing projects should stay on v1.1 until they are migrated.

The core compiler is being rebuilt from the ground up to improve compiler performance, developer experience, syntax consistency, compiler architecture, error handling, and long-term maintainability. The breaking changes are intentional: they remove accumulated technical debt from the previous architecture.

> [!IMPORTANT]
> The complete established syntax showcase is preserved below, grouped by topic. It includes imports, declarations, type forms, operators, functions, control flow, and register operations. Repeated names and dotted bodies are illustrative alternatives and placeholders, not a single complete program. The preview shows intended syntax; it is not a claim that every form is already implemented.

### Imports and import exclusions

Whole-library forms, aliases, and exclusions:

```text
import "library" as lib
import "library" as ...
import "library"

import "library" as lib without {Cs, Vs, As}
import "library" as lib without {Cs, As, Ls}
import "library" as lib without {As, Vs, Ls}
import "library" as ... without {Cs}
import "library" as ... without {Vs}
import "library" as ... without {As}
import "library" as ... without {Ls}

import "library" without {Cs, Vs, As}
import "library" without {Cs, As, Ls}
import "library" without {As, Vs, Ls}
import "library" without {Cs}
import "library" without {Vs}
import "library" without {As}
import "library" without {Ls}
```

Selective imports, namespaced forms, individual symbols, function imports, and dependency-exclusion forms:

```text
from "library" import {Cs} without {Vs, As, Ls}
from "library" import {Vs} without {As, Cs, Ls}
from "library" import {As} without {Ls}
from "library" import {Ls} without {Cs}

from "library" as lib import {Cs} without {Vs, As, Ls}
from "library" as lib import {Vs} without {As, Cs, Ls}
from "library" as ... import {As} without {Ls}
from "library" as ... import {Ls} without {Cs}

from "library" as lib import {Cs}
from "library" as lib import {Vs}
from "library" as ... import {As}
from "library" as ... import {Ls}

from "library" import {C.apples} without {Cs, Vs, As}
from "library" import {V.banana} without {Ls, As, Vs}
from "library" import {A.orange} without {Cs}
from "library" import {L.cherry} without {Vs}

from "library" import {C.apples, function}
from "library" import {V.banana, function}
from "library" import {A.orange}
from "library" import {L.cherry}

from "library" import {function, function}
from "library" import {function}

from "library" import {function} without! {dependency}
from "library" import {function} without {dependency}
```

### Comment markers and `using` forms

The showcase includes both comment-marker forms, its backslash marker, and the `Align`, `Balig`, and `Linux` directives:

```text
# ................................
; ................................

\\\

using Align of 0
using Balig of 0
using Linux
```

### Variable, array, and string declarations

The declaration examples cover typed values, explicit arrays, ranges, repeated values, strings, and repetition syntax:

```text
V.apples: [B] = [0, 1, 2, 3]
V.banana: [H] = [2;4 .. 100]
V.orange: [W] = [1 .. 5]
V.cherry: [D] = [0 ** 3]
V.apples: [F] = [4.2, 1.337]

V.apples: S = "................"
V.banana: R = '................'

V.apples: S = "..." ** 42
V.banana: R = '...' ** 42
```

### Constant declarations and bound forms

Scalar, string, array, and type-bound forms:

```text
C.apples: B = {}
C.banana: H = {}
C.orange: W = {}
C.cherry: D = {}
C.apples: F = {}

C.banana: S = {}
C.orange: R = {}

C.apples: [B] = {}
C.banana: [H] = {}
C.orange: [W] = {}
C.cherry: [D] = {}
C.apples: [F] = {}

C.apples: B = B^+
C.banana: H = H^+
C.orange: W = W^+
C.cherry: D = D^+
C.apples: F = F^+

C.apples: B = B^-
C.banana: H = H^-
C.orange: W = W^-
C.cherry: D = D^-
C.apples: F = F^-
```

### Aliases and namespaced declarations

Register-alias and library-qualified declaration forms:

```text
A.apples: `x = x
A.banana: `t = t
A.orange: `f = f

lib::C.apples: [B] = 42
lib::C.banana: [H] = 42
lib::C.orange: [W] = 42
lib::C.cherry: [D] = 42
lib::C.apples: [F] = 42.0
```

### Function declarations and dependencies

Namespaced functions, unqualified functions, and explicit dependency forms:

```text
lib::function (dependency)
{
....................
....................
@dependency
}

lib::function
{
....................
....................
@dependency
}

function (dependency)
{
....................
....................
@dependency
}
```

### Function targets, control flow, and raw forms

The showcase places these register-transfer, typed-condition, comparison, jump, and raw-code forms together:

```text
=> lib::function

=> function
{
|B| A.reg -> |||
|H| A.reg <- |||
|W| A.reg <? |||
128 A.reg <- |||

if V.var == {[B]}: @good
if V.var != {[H]}: @good
if V.var >= {[W]}: @good
if V.var <= {[D]}: @good
if V.var == {[F]}: @good

if V.var == {B}: @good
if V.var != {H}: @good
if V.var >= {W}: @good
if V.var <= {D}: @good
if V.var == {F}: @good

if V.var == {S}: @good
if V.var != {R}: @good

if V.var > V.var: @good
@bad

if V.var < V.var: @good
@bad

!! ....................
}

!! function
{
....................
....................
....................
}
```

### Arithmetic, bitwise operators, and casts

These alternatives retain the integer-style operators, dotted arithmetic forms, unary operations, shifts, and explicit casts exactly as shown in the syntax showcase:

```text
## function
{
A.reg = A.reg + A.reg
A.reg = A.reg - A.reg
A.reg = A.reg * A.reg
A.reg = A.reg / A.reg
A.reg = A.reg % A.reg

A.reg = A.reg & A.reg
A.reg = A.reg | A.reg
A.reg = A.reg ^ A.reg

A.reg = ~A.reg

A.reg = A.reg >> A.reg
A.reg = A.reg << A.reg
}

## function
{
A.reg = A.reg +. A.reg
A.reg = A.reg -. A.reg
A.reg = A.reg *. A.reg
A.reg = A.reg /. A.reg
A.reg = A.reg %. A.reg
}

## function
{
A.reg = A.reg + (B)A.reg
A.reg = A.reg - (H)A.reg
A.reg = A.reg * (W)A.reg
A.reg = A.reg / (D)A.reg
A.reg = A.reg % (B)A.reg

A.reg = (B)A.reg & A.reg
A.reg = (H)A.reg | A.reg
A.reg = (W)A.reg ^ A.reg

A.reg = ~(B)A.reg

A.reg = (B)A.reg >> (B)A.reg
A.reg = (H)A.reg << (H)A.reg
}

## function
{
A.reg = A.reg +. (F)A.reg
A.reg = A.reg -. (F)A.reg
A.reg = A.reg *. (F)A.reg
A.reg = A.reg /. (F)A.reg
A.reg = A.reg %. (F)A.reg

A.reg = (F)A.reg +. A.reg
A.reg = (F)A.reg -. A.reg
A.reg = (F)A.reg *. A.reg

A.reg = (F)A.reg +. (F)A.reg
A.reg = (F)A.reg /. (F)A.reg
A.reg = (F)A.reg %. (F)A.reg
}
```

### Self operators and compound mutations

Self-mutating forms, compound arithmetic and bitwise assignments, exchange syntax, and dotted compound assignments:

```text
function
{
A.reg^^
A.reg++
A.reg--
A.reg~~
}

function
{
A.reg += A.reg
A.reg -= A.reg
A.reg *= A.reg
A.reg /= A.reg
A.reg %= A.reg

A.reg &= A.reg
A.reg |= A.reg
A.reg ^= A.reg

A.reg <> A.reg
}

function
{
A.reg +.= A.reg
A.reg -.= A.reg
A.reg *.= A.reg
A.reg /.= A.reg
A.reg %.= A.reg
}
```

### Memory and value-transfer forms

The preview includes width-qualified transfers, explicit-width alternatives, pointer targets, conditional targets, and value-oriented transfer forms:

```text
function
{
|B| A.reg ~> C.pointer ? @function
|H| A.reg ~> C.pointer ? @function
|W| A.reg ~> C.pointer ? @.
128 A.reg ~> C.pointer ? @.
}

function
{
() V.var -> V.var
$$ V.var
<- V.var
}

function
{
|B| A.reg <~ C.pointer
|H| A.reg <~ C.pointer
128 A.reg <~ C.pointer
}
```

### Execution-block alternatives

The following alternatives retain each full form from the showcase, including the repeated import and function context. Their headings identify the visible syntax rather than assigning undocumented runtime behavior to the markers.

#### Plain `()` block

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

()
{
....................
....................
....................
}
```

#### Numbered `() 42` block

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

() 42
{
....................
....................
....................
}
```

#### `<-` with multiple numbered `()` blocks

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

<-
{
....................
....................
....................
}

() 37
{
....................
....................
....................
}

() 42
{
....................
....................
....................
}
```

#### Plain `<-` block

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

<-
{
....................
....................
....................
}
```

#### Numbered `<- 0` block

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

<- 0
{
....................
....................
....................
}
```

#### Multiple numbered `<-` blocks

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
<- lib::V.var
}

<- 0
{
....................
....................
....................
}

<- 1
{
....................
....................
....................
}
```

#### `><` form

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
><
}

><
{
....................
....................
....................
}
```

#### `>fixme<` form

```text
# ................................................................

import "library" as lib

=> function
{
if lib::V.var < lib::V.var: @good
@bad
if lib::V.var > lib::V.var: @good
>fixme<
}
```

The preview is deliberately kept separate from the v1.1 reference so you can inspect the new language forms without accidentally applying them to the older compiler. A compiler rebuild already provides enough surprises of its own.
