.org 0x42
.text
# rando
.align 256 # 0x100
.globl       _start 
.local      deez
    add t0, a0, a1 # R instruction test
    addi t1, a1, 1110 # I instruction test
    sb t1, 10(a0) # S instruction test
    # tuff
.section .feet
wakanda:
    .asciz "retarded shi"

.data
.equ fab, 123
msg: 
    .asciz "hi\n"
zeros:
    .zero 20
array:
.word 0x3f800000, 0x3f800001, 0x3f800002