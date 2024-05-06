#!/bin/bash
set -e
tsjDir=/home/vladwithcode/web/tsj
export TSJ_DIR=$tsjDir
export PATH=$PATH:/usr/local/go/bin

errorFile="$HOME/.local/log/tsj/auto-update.daily.log"

cd $tsjDir

/home/vladwithcode/web/tsj/cmd/auto-update -d 60 >> $errorFile 2>&1
