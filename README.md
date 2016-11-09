spipe tools
===========

a simple spipe daemon and client aswell as a spipe netcat clone, written in golang.

## spipeKeygen

generates a new spipe suitable key, writes to _spipe.key_ in local directory

```spipeKeygen```

## spiped

start a spipe listener on 80.244.247.218:8888 and forward to 80.244.247.5:80
	```spiped -m listen_forward -h 80.244.247.218 -p 8888 -forward 80.244.247.5:80 -k spipe.key```

start a plaintext listener on 80.244.247.5:8080 and forward to spipe endpoint 80.244.247.218:8888 
		```spiped -m dial_forward -h 80.244.247.5 -p 8080 -forward 80.244.247.218:8888 -k spipe.key```

recieve a file via spiped on 80.244.247.218:8080

		```spiped -m listen -h 80.244.247.218 -p 8080 -k spipe.key > file```

send a file via spiped to 80.244.247.218:8080

	```cat file | spiped -m dial -h 80.244.247.218 -p 8080 -k spipe.key```

## spipecat

Simple Netcat like tool for spipes.

### spipecat usage examples

Set up a spipe listener on port 127.0.0.1:8080

 ```$> spipecat -m listen -k MyLittleSecret -h 127.0.0.1 -p 8080```

Connect to the listener set up above

 ```$> spipecat -m dial -k MyLittelSecret -h 127.0.0.1 -p 8080```


Recieve a file

 ```$> spipecat -m listen -k MyLittleSecret -h 127.0.0.1 -p 8080 > myfile.dat```

Send a file

 ```$> cat myfile.dat | spipecat -m dial -k MyLittelSecret -h 127.0.0.1 -p 8080```


## Weblinks

The original spiped is at: http://www.tarsnap.com/spiped.html
