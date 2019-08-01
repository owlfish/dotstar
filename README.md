# dotstar - Go library for controlling DotStar LED strings

DotStar is a brand of LED lights that are controllable through a generic 2-wire SPI.  I've driven them from a Raspberry PI using wiring instructions from this https://learn.adafruit.com/dotstar-pi-painter/assembly-part-1 article.

This library provides a simple interface for changing the colours of the LEDs, while taking care of gamma correction, byte write orders and footers.

Hopefully someone will find it useful!

