#!/bin/bash
tsjDir=$TSJ_DIR
cd $tsjDir

go build -o cmd/auto-report/auto-report cmd/auto-report/auto-report.go
cmd/auto-report/auto-report
