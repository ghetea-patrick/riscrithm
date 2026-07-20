# !!! CAUTION !!!

I am currently actively rebuilding the core compiler for Riscrithm v2.0 to bring massive performance and developer experience improvements.
Backwards compatability: __Broken__. Most v1.1 scripts will __NOT__ be compatible with v2.0 out of the box.

Hey everyone! I'm excited to announce that the syntax for v2.0 is finally ready and work has officially started for Riscrithm v2.0!
I am actively building a completely redesigned compiler from the ground up to make Riscrithm much better.
__v1.1 scripts will break in v2.0__. Backwards compatability is being intentionally dropped due to technical debt.
### If you have active projects, keep them pinned for v1.1 for now. Stay tuned for progress updates!

# The Riscrithm Developer Manual (v1.1)
Hey there. If you're looking at this, you are probably getting your hands dirty with **Riscrithm**, a high-level macro-assembly dialect that compiles straight down to pure RISC-V assembly. Think of it as a bridge between the readability of a high-level language and the raw, deterministic control of bare-metal hardware.
With the release of **v1.1**, the language has evolved significantly beyond its original release. I have expanded the feature set to introduce file modularity, cleaner control flow, much tighter compile-time error checks, and an enhanced optimization pass. This iteration provides all the core capabilities needed for an expressive developer experience without hiding what the underlying hardware is executing.
Let's dive straight into how the compiler works, the syntax rules, and what's happening under the hood.
## 1. The CLI
To compile source code, use the CLI tool. The syntax is straightforward:
```bash
riscrithm "source_code_file" "assembly_target_file" [-o/--optimize]

```
 * **Source Code:** The Riscrithm input file.
 * **Target File:** The generated .s assembly file. If this file doesn't exist, the compiler creates it on the fly.
 * **Optimization:** Pass -o or --optimize to enable the comprehensive optimization sweep.
## 2. File Structure, Globals & Module Imports
Every Riscrithm file must declare its target section and entrypoint at the very top. These directives, alongside macro definitions and import statements, are the *only* lines allowed to exist completely unindented outside of a label block.
### Header and Entrypoint
 * header <value>: Sets the target assembly section. For instance, header default translates directly to .section .text.
 * entrypoint <symbol>: Defines where the program starts execution. Passing entrypoint main translates to .globl main.
```text
header default
entrypoint main

```
### Imports & Modular Files (New in v1.1)
Riscrithm supports modular source files, making it simple to break projects down into reusable packages and utility libraries.
> **Important Rule:** Imported modules and other secondary sub-files should not include a header or entrypoint directive, as they function strictly as modular components.
> 
 * **Global Import:** Use import to lazily drop an entire file's contents into the current compilation unit.
   ```text
   import "packages/display_utilities.txt"
   
   ```
 * **Selective Import:** Use from ... import to pull only specific label symbols from another source file, avoiding global namespace pollution.
   ```text
   from "libraries/math_helpers.txt" import qux, quux
   from "libraries/math_helpers.txt" import corge
   
   ```
### Definitions (Macros)
Text-replacement macros are declared using the define keyword. This is ideal for aliasing registers, creating constants, or establishing single-line inline code fragments.
```text
define foo = x1
define bar = x2
define baz = x3
define qux = x4
define quux = 10
define corge = 20
define clearFoo = foo ^^

```
Whenever the parser encounters foo, it swaps it with x1 *before* processing any actual logical expressions.
### Comments
Comments are written using the # symbol. The compiler strips out anything following a # on any line, allowing safe inline documentation anywhere.
## 3. Compiler Validation Passes & Error Checking (New in v1.1)
Writing bare-metal assembly can be error-prone and tedious to debug. To catch structural bugs early, v1.1 introduces a strict compile-time validation sweep before generating code. The compiler checks for the following anomalies and halts compilation with descriptive errors if found:
 * **Missing Header:** The main file must include a header directive at the top, or the assembly target cannot be initialized.
 * **Invalid Entrypoint:** The symbol passed to the entrypoint directive must resolve to a valid, defined label within the codebase.
 * **Global Duplicate Labels:** The compiler scans all unified modules and compilation units to guarantee no label is declared more than once.
 * **Undefined Jumps and Branches:** Any jump or branch targeting a label that doesn't exist anywhere in the source or imported tree triggers an instant compilation failure.
 * **Unreachable Code:** The compiler performs basic control-flow analysis to check for dead code, such as placing instructions directly after a return statement within the same block before a new label breaks scope.
 * **Duplicate File Imports:** Importing the exact same file path multiple times across your project is intercepted and flagged.
 * **Duplicate Label Imports:** Attempting to import the same specific label token multiple times via from ... import statements causes a syntax validation failure.
