#!/usr/bin/env bash

git describe --tags --abbrev=0 > version.txt
go install
rm version.txt
