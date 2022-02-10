viaproxy
========

Run any shell command in a temporary proxy environment.

---

## Usage

```shell
$ viaproxy socks5://127.0.0.1:1080 run curl -L https://www.google.com
```

```shell
$ viaproxy http://192.168.1.9:8888 run psql ...
```

```shell
$ viaproxy socks5://127.0.0.1:1080 run bash
# Got into a new shell
$ curl -L https://www.google.com
```

Planning:

```shell
$ viaproxy ssh://192.168.10.100:22 run curl -L https://www.google.com
```

## Build

```
$ git submodule update
$ make -C graftcp
$ go build
# make -C graftcp clean
```
