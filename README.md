# The Riscrithm Developer Manual
Hey there. If you're looking at this, you are probably getting your hands dirty with **Riscrithm**, a high-level macro-assembly dialect that compiles straight down to pure RISC-V assembly. Think of it as a bridge between the readability of a high-level language and the raw, deterministic control of bare-metal hardware.
Let's dive straight into how the compiler works, the syntax rules, and what's happening under the hood.
## 1. The CLI
To compile your source code, you'll use the riscrithm CLI tool. The syntax is straightforward:
```bash
riscrithm "source_code_file" "assembly_target_file" [-o/--optimize]

```
 * **Source Code:** Your Riscrithm input file.
 * **Target File:** The generated .s assembly file. If this file doesn't exist, the compiler will create it for you on the fly.
 * **Optimization:** Pass -o or --optimize to enable the optimization sweep (more on the compiler architecture later).

## 2. File Structure & Globals
Every Riscrithm file must declare its target section and entrypoint at the very top. These, along with macro definitions, are the *only* lines allowed to exist completely unindented outside of a label block.
### Header and Entrypoint
 * **header <value>**: Sets the assembly section. For instance, header default translates to .section .text.
 * **entrypoint <symbol>**: Defines where the program starts. Passing entrypoint main translates to .globl main.
```text
header default
entrypoint main

```
### Definitions (Macros)
You can define text-replacement macros using the define keyword. This is perfect for aliasing registers or creating single-line inline functions. Here are some classic developer examples:
```text
define foo = x1
define bar = x2
define baz = x3
define horseBattery = x4
define apple = 10
define orange = 20
define clearFoo = foo ^^

```
Whenever the parser sees foo, it swaps it with x1 *before* processing any actual logic.
### Comments
Comments are written using the # symbol. The compiler strips out anything following a # on any line, so you can place them anywhere safely.

## 3. Labels, Indentation, and Raw Blocks
Riscrithm is strictly scoped via indentation.
### Standard Labels
Labels define your execution blocks and must end with a colon. They **must not** have any indentation.
Conversely, every instruction inside a label **must** be indented (spaces or tabs). If you leave an instruction unindented, the compiler will throw a SyntaxError.
```text
main:
    load foo = apple
    move bar = foo

```
### Raw Assembly Labels (!!)
If you need to bypass the Riscrithm preprocessor and write raw RISC-V assembly, prefix your label with !!. The compiler strips the exclamation marks but passes everything inside that block completely untouched. Macros and shorthands will *not* expand here.
```text
!!raw_block:
    li x1, 10
    foo ^^ # This stays exactly as written!

```

## 4. Core Features & Instructions
Here is the meat of the language. Riscrithm maps readable statements directly to hardware instructions.
### System & Interrupt Controls
Instead of remembering privilege-level opcodes, use explicit system calls:
| Riscrithm | RISC-V Assembly | Description |
|---|---|---|
| interrupt.u | uret | User-mode trap return |
| interrupt.s | sret | Supervisor-mode trap return |
| interrupt.m | mret | Machine-mode trap return |
| wait | wfi | Wait for interrupt (low-power state) |
| trap | ebreak | Debugger trap |
| halt | ecall | System environment call / halt |
| ... | nop | No-operation (ellipsis) |

## 5. Naming Conventions
Let’s talk about code style. To keep your Riscrithm source files readable and consistent, the compiler expects (and highly encourages) a clean split in how you name your identifiers. Here is the naming convention breakdown:
 * **Variables & Registers (camelCase):** Any variable alias or register macro you define should start with a lowercase letter, with each subsequent word capitalized.
   * *Examples:* firstNum, addressRegister, stackOffset
 * **Labels & Code Blocks (snake_case):** Execution targets, loop boundaries, and conditional blocks use lowercase words separated by underscores. This makes them pop visually against instructions.
   * *Examples:* loop_start, on_true, error_handler
 * **Constants & Literals (SCREAMING_SNAKE_CASE):** Hardcoded configuration values, static offsets, or global definitions that shouldn't change use all-uppercase letters separated by underscores.
   * *Examples:* DEFAULT_HEADER, MAX_BUFFER_SIZE, IMM_VALUE

## 6. Complete Operator & Expression Reference
Excluding the hardware system traps and conditional branching symbols, here is the complete table of mutators, arithmetic expressions, and memory operators supported by the single-pass compiler engine.
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
This section covers basic math operations, self-mutators, and compound shorthands. Remember, the compiler automatically realigns regular operations to their immediate equivalent (addi, andi, etc.) if the right-hand side is an integer literal.
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

