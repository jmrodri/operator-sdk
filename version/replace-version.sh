#!/usr/bin/env bash

if [[ $# != 1 ]]; then
    echo "usage: $0 vX.Y.Z"
    exit 1
fi

VER=$1

if ! [[ "$VER" =~ ^v[0-9]+\.[0-9]+\.[0-9x]+(?:\+git)?$ ]]; then
    echo "malformed version: \"$VER\""
    exit 1
fi

#v[0-9]+\.[0-9]+\.[0-9x]+[\+git]*
sed -i "s/\"v[0-9]+\.[0-9]+\.[0-9x]+(?:\+git)?\"/\"${VER}\"/g" version.go
