#!/bin/bash

sys_check_repo_location="$1"
user=$(logname)

#setup environment configuration files
mkdir "/home/${user}/.sys-check/"
mkdir "/home/${user}/.sys-check/.env/"
mkdir "/home/${user}/.sys-check/reports/"
cp "${sys_check_repo_location}/analyzer_service/analyzer/.env.example" "/home/${user}/.sys-check/.env/analyzer.env"
cp "${sys_check_repo_location}/analyzer_service/listener/.env.example" "/home/${user}/.sys-check/.env/listener.env"
cp "${sys_check_repo_location}/analyzer_service/report_finalizer/.env.example" "/home/${user}/.sys-check/.env/report_finalizer.env"

sudo apt install -y golang-go