.org 0x40
.text # 0x40 aligns to 0x50
# rando
.align 128 # 256 bits = 32 bytes
.globl       _start 
.local      deez
_start: 
    add t0, a0, a1 # R instruction test
    addi t1, a1, 1110 # I instruction test
    sb t1, 10(a0) # S instruction test
    # tuff
deez:
    beq, t0, a2

.section .feet
wakanda:
    .asciz "dumb shi" # char is 1 byte. 

.data # 8 + 8 + 8 + 20 + 8 + 12 = 64
.equ fab, 123
msg: 
    .asciz "hi\n"
zeros:
    .zero 20 # 20 bytes of zeros
array:
.word 0x3f800000, 0x3f800001, 0x3f800002 # 12 bytes