.org 0x42
.text
# rando
.globl       _start 
.local      deez
    add t0, a0, a1 # R instruction test
    addi t1, a1, 1110 # I instruction test
    # tuff


.data
msg: 
    .asciz "hi\n"