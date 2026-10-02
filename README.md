# Riscrithm

### A high-level macro-assembly dialect with slightly fewer bare-metal inconveniences.

Welcome to **Riscrithm**, a lightweight macro-assembly dialect for writing expressive code that compiles directly to pure RISC-V assembly.

Riscrithm is built as a bridge between high-level readability and raw deterministic control, providing a simpler and more expressive interface for bare-metal programming.

Riscrithm provides several conveniences:

- **File modularity** through global and selective label imports.
- **Strict compile-time validation** for missing headers, duplicate labels, and undefined jumps.
- **Readable control flow** with inline ternary-like conditionals and guard clauses.
- **Expressive compound math operators** and built-in value swapping.
- **Clean memory operations** for both stack and heap management.
- **Automatic register swapping** for asymmetric relational operations.
- **Two-pass compilation** with an optional optimizer for dead code and identity math elimination.
- **Seamless raw assembly injection** for specialized hardware instructions.

Riscrithm intentionally keeps its bare-metal transparency while making assembly generation easier to express.

It is designed for programmers who want to manipulate registers and memory without repeatedly writing raw jump instructions and calculating manual bit-shifts.

Because apparently calculating exact stack pointer offsets by hand inside a conditional loop was considered a healthy way to spend an afternoon.

---

## 1. Usage

Riscrithm is invoked via its command-line interface.

Compile a source code file into a target assembly file.

```bash
riscrithm "source_code_file" "assembly_target_file" [-o/--optimize]
```

The compiler translates the source file directly into a `.s` assembly file.

If the target file does not exist, it is created automatically.

Use the `-o` or `--optimize` flag to enable the comprehensive optimization sweep.

---

## 2. Project Structure

Every main Riscrithm file must declare its target section and entrypoint at the top.

```python
header default
entrypoint main
```

This translates directly to `.section .text` and `.globl main`.

Riscrithm also supports modular source files.

You can import an entire file's contents:

```python
import "packages/display_utilities.txt"
```

Or selectively import specific labels to avoid namespace pollution:

```python
from "libraries/math_helpers.txt" import qux, quux
```

Imported sub-files function strictly as modular components and should not include their own header or entrypoint directives.

---

## 3. Macros and Comments

Text-replacement macros are declared using the `define` keyword.

```python
define foo = x1
define bar = x2
define MAX_COUNT = 10
```

The parser swaps these values before evaluating actual logical expressions.

This is useful for aliasing registers and creating constants.

Comments are written using the `#` symbol.

The compiler strips out anything following a `#` on any line, allowing safe inline documentation.

---

## 4. Labels and Scoping

Riscrithm enforces strict layout scoping via indentation.

Labels define execution blocks, must end with a colon, and must not have any indentation.

```python
main:
    load foo = MAX_COUNT
    move bar = foo
```

Instructions inside a label block must be indented with spaces or tabs.

Unindented instructions trigger a compilation failure.

### Raw Assembly Blocks

Raw assembly can be written by prefixing a block with `!!`.

```python
!!raw_block:
    li x1, 10
    variable ^^
```

You can also inject single raw instructions inline within a standard block.

```python
process_data:
    load foo = 5
    !!addi x1, x1, 10
    foo ++
```

---

## 5. Control Flow

Unconditional jumps use the `@` prefix.

```python
@some_label
```

Conditional branching uses an inline layout that maps dynamically to hardware branch instructions.

```python
if foo == bar @true_block else @false_block
```

Guard clauses can skip the `else` branch entirely.

```python
if foo > baz @greater_block
```

Subroutines use the `return` keyword, which compiles straight to a hardware `ret` instruction.

```python
multiply_logic:
    foo *= bar
    return
```

System controls are handled through explicit keywords.

| Keyword | RISC-V Instruction |
|---|---|
| `wait` | `wfi` |
| `trap` | `ebreak` |
| `halt` | `ecall` |

---

## 6. Math and Operations

