#!/bin/bash

sys_check_repo_location="$1"

#setup environment configuration files
mkdir /tmp/sys-check/
mkdir /tmp/sys-check/.env/
cp "${sys_check_repo_location}/analyzer_service/analyzer/.env.example" /tmp/sys-check/.env/analyzer.env
cp "${sys_check_repo_location}/analyzer_service/listener/.env.example" /tmp/sys-check/.env/listener.env
cp "${sys_check_repo_location}/analyzer_service/report_finalizer/.env.example" /tmp/sys-check/.env/report_finalizer.env

sudo apt install -y golang-go