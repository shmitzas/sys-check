#!/usr/bin/python

from ansible.module_utils.basic import AnsibleModule
import hashlib
import os
import pwd
import grp
import datetime
import threading
import queue

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

def process_batch(batch, results_queue):
    result = []
    for file in batch:
        result.append(process_file(file))
    results_queue.put(result)

def main():
    module = AnsibleModule(
        argument_spec=dict(
            files=dict(type='list', required=True),
        )
    )

    files = module.params['files']

    results = []
    if len(files) > 10000:
        batches = to_batches(files, 10000)
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
        for file in files:
            results.append(process_file(file))

    module.exit_json(changed=False, result=results)

if __name__ == '__main__':
    main()