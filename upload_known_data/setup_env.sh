#!/bin/bash

sys_check_repo_location="$1"

#setup environment configuration files
mkdir /etc/sys_check/
cp "${sys_check_repo_location}/upload_known_data/.env.example" /etc/sys_check/upload_data.env

#setup golang
sudo snap install go

#setup python
sudo apt install python3
sudo apt install python3-pip
sudo pip install -r "${sys_check_repo_location}/upload_known_data/upload_nsrl_data/formatter/requirements.txt"