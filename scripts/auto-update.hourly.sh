#!/bin/bash
set -e
tsjDir=/home/vladwithcode/web/tsj
export TSJ_DIR=$tsjDir
export PATH=$PATH:/usr/local/go/bin

errorFile="$HOME/.local/log/tsj/auto-update.log"

cd $tsjDir

/home/vladwithcode/web/tsj/cmd/auto-update >> $errorFile 2>&1
