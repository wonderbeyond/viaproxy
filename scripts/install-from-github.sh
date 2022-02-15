#/usr/bin/env sh
set -eu

ver="0.0.1"
os="linux"
arch="amd64"
install_dir="/usr/local/bin"
bin_file="viaproxy"

set -x
curl -o- -L "https://github.com/wonderbeyond/viaproxy/releases/download/${ver}/${bin_file}-${ver}-${os}-${arch}.tar.gz" |
    tar xvzf /dev/stdin -C "${install_dir}" "${bin_file}"