### Branching and Conditionals
To unconditionally jump, use the @ symbol:
```text
@some_label # Compiles to: j some_label

```
For conditional branching, Riscrithm uses an inline if/else ternary style. The compiler automatically maps your logic to beq, bne, blt, or bge, and will even swap registers dynamically to handle > and <=.
```text
if foo == bar @true_block else @false_block
if foo > baz @greater_block else @lesser_block

```
### Loops (Infinite and Conditional)
Riscrithm doesn't have a dedicated while or for keyword because you don't need them. You build loops the old-school way using labels, conditionals, and jumps.
**An Infinite Loop:**
```text
infinite_loop:
    foo ++
    @infinite_loop

```
**A Conditional Loop:**
```text
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
### Operations and Mutators
Riscrithm supports immediate assignments and compound math shorthands. The compiler is smart enough to append the i suffix (e.g., addi, xori) when it detects you are working with an immediate integer instead of a register.
 * **Load/Move:** load foo = 100, move bar = foo
 * **Math:** foo += 5, bar *= baz, foo <<= 2
 * **Increments:** foo ++ (addi foo, foo, 1), bar -- (addi bar, bar, -1)
> **The ^^ Shorthand:** Want to clear a register fast? Use the XOR-self operator ^^.
> foo ^^ translates to xor foo, foo, foo, immediately zeroing out the register.
> 
### Swapping Variables
Need to swap two registers without a temporary third register? The swap command uses a non-destructive triple-XOR sequence:
```text
foo swap bar

```
Translates to:
```text
xor foo, foo, bar
xor bar, foo, bar
xor foo, foo, bar

```

## 7. Memory Operations (Stack & Heap)
Memory interaction requires strict data width extensions: .b (byte/8-bit), .w (word/32-bit), or .d (double-word/64-bit).
### Stack Operations
Stack commands automatically adjust the hardware stack pointer (sp) by the correct byte offset.
 * **Push (->):** foo -> stack.w (Decrements sp by 4, stores word)
 * **Pop (<-):** bar <- stack.d (Loads double-word, increments sp by 8)
 * **Peek (=):** baz = stack.b (Loads byte without moving sp)
### Heap Operations
Heap commands require you to provide a base address register using the & pointer syntax.
 * **Store (->):** foo -> heap.w from &bar (Stores word from foo into address at bar)
 * **Load (<-):** baz <- heap.b from &foo (Loads byte into baz from address at foo)

## 8. Compound Snippet Example
Here is what a cohesive block of Riscrithm looks like with these features combined:
```text
main:
    # Setup
    load foo = 10
    load bar = 20
    baz ^^

    # Math and Memory
    foo += 5
    foo -> stack.w
    bar *= foo
    baz <- heap.w from &bar

    # Branching
    if foo != bar @continue else @fail

continue:
    foo swap bar
    halt

fail:
    trap

```

## 9. The Compiler Architecture & Optimizer (-o / --optimize)
Let's clear something up: Riscrithm is not some bloated, complex multi-pass optimization engine. It operates on a lightning-fast **two-pass system**:
 1. **Pass 1 (Sanitization):** The compiler reads the source file, strips out all comments, standardizes the whitespace, and verifies the strict indentation rules. It gets the raw text completely clean.
 2. **Pass 2 (Parse & Optimize):** This is where the magic happens in a *single pass*. It parses the instructions, replaces macros, expands shorthands, and—if the -o flag is active—applies optimizations on the fly before writing the assembly.
When you compile with -o or --optimize, this second pass applies a lightweight AST sweep that cleans up your code in three distinct ways:
 * **Dead Assignment Elimination:** Consecutive duplicate modifications or redundant load/move sequences to the same register are discarded. (e.g., calling load foo = 128 twice in a row results in only one instruction).
 * **Identity Math Elimination:** Mathematical operations that leave the value completely unchanged are dropped entirely if the destination matches the source register (e.g., foo = foo + 0 or bar = bar / 1 are deleted).
 * **Strength Reduction (Bitwise Folding):** Multiplication and division are computationally expensive. If the optimizer catches you multiplying or dividing by a static power of two, it intercepts the instruction and rewrites it as a highly efficient bit-shift.
   * foo = bar * 2 translates to slli foo, bar, 1 (Shift Left Logical).
   * baz = foo / 8 translates to srli baz, foo, 3 (Shift Right Logical).

## 10. Clean, Ready-to-Use Output
One of the best parts about Riscrithm is that the assembly file it spits out isn't an unreadable mess. The output .s file is automatically pretty-printed. Instructions inside blocks are neatly indented, labels sit flush to the margin, and the entire structure is completely human-readable.
You can take the generated assembly and drop it directly into your hardware simulator, debugger, or desktop workflow without formatting a thing.
Enjoy writing assembly without the headache. Happy coding!

## 11. Roadmap: What’s Brewing for v1.1.0?
​Let’s be real—building a language alone is an iterative grind. While my current two-pass compiler engine handles the heavy lifting by separating symbol resolution from code generation, I am already actively breaking things behind the scenes to bring you a much more robust DX.
​Here is what I am cooking up for the v1.1.0 release:
​Proper Module Imports: Right now, splitting code across multiple files is a headache. I am working on a dedicated import system so you can natively break your codebase down into clean, reusable modules without breaking the build pipeline.
​Better Error Handling: I know the current compiler diagnostics can be... cryptic. The next minor release will introduce accurate line/column tracking and actual, human-readable error messages instead of just blowing up your terminal.
​Guard Clauses & Simple if Statements: You shouldn't be forced to write an empty else block just to satisfy the parser. I am updating the AST to natively support standalone if branches for cleaner, early-return guard patterns.
​Contribution & Feedback
​Have ideas for the syntax, or found an edge-case bug that completely broke the register allocation? Open an issue or drop a PR. This project is built by a developer, for developers—let's make it better together.