## 4. Labels, Indentation, and Raw Blocks
Riscrithm enforces strict layout scoping via indentation.
### Standard Labels
Labels define execution blocks, must end with a colon, and **must not** have any indentation. Conversely, every instruction inside a label block **must** be indented with spaces or tabs. Leaving an instruction unindented triggers a SyntaxError.
```text
main:
    load foo = quux
    move bar = foo

```
### Raw Assembly Labels (!!)
To completely bypass the preprocessor and write raw RISC-V assembly, prefix your block label with !!. The compiler strips the exclamation marks but passes everything inside that block completely untouched. Macros and shorthands will *not* expand here.
```text
!!raw_block:
    li x1, 10
    variable ^^ # This stays exactly as written!

```
### Inline Raw Assembly (New in v1.1)
If you need a single specialized hardware instruction without changing your entire block style, use the !! prefix inline inside an indented execution block:
```text
process_data:
    load foo = 5
    !!addi x1, x1, 10 # This line bypasses processing and prints raw
    foo ++

```
## 5. Core Features & Instructions
Riscrithm maps expressive statements directly down to hardware-level instructions.
### System & Interrupt Controls
Instead of memorizing low-level privilege opcodes, use explicit system keywords:
| Riscrithm | RISC-V Assembly | Description |
|---|---|---|
| interrupt.u | uret | User-mode trap return |
| interrupt.s | sret | Supervisor-mode trap return |
| interrupt.m | mret | Machine-mode trap return |
| wait | wfi | Wait for interrupt (low-power state) |
| trap | ebreak | Debugger trap |
| halt | ecall | System environment call / halt |
| ... | nop | No-operation (ellipsis) |
```text
handle_system_events:
    wait
    interrupt.u

```
### Branching and Conditionals
To unconditionally jump to a label, use the @ prefix:
```text
execute_jump:
    @some_label # Compiles to: j some_label

```
For conditional branching, Riscrithm uses an inline ternary-like layout. The parser automatically maps your operators to beq, bne, blt, or bge, and swaps registers dynamically to handle asymmetric operations like > and <=.
 * **Standard If/Else Conditionals:**
   ```text
   compare_registers:
       if foo == bar @true_block else @false_block
       if foo > baz @greater_block else @lesser_block
   
   ```
 * **Else-less If Statements (New in v1.1):** Guard clauses and lightweight conditional jumps can skip the else branch entirely:
   ```text
   guard_check:
       if foo == bar @true_block
   
   ```
### Loops (Infinite and Conditional)
Riscrithm avoids high-level loops like while or for to preserve bare-metal transparency. Instead, you build loops using labels, jumps, and inline conditionals.
**An Infinite Loop:**
```text
infinite_loop:
    foo ++
    @infinite_loop

```
**A Conditional Loop:**
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
### Subroutine Return Statements (New in v1.1)
Labels can act as structured, reusable functions using the native return statement, which compiles straight to a hardware ret instruction.
```text
multiply_logic:
    foo *= bar
    return

```
### Operations and Mutators
Riscrithm supports immediate assignments and compound mathematical expressions. The engine automatically appends the i suffix (e.g., addi, xori) when it detects you are working with an integer literal instead of an aliased register.
 * **Load/Move:**
   ```text
   assign_values:
       load foo = 100
       move bar = foo
   
   ```
 * **Compound Math:**
   ```text
   apply_math:
       foo += 5
       bar *= baz
       foo <<= 2
   
   ```
 * **Increments and Decrements:**
   ```text
   adjust_counters:
       foo ++ # Compiles to: addi foo, foo, 1
       bar -- # Compiles to: addi bar, bar, -1
   
   ```
> **The ^^ Shorthand:** To quickly and efficiently clear a register, use the XOR-self operator ^^.
> foo ^^ translates directly to xor foo, foo, foo, zeroing out the register in a single cycle.
> 
```text
reset_state:
    foo ^^

```
### Swapping Variables
To swap the values of two registers without consuming a temporary third register, use the built-in swap command, which expands into a non-destructive triple-XOR sequence:
```text
perform_swap:
    foo swap bar

```
Translates directly into:
```text
xor foo, foo, bar
xor bar, foo, bar
xor foo, foo, bar

```
## 6. Memory Operations (Stack & Heap)
Interacting with system memory requires strict data width indicators: .b (byte/8-bit), .w (word/32-bit), or .d (double-word/64-bit).
### Stack Operations
Stack expressions automatically handle the hardware stack pointer (sp) by shifting its offset before or after data access.
 * **Push (->):** Decrements sp by the relative width, then stores the register data.
   ```text
   save_context:
       foo -> stack.w # Decrements sp by 4, stores word
   
   ```
 * **Pop (<-):** Loads data from the stack pointer, then increments sp by the relative width.
   ```text
   restore_context:
       bar <- stack.d # Loads double-word, increments sp by 8
   
   ```
 * **Peek (=):** Standard memory load from the current stack address without adjusting sp.
   ```text
   check_top:
       baz = stack.b # Loads byte from stack top without moving sp
   
   ```
