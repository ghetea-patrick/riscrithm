# Riscrithm

**Readable source in. RISC-V assembly out.**

Riscrithm is a small macro-assembly language and compiler written in Go. It gives register math, branches, stack operations, and imports a cleaner syntax while keeping the generated assembly easy to follow. The registers are still yours to manage. At least the paperwork is shorter.

[What it does](#what-it-does) · [Get started](#get-started) · [A quick example](#a-quick-example) · [How it fits together](#how-it-fits-together) · [Reference](#reference) · [Good to know](#good-to-know)

> [!IMPORTANT]
> This README documents **v1.1**. The **v2.0** compiler is being rebuilt with a new syntax and intentionally breaks compatibility. Keep existing projects pinned to v1.1 until you are ready to migrate. A short v2 preview lives at the end.

## What it does

- Compiles readable register expressions into RISC-V assembly text.
- Adds macros, file imports, conditional jumps, and compact register shorthands.
- Expands stack and address-based memory operations.
- Lets you drop into raw assembly for a line or a whole block.
- Offers optional local optimizations and checks common source-layout mistakes.

## Get started

> [!NOTE]
> Building the compiler requires **Go 1.21+**. It uses only the Go standard library. Assembling or running the output requires a separate RISC-V toolchain or simulator.

With the standalone compiler source saved as `main.go`, build the executable:

```bash
go build -o riscrithm main.go
```

Compile a source file into an assembly file:

```bash
./riscrithm main.txt main.s
./riscrithm main.txt main.s --optimize
```

The argument order is **source, output, optional optimization flag**. `-o` is the short form of `--optimize`; it enables optimization rather than naming the output file. Source filenames can use any extension.

## A quick example

Save this as `main.txt`:

```text
header default
entrypoint main

define countReg = t0
define stepReg = t1
define sumReg = t2

main:
    load countReg = 10
    load stepReg = 20
    countReg += 5
    sumReg = countReg + stepReg
    trap
```

Run `./riscrithm main.txt main.s`. The result is:

```asm
.section .text
.globl main
main:
   li t0, 10
   li t1, 20
   addi t0, t0, 5
   add t2, t0, t1
   ebreak
```

That leaves `35` in `t2` before the debugger trap. Riscrithm writes the assembly; your target environment handles the next step.

## How it fits together

### Start with the file layout

The first two nonempty, non-comment directives in the main file are:

```text
header default
entrypoint main
```

`header default` emits `.section .text`. A custom header, such as `header .section .text`, emits the text after `header`. `entrypoint main` exports `main` through `.globl main`; define that label and keep your linker or simulator entry configuration aligned with it.

Keep labels flush left and end them with `:`. Indent instructions with spaces or tabs. Put definitions and imports at the top level. A `#` begins a comment, and everything after it on that line is removed.

### Give registers readable names

```text
define countReg = t0
define LIMIT = 10

main:
    load countReg = LIMIT
    countReg --
```

Definitions are text replacements, applied before instruction translation. Use distinct macro names that do not overlap with other names, and keep expansions independent of other macros.

Use `camelCase` for aliases, `snake_case` for labels, and `UPPERCASE` for constants if you want a consistent visual rhythm. The processor will survive your naming choices, but your future self may have preferences.

### Build loops from labels

```text
header default
entrypoint main

main:
    load t0 = 0
    load t1 = 10

count_loop:
    if t0 == t1 @done
    t0 ++
    @count_loop

done:
    trap
```

Conditions compare registers. An `if` without `else` falls through when false. With `else`, it branches to one of the two named labels:

```text
    if t0 < t1 @less else @not_less
```

### Split out reusable labels

Import a whole file or choose labels from one:

```text
import "helpers.txt"
from "other_helpers.txt" import tick, reset_counter
```

Only the main file gets `header` and `entrypoint`. Imported files contain reusable labels and their instructions. Import paths are resolved from the compiler's working directory. Include any labels your selected blocks depend on, and avoid circular imports.

For example, `helpers.txt` can contain:

```text
tick:
    t0 ++
    return
```

A main file can call it with a raw RISC-V call instruction:

```text
header default
entrypoint main

from "helpers.txt" import tick

main:
    load t0 = 0
    !!jal ra, tick
    trap
```

`@tick` is a jump; `jal ra, tick` also sets a return address. Manage return addresses and register preservation when your calls need them.

### Use raw assembly where it helps

Prefix one instruction with `!!`, or prefix a label to make its whole block raw:

```text
main:
    !!mv t1, t0
    @native_block

!!native_block:
    addi t1, t1, 1
    ebreak
```

Raw instructions bypass macro and shorthand expansion. Source comment removal and whitespace cleaning still apply.

## Reference

### Register operations

| Source | Assembly or effect |
| --- | --- |
| `load t0 = 42` | `li t0, 42` |
| `!!mv t1, t0` | A direct register copy using native assembly. |
| `t0 ++`, `t0 --` | Add or subtract one. |
| `t0 ^^` | Clear the register with `xor t0, t0, t0`. |
| `t0 swap t1` | Exchange two distinct registers with three XOR instructions. |
| `t0 = t1 + t2` | Register arithmetic. |
| `t0 = t1 + 5` | Use an immediate instruction where available. |
| `t0 += 5` | Expand to `t0 = t0 + 5`. |

Use `load` for nonnegative decimal literals. For other assembler literal forms, write the native instruction, such as `!!li t0, -1`.

| Operation | Operators | Notes |
| --- | --- | --- |
| Addition and subtraction | `+`, `-` | Register operands or supported immediate forms. |
| Bitwise operations | `&`, `\|`, `^` | Register operands or supported immediate forms. |
| Logical shifts | `<<`, `>>` | Register or immediate shift counts. |
| Multiplication, division, remainder | `*`, `/`, `%` | Use register operands; the emitted instructions need the matching target extensions. |
| Compound assignment | `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `\|=`, `^=`, `<<=`, `>>=` | Apply the corresponding operation to the destination register. |

Keep immediate values within the target instruction's range. When using optimization, positive power-of-two literals in multiplication or division can be rewritten as shifts.

### Branches

| Source | Emitted instruction |
| --- | --- |
| `@target` | `j target` |
| `if t0 == t1 @target` | `beq t0, t1, target` |
| `if t0 != t1 @target` | `bne t0, t1, target` |
| `if t0 < t1 @target` | `blt t0, t1, target` |
| `if t0 >= t1 @target` | `bge t0, t1, target` |
| `if t0 > t1 @target` | `blt t1, t0, target` |
| `if t0 <= t1 @target` | `bge t1, t0, target` |

Add `else @other_target` to emit a jump for the false path. Ordering comparisons use the signed RISC-V branch forms shown above.

### Stack and memory

| Source | Effect |
| --- | --- |
| `t0 -> stack.w` | Decrement `sp` by four, then store a word. |
| `t0 <- stack.w` | Load a word, then increment `sp` by four. |
| `t0 = stack.w` | Read the top word without moving `sp`. |
| `t0 -> heap.w from &t1` | Store a word at the address in `t1`. |
| `t0 <- heap.w from &t1` | Load a word from the address in `t1`. |

Width suffixes are `.b` for one byte, `.w` for four bytes, and `.d` for eight bytes. Loads use `lb`, `lw`, or `ld`; stores use `sb`, `sw`, or `sd`. Byte and word loads use the signed load forms.

Initialize a valid stack pointer and valid memory addresses in your target environment. `heap` is an address-based load/store shorthand; it does not allocate memory. Stack operations adjust `sp` by exactly the selected width, so maintain the alignment required by your target and calling convention.

### System instructions

| Source | Assembly |
| --- | --- |
| `return` | `ret` |
| `trap` | `ebreak` |
| `wait` | `wfi` |
| `halt` | `ecall` |
| `interrupt.m`, `interrupt.s`, `interrupt.u` | `mret`, `sret`, `uret` |
| `...` | `nop` |

`halt` emits an environment call. Whether that exits, traps, or invokes a service depends on the runtime and register setup. Interrupt returns and `wait` also depend on the target's privilege mode and supported instructions.

### Optional optimization

Append `-o` or `--optimize` to enable local rewrites:

| Source pattern | Rewrite |
| --- | --- |
| Consecutive identical loads | Keep one load. |
| `t0 = t0 + 0`, `t0 = t0 * 1` | Remove the identity operation. |
| `t0 = t1 + 0`, `t0 = t1 * 1` | Emit a register copy. |
| `t0 = t1 * 8` | Emit `slli t0, t1, 3`. |
| `t0 = t1 / 8` | Emit `srli t0, t1, 3`. |

Subtraction by zero and division by one receive the same identity treatment. These are local simplifications, rather than whole-program optimization.

> [!CAUTION]
> Power-of-two division is rewritten as a **logical right shift**. Choose optimization with your value ranges and signedness in mind; shifting a negative value does not behave like signed division.

## Good to know

### The output still belongs to your toolchain

- The compiler checks source layout, the declared entrypoint, duplicate labels, duplicate imports, direct jump targets, and instructions immediately following a `return`.
- Register allocation, calling conventions, target instruction support, memory setup, assembly, and linking remain under your control.
- Keep generated assembly in your review loop, particularly when using raw instructions or optimization.
- The output path is written as an assembly text file. Choose it deliberately, since an existing file at that path is replaced.

### A peek at v2

The v2 rewrite introduces a different syntax for typed declarations, namespaces, imports, register aliases, functions, and control flow. The goals include more consistent syntax, improved error handling, and a cleaner compiler architecture.

<details>
<summary>Show a small v2 syntax preview</summary>

This is a syntax preview, not a program for the v1.1 compiler:

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

# ................................
; ................................

\\\

using Align of 0
using Balig of 0
using Linux

V.apples: [B] = [0, 1, 2, 3]
V.banana: [H] = [2;4 .. 100]
V.orange: [W] = [1 .. 5]
V.cherry: [D] = [0 ** 3]
V.apples: [F] = [4.2, 1.337]

V.apples: S   = "................"
V.banana: R   = '................'

V.apples: S   = "..." ** 42
V.banana: R   = '...' ** 42

C.apples: B   = {}
C.banana: H   = {}
C.orange: W   = {}
C.cherry: D   = {}
C.apples: F   = {}

C.banana: S   = {}
C.orange: R   = {}

C.apples: [B] = {}
C.banana: [H] = {}
C.orange: [W] = {}
C.cherry: [D] = {}
C.apples: [F] = {}

C.apples: B   = B^+
C.banana: H   = H^+
C.orange: W   = W^+
C.cherry: D   = D^+
C.apples: F   = F^+

C.apples: B   = B^-
C.banana: H   = H^-
C.orange: W   = W^-
C.cherry: D   = D^-
C.apples: F   = F^-

A.apples: `x  = x
A.banana: `t  = t
A.orange: `f  = f

lib::C.apples: [B] = 42
lib::C.banana: [H] = 42
lib::C.orange: [W] = 42
lib::C.cherry: [D] = 42
lib::C.apples: [F] = 42.0

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

# ................................................................

import "library" as lib

=> function
{
    if lib::V.var < lib::V.var: @good
    @bad
    if lib::V.var > lib::V.var: @good
    >fixme<
}

>< fixme
{
    ....................
    ....................
    ....................
}

# ................................................................

import "library" as lib

=> function
{
    if lib::V.var < lib::V.var: @good
    @bad
    if lib::V.var > lib::V.var: @good
    >fixme<
}

><
{
    ....................
    ....................
    ....................
}

>< fixme
{
    ....................
    ....................
    ....................
}

# ................................................................

import "library" as lib

=> function
{
    if lib::V.var < lib::V.var: @good
    @bad
    if lib::V.var > lib::V.var: @good
    []
}

[]
{
    ....................
    ....................
    ....................
}

# ................................................................

import "library" as lib

=> function
{
    if lib::V.var < lib::V.var: @good
    @bad
    if lib::V.var > lib::V.var: @good
    [fixme]
}

[] fixme
{
    ....................
    ....................
    ....................
}

# ................................................................

import "library" as lib

=> function
{
    if lib::V.var < lib::V.var: @good
    @bad
    if lib::V.var > lib::V.var: @good
    [fixme]
}

[]
{
    ....................
    ....................
    ....................
}

[] fixme
{
    ....................
    ....................
    ....................
}

# ................................................................

from "library" as lib import {C.apples, V.banana, function}

=> function
{
    if lib::C.apples == lib::V.banana: @good
    @bad

    @lib::function
}

# ................................................................

from "library" as ... import {C.apples, V.banana, function}

=> function
{
    if C.apples == V.banana: @good
    @bad

    @function
}

# ................................................................

from "library" import {C.apples, V.banana, function}

=> function
{
    if library::C.apples == library::V.banana: @good
    @bad

    @library::function
}

# ................................................................

=> function
{
    if V.apples == V.banana: <- 0
    <- 1

    if V.apples != V.banana: <-
    @function
}

# ................................................................

=> function
{
    if V.apples == V.banana: >fixme<
    >bug<

    if V.apples != V.banana: ><
    @function
}

# ................................................................

=> function
{
    if V.apples == V.banana: [fixme]
    [bug]

    if V.apples != V.banana: []
    @function
}

# ................................................................
```

The broader design includes selective imports, array and string declarations, typed register operations, and new memory and control-flow forms. Migration will require updating v1.1 source to the new language.

</details>

