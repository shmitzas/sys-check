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
import queue
from dotenv import load_dotenv
import requests

def get_file_paths(directory):
    file_paths = []
    excluded_extensions = ['tar', 'zip', 'rar', '.gz']
    
    for root, dirs, files in os.walk(directory):
        for file in files:
            file_path = os.path.join(root, file)
            file_extension = os.path.splitext(file_path)[1].lower()
            
            if file_extension not in excluded_extensions:
                file_paths.append(file_path)
    
    return file_paths

def calculate_checksum(file, checksum_algorithm, results_queue):
    hash_algo = hashlib.new(checksum_algorithm)
    try:
        with open(file, 'rb') as f:
            while True:
                try:
                    data = f.read(8192)
                    if not data:
                        break
                except:
                    return 'An error occured while calculating checksum'
                hash_algo.update(data)
    except:
        return 'Checksum cannot be calculated due to insufficient permissions'
    results_queue.put(hash_algo.hexdigest()) 

def calculate_checksums(file):
    checksum_algorithms = [ 'MD5', 'SHA1', 'SHA256', 'SHA512' ]
    checksums = []
    results_queue = queue.Queue()
    threads = []
    for algo in checksum_algorithms:
        thread = threading.Thread(target=calculate_checksum, args=(file, algo, results_queue))
        threads.append(thread)
        thread.start()

        for thread in threads:
            thread.join()

        while not results_queue.empty():
            checksums.append(results_queue.get())
    return checksums   

def get_file_details(file):
    full_path = os.path.abspath(file)
    file_name = os.path.basename(full_path)
    create_time = os.path.getctime(full_path)
    create_date = datetime.datetime.fromtimestamp(create_time)
    modify_time = os.path.getmtime(full_path)
    modify_date = datetime.datetime.fromtimestamp(modify_time)
    access_time = os.path.getatime(full_path)
    access_date = datetime.datetime.fromtimestamp(access_time)
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

def to_batches(initial_list, batch_size):
    batches = []
    for i in range(0, len(initial_list), batch_size):
        batch = initial_list[i:i+batch_size]
        batches.append(batch)
    return batches

def process_file(file):
    if os.path.exists(file) and os.path.isfile(file):
        file_details = get_file_details(file)
        checksums = calculate_checksums(file)
        file_details['MD5'] = checksums[0]
        file_details['SHA1'] = checksums[1]
        file_details['SHA256'] = checksums[2]
        file_details['SHA512'] = checksums[3]
        return file_details
    return None

def process_batch(batch, results_queue):
    result = []
    for file in batch:
        processed_file = process_file(file)
        if processed_file != None:
            result.append(processed_file)
    results_queue.put(result)

def get_ipv4_address():
    hostname = socket.gethostname()
    ip_address = socket.gethostbyname(hostname)
    return ip_address

def check_files_integrity(file_list):
    payload_data = {}
    payload_data['files'] = file_list
    payload_data['ipv4'] = get_ipv4_address()
    payload_data['status'] = 'processing'

def send_integrity_request(payload):
    port = int(os.getenv('PORT', '8000'))
    host = os.getenv('HOST', '127.0.0.1')
    url = f'http://{host}:{port}'

    json_payload = json.dumps(payload)
    headers = {'Content-Type': 'application/json'}
    response = requests.post(url, data=json_payload, headers=headers)

    if response.status_code == 200:
        response_data = response.json()
        print(response_data)
    else:
        # Request failed
        print('Request failed:', response.status_code)

def process_dir(directory):
    batch_limit = 10000
    file_paths = get_file_paths(directory)

    results = []
    if len(file_paths) > batch_limit:
        batches = to_batches(file_paths, batch_limit)
        results_queue = queue.Queue()
        threads = []
        for batch in batches:
            thread = threading.Thread(target=process_batch, args=(batch, results_queue))
            threads.append(thread)
            thread.start()

        for thread in threads:
            thread.join()

        while not results_queue.empty():
            results.extend(results_queue.get())
    else:
        for file in file_paths:
            processed_file = process_file(file)
            if processed_file != None:
                results.append(processed_file)

    if len(results) > 1:
        check_files_integrity(results)


def main():
    module = AnsibleModule(
        argument_spec=dict(
            dirs=dict(type='list', required=True),
        )
    )
    dirs = module.params['dirs']

    threads = []
    for dir in dirs:
        thread = threading.Thread(target=process_dir, args=(dir))
        threads.append(thread)
        thread.start()

    for thread in threads:
        thread.join()


    # Sends a signal indicating that all data was sent out
    payload_data = {}
    payload_data['files'] = []
    payload_data['ipv4'] = get_ipv4_address()
    payload_data['status'] = 'final'

    send_integrity_request(payload_data)

    module.exit_json(changed=False)

if __name__ == '__main__':
    load_dotenv()
    main()
