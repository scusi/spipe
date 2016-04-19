spipe tools
===========

a simple spipe daemon and client aswell as a spipe netcat clone, written in golang.

Usage
=====

Set up a spipe listener on port 127.0.0.1:8080

 ```$> spipecat -m listen -k MyLittleSecret -h 127.0.0.1 -p 8080```


Connect to the listener set up above

 $> spipecat -m dial -k MyLittelSecret -h 127.0.0.1 -p 8080


Recieve a file

 $> spipecat -m listen -k MyLittleSecret -h 127.0.0.1 -p 8080 > myfile.dat

Send a file

 $> cat myfile.dat | spipecat -m dial -k MyLittelSecret -h 127.0.0.1 -p 8080

Weblinks
========
The original spiped is at: http://www.tarsnap.com/spiped.html