### Heap Operations
Heap expressions require an explicit base address register indicated via & pointer notation.
 * **Store (->):** Writes data from a register out to the target memory pointer.
   ```text
   write_memory:
       foo -> heap.w from &bar # Stores word from foo into address at bar
   
   ```
 * **Load (<-):** Reads data into a destination register from the target memory pointer.
   ```text
   read_memory:
       baz <- heap.b from &foo # Loads byte into baz from address at foo
   
   ```
## 7. The Compiler Architecture & Optimizer (-o / --optimize)
The compiler uses a fast, lightweight **two-pass system**:
 1. **Pass 1 (Sanitization & Validation):** The parser reads files, strips comments, resolves layout whitespace, evaluates module imports, and executes the compile-time safety and structure checks.
 2. **Pass 2 (Parse & Optimize):** The engine iterates through statements, replaces text macros, handles code shorthands, and applies code optimizations before generating the output text.
When compiling with -o or --optimize, three transformations are applied to make your final binary leaner:
### Dead Assignment Elimination
Consecutive redundant assignments or useless load/move operations targeted at the same register are stripped out.
```text
# Source Input
load foo = 128
load foo = 128

# Optimized Output
li x1, 128

```
### Identity Math Elimination & Transformation (Upgraded in v1.1)
Mathematical expressions that result in no structural change are optimized based on destination contexts:
 * **Self-Identity Elimination:** If a register undergoes identity operations where it is assigned to itself, the instruction is dropped completely.
   ```text
   # Source Input
   foo = foo + 0
   bar = bar * 1
   
   # Optimized Output
   # (Instructions deleted entirely)
   
   ```
 * **Cross-Register Identity Transformation:** If you perform an identity operation where the target destination is a *different* register, the compiler converts the operation into a cheap, fast register copy (mv). This rule spans addition, subtraction, multiplication, and division.
   ```text
   # Source Input
   foo = bar + 0
   foo = bar * 1
   
   # Optimized Output
   mv x1, x2
   mv x1, x2
   
   ```
### Strength Reduction (Bitwise Folding)
Multiplication and division are performance-intensive on hardware. If the optimizer identifies multiplication or division by a constant power of two, it rewrites the command as a highly efficient logical bit-shift.
```text
# Source Input
foo = bar * 2
baz = foo / 8

# Optimized Output
slli x1, x2, 1 # Shift Left Logical by 1
srli x3, x1, 3 # Shift Right Logical by 3

```
## 8. Clean, Ready-to-Use Output
The assembly file output by Riscrithm is cleanly formatted. The produced .s text is automatically pretty-printed: instructions inside execution blocks are neatly aligned, label targets sit perfectly flush against the left margin, and instructions are highly human-readable.
You can take the resulting output and drop it straight into hardware simulators, binary toolchains, linkers, or desktop debuggers without manual formatting adjustments.
## 9. Naming Conventions
To guarantee projects stay scannable, the compiler encourages a clean visual divide across identifier types:
 * **Variables & Registers (camelCase):** Aliases for registers or dynamic variables must start with a lowercase letter, with each subsequent word capitalized.
   * *Examples:* firstNum, addressRegister, stackOffset
 * **Labels & Code Blocks (snake_case):** Jump locations, block scopes, loop entry boundaries, and subroutines use lowercase words divided by underscores.
   * *Examples:* loop_start, on_true, error_handler
 * **Constants & Literals (SCREAMING_SNAKE_CASE):** Static configurations, macro constants, or invariant boundaries use uppercase letters separated by underscores.
   * *Examples:* DEFAULT_HEADER, MAX_BUFFER_SIZE, IMM_VALUE
