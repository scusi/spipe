# LoginShell

a login shell via spipe.

This LoginShell is not meant for production use. It is more like a proof of concept.

Without the correct spipe key it is not possible to connect to this LoginShell.
After connect you also need to have valid credentials for an account on the host.

## build requirements

In order to build this login shell you need to have libpam development files 
installed on your build system.

- on debian based systems install with: ```sudo apt install libpam0g-dev```.
- on red-hat based systems install with: ```sudo yum install pam-devel```.

## build

```
cd shells/LoginShell/
go build .
```

## known limitations

- Echo for password input is not set off. 
  So your password will be echoed when entered. 
  Be aware of shoulder surfers.
- After successful authentication you have a shell, but no prompt.
- PAM authentication only works for the user the LoginShell was started by.
  It works for all available systemusers when you start the shell as root,
  which is not recommended. 

## Quick Start Guide

In one terminal execute the loginshell:
```
go run shells/LoginShell/spipeLoginBindShell.go
```

In another terminal use spipenetcat to connect to it:

```
foo@testbox:~/sources/spipe$ go run cmd/spipecat/spipecat.go -p 7777
login: foo 
Password:  1q2w3e4r5t
Authentication succeeded!
starting a shell...
ls -al
insgesamt 44
drwxrwxr-x   7 foo foo 4096 Sep 16 15:06 .
drwxrwxr-x 164 foo foo 4096 Sep 13 16:45 ..
drwxrwxr-x   6 foo foo 4096 Sep 16 14:31 cmd
drwxrwxr-x   2 foo foo 4096 Apr 18  2016 dev
drwxr-xr-x   7 foo foo 4096 Sep 16 14:38 dist
drwxrwxr-x   8 foo foo 4096 Sep 16 15:07 .git
-rw-rw-r--   1 foo foo   10 Sep 16 14:36 .gitignore
-rw-rw-r--   1 foo foo 1093 Sep 16 14:28 .goreleaser.yml
-rw-rw-r--   1 foo foo 1511 Sep 16 14:33 README.md
drwxrwxr-x   5 foo foo 4096 Nov 10  2016 shells
-rw-------   1 foo foo   32 Nov  9  2016 spipe.key
quit
2019/09/16 15:23:47 Remote connection is closed
```
