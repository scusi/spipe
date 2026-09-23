spipe tools
===========

a simple spipe daemon and client aswell as a spipe netcat clone, written in golang.
A tool to generate keys suitable for spipe tools is also included.

All tools are under the _cmd_ directory.

## spipeKeygen

generates a new spipe suitable key, writes to _spipe.key_ in local directory.

Instead of `spipeKeygen` you can simply use:
```
openssl rand 32 > spipe.key
```

## spiped

### Quickstart

`spiped` is command line compatible with the [original spiped](https://manpages.debian.org/testing/spiped/spiped.1.en.html): it takes encrypted connections on the source socket (`-d`) or sends encrypted connections to the target socket (`-e`), using the options `-s`, `-t`, `-k`, `-F` and more.
Together with the provided systemd service file you can protect your sshd with spiped.

Change your `/etc/ssh/sshd_config` to listen only to 127.0.0.1:22

```
# Nur localhost (IPv4 und IPv6)
ListenAddress 127.0.0.1
ListenAddress ::1
```

Build spiped and spipeKeygen, or download from [release page].

```
go build ./cmd/spiped
go build ./cmd/spipeKeygen
```

Then install spiped and a service file to have spiped listen on `*:8022` and forward to localhost:22
```
sudo cp spiped /usr/local/bin/spiped
sudo cp examples/spiped-ssh.service /etc/systemd/system/
sudo ./spipeKeygen -o /etc/ssh/spiped.key
sudo systemctl daemon-reload
sudo systemctl enable spiped-ssh.service
sudo systemctl start spiped-ssh.service 
```

### Example Usage

take encrypted connections on 80.244.247.218:8888 and forward unencrypted to 80.244.247.5:80

```
spiped -d -s 80.244.247.218:8888 -t 80.244.247.5:80 -k spipe.key
```

take unencrypted connections on 80.244.247.5:8080 and forward them encrypted to the spipe endpoint 80.244.247.218:8888

```
spiped -e -s 80.244.247.5:8080 -t 80.244.247.218:8888 -k spipe.key
```

run in the foreground, useful with systemd or daemontools

```
spiped -d -s [::]:8022 -t 127.0.0.1:22 -k spipe.key -F
```

use a unix domain socket as target

```
spiped -d -s [::]:8022 -t /run/sshd.sock -k spipe.key -F
```

limit the number of simultaneous connections and disable keep-alives

```
spiped -d -s [::]:8022 -t 127.0.0.1:22 -k spipe.key -F -n 16 -j
```

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
