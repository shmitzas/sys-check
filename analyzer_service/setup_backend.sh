#!/bin/bash

sys_check_repo_location="$1"

#setup environment configuration files
mkdir ~/.sys-check/
mkdir ~/.sys-check/.env/
cp "${sys_check_repo_location}/analyzer_service/analyzer/.env.example" ~/.sys-check/.env/sys_check/analyzer.env
cp "${sys_check_repo_location}/analyzer_service/listener/.env.example" ~/.sys-check/.env/sys_check/listener.env
cp "${sys_check_repo_location}/analyzer_service/report_finalizer/.env.example" ~/.sys-check/.env/sys_check/report_finalizer.env

sudo snap install go