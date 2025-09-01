# Phissembler
Risc-V Assembler because macos doesn't have one : (. This assembler will be created in Go and will generate a bin file with virtual addressing. Currently it needs to be converted to ELF file format, but this will be implemented later.

## Future Plans
- assembler also creates .elf file
- assembler works for multiple files
- create linker to work with assembler

## How it works
Assembly is made up of 3 components: directives, labels, and instructions. The assembler is in charge of parsing these assembly from all the instructions, and mapping them directly to binary that can be directly executed on the CPU. This of course changes depending on what ISA and machine that the assembler is on. For example, my current machine is a 64-bit ARM CPU, so all instructions are of length 64 bits and follow the ARM ISA. On the official ARM website it does say that ARM only supports 32-bit addressing range, but this is a bit deceiving. In reality, ARM is split into 2 ISAs: AArch32 and AArch64. As the name suggests, AArch64 is the 64-bit range version of ARM and AArch32 is the 32-bit range version of ARM. If more background is needed on understanding the difference between 32-bit and 64-bit machines, then I suggest googling virtual addressing and register length differences between the two types of instruction sets. Otherwise, I will continue explaining the how an assembler works assuming the reader knows this background already.  

### Understanding Assembly Components
Before diving into how the assembler works, some more background information is needed on the different types of components that can make up assembly. As stated before assembly can be split up into directives, instructions, and labels

#### Sections
There are 4 main different sections that assembly can be apart of: .text, .data, .rodata, and .bss (additional ones include .ctor, .plt, etc. but aren't implemented in this assembler). If .text is not specified, then it is assumed that all the instructions are apart of that section. The .text contains all the instructions, .data contains initalized writable globals, .rodata contains read only data, .bss contains uninitialized global variables. Later, the linker uses those sections to lay out the final executable, and the loader maps them into memory at runtime.

#### Directives
Directives can be many things
Description of what each directive does: https://developer.arm.com/documentation/den0013/0400/Introduction-to-Assembly-Language/Introduction-to-the-GNU-Assembler/Assembler-directives

#### Instructions
There are 7 different types of instruction types that I implemented in the assembler. I chose not to implement certain instructions like floating point and vectorization instructions, because I did not think it was worth my time to implement 150 more instructions. The instructions consist of different fields like opcode, funct3, funct5, funct7, & immediates. Generally instructions of the same type have the same opcode but differ in fields like funct7. For example, ADD and SUB instructions have the same opcode and funct 3 but only differ in funct7 where ADD has 0000000 and SUB has 0100000.

### General Assembler Architecture
The assembler is split into 3 different processes: Basic cleaning, First Pass, and Second Pass. There are such things as 1-pass assemblers, but in my implementation, I chose to implement a 2-pass assembler. I will dive more into what is done in each pass within each description. 
<b>1. Basic cleaning of the assembly file:</b> This means removing all the white spaces such that there is only one space between every word, removing, comments, removing new lines, etc. This is an important step so that parsing in the next steps work correctly.
<b>2. First Pass:</b> This first pass loops through every line and records which section every assembly instruction and directive is in and recording the offset address each component is from the section address. The size of each section is recorded.
<b>3.</b> Second Pass: The second pass is in charge of actually generating the binary executable. This means translating all the instructions into binary and adjusting address depending on directive type. For example .org sets the address of the next instruction to be at an exact offset. This will change the next address and fill the rest of the address with no ops.

## Useful Background Knowledge
<b>Little Endian vs Big Endian:</b> Little Endian is the normal way of addressing bytes for CPUs. The difference is the order of addressing the most significant bytes. For example, with the number 4660 the binary we read is 0b00010010 00110100. Suppose we have 0x0 as address 1 and 0x1 as address 2. Then in big endian, 0x0 = 0b00010010 and 0x1 = 0b00110100 as the most significant bytes are at the lowest memory address, but in little endian 0x0 = 0b00110100 and 0x1 = 0b00010010, since the least significant bytes are at the lowest memory address. This all supposes addresses are byte addressed.
<br>
<b>Memory Layout of Process:</b> It is important to note that process memory is apart of the virtual memory, and different sections are mapped to pages in physical memory depending on how the OS maps them. Essentially, the OS loader sets a space in virtual memory for the process memory, and it is set up in the order specified by the ELF file. This is typically a NULL guard page, .text, .rodata, .data, .bss, heap that grows up, memory mapped region for shared libraries, and stack that grows down. This is all contiguous in virtual memory, but under the hood, the various parts of the process may be mapped to different physical pages.
<br>
<b>.ELF File Execution Process:</b> A useful thing to note is how process is actually run. When you run an ELF binary, the shell first calls fork() to create a child process. In the child, execve(path, argv, envp) is invoked. Inside the kernel, the old process image is discarded, and the ELF loader reads the ELF headers to determine how to map the executable’s segments into the process’s address space. It sets up the .text, .data, .bss, stack, heap, and auxiliary vectors (info about execution environment i.e. entry point address, address of program's ELF program header's in memory, etc.). Physical memory isn’t filled immediately; instead, the kernel prepares virtual memory mappings and uses demand paging. Finally, the kernel sets the CPU’s program counter to the ELF entry point, switches back to user mode, and execution begins in the new program.


## Useful Links
ARM instruction set + descriptions = https://iitd-plos.github.io/col718/ref/arm-instructionset.pdf 
2 Pass Assembler = https://kshitij-taley19.medium.com/two-pass-assembler-3c193c612cb8 