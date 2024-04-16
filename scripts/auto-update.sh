#!/bin/bash
set -e
tsjDir=/home/vladwithcode/web/tsj
export TSJ_DIR=$tsjDir
export PATH=$PATH:/usr/local/go/bin

errorFile="$HOME/.local/log/auto-report.log/error"
logFile="$HOME/.local/log/auto-report.log/log"

cd $tsjDir

#go build -o "$tsjDir/cmd/auto-report/auto-report" "$tsjDir/cmd/auto-report/auto-report.go" >> $errorFile 2>&1
"$tsjDir/cmd/auto-update" >> $errorFile 2>&1