## 10. Complete Operator & Expression Reference
### Core Expressions and Memory Operators
| Riscrithm Syntax | Category | Internal Expansion / Behavior | Target RISC-V Assembly |
|---|---|---|---|
| load <reg> = <imm> | Assignment | Direct immediate assignment | li reg, imm |
| move <reg1> = <reg2> | Assignment | Register-to-register copy | mv reg1, reg2 |
| <reg1> swap <reg2> | Value Exchange | Triple-XOR non-destructive swap | xor reg1, reg1, reg2
xor reg2, reg1, reg2
xor reg1, reg1, reg2 |
| <reg> -> stack.[b/w/d] | Stack Memory | Dec pointer, store byte/word/double | addi sp, sp, -offset
s[b/w/d] reg, 0(sp) |
| <reg> <- stack.[b/w/d] | Stack Memory | Load byte/word/double, inc pointer | l[b/w/d] reg, 0(sp)
addi sp, sp, offset |
| <reg> = stack.[b/w/d] | Stack Memory | Peek value from top of stack | l[b/w/d] reg, 0(sp) |
| <reg1> <- heap.[b/w/d] from &<reg2> | Heap Memory | Base-register memory read (load) | l[b/w/d] reg1, 0(reg2) |
| <reg1> -> heap.[b/w/d] from &<reg2> | Heap Memory | Base-register memory write (store) | s[b/w/d] reg1, 0(reg2) |
### Math & Bitwise Operators
| Riscrithm Syntax | Operator Type | Evaluated Expression |
|---|---|---|
| <reg> ++ | Self Operator | <reg> = <reg> + 1 |
| <reg> -- | Self Operator | <reg> = <reg> - 1 |
| <reg> ^^ | Self Operator | <reg> = <reg> ^ <reg> *(Fast Register Clear)* |
| <reg> += <val> | Compound Tag | <reg> = <reg> + <val> |
| <reg> -= <val> | Compound Tag | <reg> = <reg> - <val> |
| <reg> *= <val> | Compound Tag | <reg> = <reg> * <val> |
| <reg> /= <val> | Compound Tag | <reg> = <reg> / <val> |
| <reg> %= <val> | Compound Tag | <reg> = <reg> % <val> |
| <reg> <<= <val> | Compound Tag | <reg> = <reg> << <val> |
| <reg> >>= <val> | Compound Tag | <reg> = <reg> >> <val> |
| <reg1> = <reg2> + <val> | Base Arithmetic | Addition (Supports immediate realignment) |
| <reg1> = <reg2> - <val> | Base Arithmetic | Subtraction (Supports immediate realignment) |
| <reg1> = <reg2> & <val> | Base Arithmetic | Bitwise AND (Supports immediate realignment) |
| <reg1> = <reg2> | <val> | Base Arithmetic | Bitwise OR (Supports immediate realignment) |
| <reg1> = <reg2> ^ <val> | Base Arithmetic | Bitwise XOR (Supports immediate realignment) |
| <reg1> = <reg2> << <val> | Base Arithmetic | Logical Shift Left (Supports immediate realignment) |
| <reg1> = <reg2> >> <val> | Base Arithmetic | Logical Shift Right (Supports immediate realignment) |
| <reg1> = <reg2> * <val> | Base Arithmetic | Hardware Multiplication (M-Extension) |
| <reg1> = <reg2> / <val> | Base Arithmetic | Hardware Division (M-Extension) |
| <reg1> = <reg2> % <val> | Base Arithmetic | Hardware Remainder (M-Extension) |
## 11. Comprehensive Code Examples
### Riscrithm Source Code
Example one:
``` text
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
Example two:
``` text
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
Example three:
``` text
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
Example four:
``` text
# orange_banana.txt
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
``` text
# horse_battery.txt
qux:
    foo swap bar
    foo ^^

quux:
    !!li x1, 10
    bar swap foo
    bar --
    foo ++
```
``` text
# main.txt
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
### Unoptimized RISC-V Assembly Output
Example one:
``` text
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
Example two:
``` text
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
Example three:
``` text
.section .text
.globl main
main:
   li x1, 7
   li x2, 7
   mul x1, x1, 1
   addi x2, x2, 0
   mul x1, x2, 1
   mul x1, x1, 8
   div x2, x2, 2
   beq x1, x2, equal_block
   j not_equal_block
equal_block:
   addi x1, x1, 1
   ecall
not_equal_block:
   addi x1, x1, -1
   ebreak
```
Example four:
``` text
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
### Optimized RISC-V Assembly Output (-o)
Example one:
``` text
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
Example two:
``` text
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
Example three:
``` text
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
Example four:
``` text
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
## Enjoy writing assembly without the traditional architecture headaches. Happy coding!
