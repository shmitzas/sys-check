#!/bin/bash

sys_check_repo_location="$1"

#setup environment configuration files
mkdir /tmp/sys-check/
mkdir /tmp/sys-check/.env/
cp "${sys_check_repo_location}/upload_known_data/.env.example" /tmp/sys-check/.env/upload_data.env

#setup golang
sudo apt install -y golang-go

#setup python
sudo apt install -y python3
sudo apt install -y python3-pip
sudo pip install -r "${sys_check_repo_location}/upload_known_data/upload_nsrl_data/formatter/requirements.txt"