#!/bin/bash
cd /home/vladwb/me/Dev/go/juzgados
go build -o cmd/auto-report/auto-report cmd/auto-report/auto-report.go 2>&1>/dev/null || awk '{print "Build Err: "$0}' | tee -a ~/.local/log/auto-report.log/error
cmd/auto-report/auto-report 2>&1>/dev/null || awk '{print "Exec Err: "$0}' | tee -a ~/.local/log/auto-report.log/error