Riscrithm supports immediate assignments and compound mathematical expressions.

```python
load foo = 100
foo += 5
bar *= baz
foo <<= 2
```

The compiler automatically appends the `i` suffix when it detects you are working with an integer literal.

### Increment and Decrement

Fast increment and decrement operators are provided.

```python
foo ++
bar --
```

### Register Clearing

The `^^` operator clears a register efficiently.

```python
foo ^^
```

This translates to `xor foo, foo, foo, zeroing out the register in a single cycle.

### Register Swapping

Registers can be swapped without a temporary third register.

```python
foo swap bar
```

---

## 7. Memory Access

Memory operations require explicit data width indicators: `.b`, `.w`, or `.d`.

### Stack Operations

Stack operations automatically handle the hardware stack pointer.

```python
foo -> stack.w
bar <- stack.d
baz = stack.b
```

The `->` operator pushes, `<-` pops, and `=` peeks without moving the stack pointer.

### Heap Operations

Heap operations require an explicit base address register using pointer notation.

```python
foo -> heap.w from &bar
baz <- heap.b from &foo
```

---

## 8. Optimization

The `-o` flag enables a two-pass optimizer that reduces binary size and execution cost.

### Dead Assignment Elimination

It eliminates dead assignments.

```python
load foo = 128
load foo = 128
```

Becomes a single `li x1, 128`.

### Identity Math Elimination

It removes useless identity math, such as `foo = foo + 0`.

Cross-register identity operations are converted into fast register copies (`mv`).

```python
foo = bar + 0
```

Becomes:

```assembly
mv x1, x2
```

### Strength Reduction

Strength reduction converts expensive multiplications or divisions by constant powers of two into highly efficient bit-shifts.

```python
foo = bar * 2
```

Becomes:

```assembly
slli x1, x2, 1
```

---

## 9. Complete Example

The following example demonstrates macro definitions, compound math, branching, and stack operations.

```python
header default
entrypoint main

define foo = x1
define bar = x2

main:
    load foo = 7
    load bar = 7

    foo *= 8
    bar /= 2

    foo -> stack.w
    bar -> stack.w

    if foo == bar @equal_block else @not_equal_block

equal_block:
    foo ++
    halt

not_equal_block:
    foo --
    trap
```

The script demonstrates the basic Riscrithm workflow:

1. Define the entrypoint and aliases.
2. Perform mathematical operations.
3. Store state in memory.
4. Execute conditional control flow.

---

## 10. Design Philosophy

Riscrithm is designed to provide a readable interface over raw RISC-V instructions.

The compiler deliberately uses descriptive keywords and intuitive operators to make low-level logic concise and visually distinct.

The architecture separates responsibilities into logical steps:

1. **Target acquisition and safety validation:** Pass 1.
2. **Macro expansion and expression evaluation:** Pass 2.
3. **Action execution and code optimization:** Pass 2.

Riscrithm does not attempt to abstract away the hardware or implement a virtual machine.

It simply makes bare-metal execution less tedious to orchestrate.

Because apparently generating a binary needed a calculator, a notepad, and three consecutive XOR instructions just to swap a variable.

---

## 11. Limitations

Riscrithm intentionally remains a lightweight bridge to assembly.

The current implementation does not provide:

- Automatic register allocation.
- High-level loop constructs like `while` or `for`.
- Complex data types or structs.
- A standard library of pre-written software routines.
- Object-oriented abstractions.
- Built-in dynamic memory allocation algorithms.

The v1.1 compiler enforces strict backwards incompatibility with v1.0 scripts.

Old scripts will break. They are meant to break. Technical debt is not a feature.

---

## 12. Final Example

A compact Riscrithm program can therefore look like this:

```python
header default
entrypoint main

define counter = x1

main:
    load counter = 0

loop_start:
    counter ++
    if counter < 10 @loop_start else @end

end:
    halt
```

A small interface for doing the things hardware architectures have spent decades making unnecessarily tedious.
