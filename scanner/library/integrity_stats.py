#!/usr/bin/python

import json
import socket
from ansible.module_utils.basic import AnsibleModule
import hashlib
import os
import pwd
import grp
import datetime
import threading
import requests

def search_files(starting_directory, depth=0, depth_limit=4):
    if depth > depth_limit:
        return []

    found_files = []
    excluded_extensions = ['.zip', '.rar', '.tar', '.gz', '.7z', '.bz2', '.xz', '.tar.gz', '.tar.bz2', '.tar.xz', '.tar.7z']
    for root, dirs, files in os.walk(starting_directory):
        for file in files:
            file_path = os.path.join(root, file)
            file_extension = os.path.splitext(file_path)[1].lower()

            if file_extension not in excluded_extensions:
                found_files.append(file_path)

        if depth < depth_limit:
            for dir in dirs:
                found_files.extend(search_files(os.path.join(root, dir), depth + 1))

        break

    return found_files


def search_directories(starting_directory, depth=0, depth_limit=4):
    if depth > depth_limit:
        return []

    found_directories = []

    for root, dirs, _ in os.walk(starting_directory):
        for dir in dirs:
            found_directories.append(os.path.join(root, dir))

            if depth < depth_limit:
                found_directories.extend(search_directories(os.path.join(root, dir), depth + 1))

        break

    return found_directories

def get_file_paths(directory):
    file_paths = []
    excluded_extensions = ['.tar', '.zip', '.rar', '.gz']

    def search_files(curr_dir):
        for root, dirs, files in os.walk(curr_dir):
            for file in files:
                file_path = os.path.join(root, file)
                file_extension = os.path.splitext(file_path)[1].lower()

                if file_extension not in excluded_extensions:
                    file_paths.append(file_path)

            for dir in dirs:
                next_dir = os.path.join(root, dir)
                search_files(next_dir)

    search_files(directory)
    return file_paths

def calculate_checksum(file, checksum_algorithm):
    hash_algo = hashlib.new(checksum_algorithm)
    try:
        with open(file, 'rb') as f:
            while True:
                try:
                    data = f.read(8192)
                    if not data:
                        break
                except:
                    return None
                hash_algo.update(data)
    except:
        return None
    return hash_algo.hexdigest()

def calculate_checksums(file):
    checksum_algorithms = [ 'MD5', 'SHA1', 'SHA256', 'SHA512' ]
    checksums = []
    for algo in checksum_algorithms:
        checksum = calculate_checksum(file, algo)
        if checksum != None:
            checksums.append(checksum)
    return checksums  

def get_file_details(file):
    full_path = os.path.abspath(file)
    file_name = os.path.basename(full_path)
    create_time = os.path.getctime(full_path)
    create_date = datetime.datetime.fromtimestamp(create_time).isoformat()
    modify_time = os.path.getmtime(full_path)
    modify_date = datetime.datetime.fromtimestamp(modify_time).isoformat()
    access_time = os.path.getatime(full_path)
    access_date = datetime.datetime.fromtimestamp(access_time).isoformat()
    file_stat = os.stat(full_path)
    owner = pwd.getpwuid(file_stat.st_uid).pw_name
    group = grp.getgrgid(file_stat.st_gid).gr_name
    permissions = oct(file_stat.st_mode)[-3:]
    size = os.path.getsize(full_path)
    
    file_details = {
        "path": full_path,
        "name": file_name,
        "created": create_date,
        "modified": modify_date,
        "accessed": access_date,
        "owner": owner,
        "group": group,
        "perm": permissions,
        "size": size
    }
    return file_details

def process_file(file):
    if os.path.exists(file) and os.path.isfile(file):
        file_details = get_file_details(file)
        checksums = calculate_checksums(file)
        if len(checksums) < 4:
            return None
        file_details['MD5'] = checksums[0]
        file_details['SHA1'] = checksums[1]
        file_details['SHA256'] = checksums[2]
        file_details['SHA512'] = checksums[3]
        return file_details
    return None

def get_metadata():
    metadata = {
    'hostname': socket.gethostname(),
    'ip_address': socket.gethostbyname(socket.gethostname())
    }
    return metadata

def check_files_integrity(file_list):
    payload_data = {
    "files" : file_list,
    "metadata" : get_metadata(),
    "status" : "processing"
    }
    
    send_integrity_request(payload_data)

def send_integrity_request(payload):
    url = f'http://{service_host}:{service_port}'
    json_payload = json.dumps(payload)
    headers = {'Content-Type': 'application/json'}
    response = requests.post(url, data=json_payload, headers=headers)

    if response.status_code != 200:
        print('Request failed:', response.status_code)

def process_root_dir(directory):
    threads = []
    
    partial_files = search_files(directory)
    process_file_paths(partial_files)
    
    thread = threading.Thread(target=process_file_paths, args=(partial_files,))
    threads.append(thread)
    thread.start()
    
    sub_dirs = search_directories(directory)   
    for dir in sub_dirs:
        thread = threading.Thread(target=process_dir, args=(dir,))
        threads.append(thread)
        thread.start()

    for thread in threads:
        thread.join()

def process_dir(directory):
    file_paths = get_file_paths(directory) 
    process_file_paths(file_paths)

def process_file_paths(file_paths):
    results = []
    for file in file_paths:
            processed_file = process_file(file)
            if processed_file != None:
                results.append(processed_file)
                
    if len(results) > 0:
        check_files_integrity(results)

def main():
    global service_host
    global service_port
    module = AnsibleModule(
        argument_spec=dict(
            directories=dict(type='list', required=True),
            service_host=dict(type='str', required=True),
            service_port=dict(type='int', required=True),
        )
    )
    
    dirs = module.params["directories"]
    service_host = module.params['service_host']
    service_port = module.params['service_port']
    
    for dir in dirs:
        process_root_dir(dir)

    # Sends a signal indicating that all data was sent out
    payload_data = {
    "files" : [],
    "metadata" : get_metadata(),
    "status" : "final"
    }

    send_integrity_request(payload_data)

    module.exit_json(changed=False)

if __name__ == '__main__':
    main()
