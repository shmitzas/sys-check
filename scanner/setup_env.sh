#!/bin/bash

#setup python
sudo apt install -y python3
sudo apt install -y python3-pip

#setup ansible
sudo apt update
sudo apt install -y software-properties-common
sudo add-apt-repository --yes --update ppa:ansible/ansible
sudo apt install -y ansible