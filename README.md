# Phissembler
Risc-V Assembler because macos doesn't have one : (. This assembler will be created in Go and will generate a bin file that can be executed on the CPU.

## Future Plans
- assembler also creates .elf file
- assembler works for multiple files
- create linker to work with assembler

## How it works
Assembly is made up of 3 components: directives, labels, and instructions. The assembler is in charge of parsing these assembly from all the instructions, and mapping them directly to binary that can be directly executed on the CPU. This of course changes depending on what ISA and machine that the assembler is on. For example, my current machine is a 64-bit ARM CPU, so all instructions are of length 64 bits and follow the ARM ISA. On the official ARM website it does say that ARM only supports 32-bit addressing range, but this is a bit deceiving. In reality, ARM is split into 2 ISAs: AArch32 and AArch64. As the name suggests, AArch64 is the 64-bit range version of ARM and AArch32 is the 32-bit range version of ARM. If more background is needed on understanding the difference between 32-bit and 64-bit machines, then I suggest googling virtual addressing and register length differences between the two types of instruction sets. Otherwise, I will continue explaining the how an assembler works assuming the reader knows this background already.  

### Understanding Assembly Components
Before diving into how the assembler works, some more background information is needed on the different types of components that can make up assembly. As stated before assembly can be split up into directives, instructions, and labels
#### Directives
Directives can be many things

### General Assembler Architecture
The assembler is split into 3 different processes: Basic cleaning, First Pass, and Second Pass. There are such things as 1-pass assemblers, but in my implementation, I chose to implement a 2-pass assembler. I will dive more into what is done in each pass within each description. 
1. Basic cleaning of the assembly file: This means removing all the white spaces such that there is only one space between every word, removing, comments, removing new lines, etc. This is an important step so that parsing in the next steps work correctly.
2. First Pass: This first pass loops through every line and records which section every assembly instruction and directive is in and recording the offset address each component is from the section address. The size of each section is recorded, and  For example if 
3. Second Pass: The second pass is in charge of actually generating the binary executable and 

## Useful Background Knowledge
<b>Little Endian vs Big Endian:</b> Little Endian is the normal way of addressing bytes for CPUs. The difference is the order of addressing the most significant bytes. For example, with the number 4660 the binary we read is 0b00010010 00110100. Suppose we have 0x0 as address 1 and 0x1 as address 2. Then in big endian, 0x0 = 0b00010010 and 0x1 = 0b00110100 as the most significant bytes are at the lowest memory address, but in little endian 0x0 = 0b00110100 and 0x1 = 0b00010010, since the least significant bytes are at the lowest memory address. This all supposes addresses are byte addressed.

## Useful Links
ARM instruction set + descriptions = https://iitd-plos.github.io/col718/ref/arm-instructionset.pdf 
