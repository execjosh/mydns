# mydns

`mydns` is a simple and opinionated blocklisted DNS stub resolver for use
with small networks that can be run on low-power, single-board computers such
as the ARMv6-based [Raspberry Pi Zero W][rpi0w]. It should Just Work™
out-of-the-box with minimal configuration.

This project aims to have as few external dependencies as possible by being a
single, stand-alone, static binary. It is currently a work in progress, and
as such it basically does what the maintainer needs it to do.

The program is implemented to hold everything in memory in order to minimize
disk access. As such, YMMV depending on how much memory your system has and
how large your blocklist is and how many glob patterns you have.

[rpi0w]: https://www.raspberrypi.org/products/raspberry-pi-zero-w

## Installation

```bash
go get github.com/execjosh/mydns
```

## How to Run

```bash
mydns \
    -nameservers 1.1.1.1,1.0.0.1 \
    -tls-server-name cloudflare-dns.com \
    -udp 1337 \
    -blocklist example/block.list
```

Example queries:

```
$ dig @127.0.0.1 -p 1337 sub1.example.com +short
0.0.0.0
$ dig @127.0.0.1 -p 1337 sub2.example.com +short
0.0.0.0
$ dig @127.0.0.1 -p 1337 sub3.sub2.example.com +short
0.0.0.0
$ dig @127.0.0.1 -p 1337 example.com +short
93.184.216.34
```

## Flags

A comma-separated list of upstream `-nameservers` must be specified. An
upstream nameserver is automatically chosen using round-robin upon each
request. Be aware that there are no healthcheks for upstream nameservers.

Either `-tcp` or `-udp` must be specified. You may specify both. If multiple
`-tcp` or multiple `-udp` are specified, the last value will be used
respectively.

Optionally, a blocklist file may be specified with `-blocklist`.

## Blocklist File Format

The blocklist file contains one (`1`) fqdn per line. The whole blocklist is
loaded into memory.

See example below or have a look at the [example blocklist
file](https://github.com/execjosh/mydns/tree/master/example/block.list):

```
sub1.example.com
sub2.example.com
sub3.*.example.com
```
