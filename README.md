# sys-check

## How to update NIST NSRL Uniqe File Corpus data set
1. Run NIST NSRL Uniqe File Corpus data file re-formatting
    - ```python3 ensure_utf8.py <input file> <output file>```
2. Upload NIST NSRL Uniqe File Corpus data file to a Postgres database
    - ```go run nsrl_to_db.go <formatted data file path>```
    - **Note that this process takes up to a few hours**

## How to use the sys-check tool
1. Add necessary information for **Postgres** database connection to `.env`
    - DB_HOST=
    - DB_PORT=
    - DB_NAME=
    - DB_USER=
    - DB_PASSWORD=
2. Add necessary information about the devices you want to analyze to `hosts` file
    <br>**Add each device to *linux* group**
    - Linux
        - ipv4 address
        - ansible_ssh_user=*(user with sudo privileges)*
        - ansible_password=*(ansible_ssh_user password)*
        - ansible_become=true
        - ansible_become_method=sudo
        - ansible_become_password=*(ansible_ssh_user's sudo password)*
          
</br>**Steps 3-5 are optional if you already have uploaded NIST NSRL Unique File Corpus data to a Postgres database**

3. Download [NIST NSRL Unique File Corpus data file](https://s3.amazonaws.com/docs.nsrl.nist.gov/morealgs/corpus/CorpIdMetadata.tab.zip) and extract it
4. Navigate to `Analyzer -> Pyhton scripts` and launch `ensure_utf8.py` to format NIST NSRL Unique File Corpus data file
    <br>```python3 ensure_utf8.py <CorpIdMetadata.tab file path> <output file name>```
5. Navigate to `Analyzer -> data` and run `nsrl_to_db.go` to upload formatted NIST NSRL Unique File Corpus data to a Postgres database
    <br>```./update_db <formatted CorpIdMetadata.tab data file path>```
6. Launch "Scanner" ansible playbook by navigating to `Scanner` and launching `osinfo.yml` playbook
    <br>```ansible-playbook osinfo.yml -i inventory/hosts```
7. After it is done launch analyzer script by navigating to `Analyzer` and launching `analyzer` program
    <br>```./analyzer /tmp/sys_check/results/scans/<file name>.json```
8. The results of each machine analyzed will be storred at `/tmp/sys_check/results` directory named `report-<machine's ipv4>-YYYY-MM-DD.HH.mm.ss.json`
