# How to use sys-check tool

1. Setup and configure sys-check system
2. Start Analyzer listener service

## Start Analyzer listener service
1. Navigate to the cloned repository's analyzer_service directory
    ```
    cd <cloned sys-check repository path>/analyzer_service/listener
    ```
2. Start listener service
    ```
    ./listener
    ```
3. Target computer's file system's integrity report can be found at `<REPORTS_DIR>/reports-<target computer's ipv4 address>/final-report.json`

## Application for scanning target computers
1. Navigate to the cloned repository's scanner directory
    ```
    cd <cloned sys-check repository path>/scanner
    ```
2. Configure `hosts` and `file_scan_linux.yml` files as per setup instructions
- **NOTE: Target computers must have an ssh server (for example openssh-server) installed and running**
3. Start scanner
    ```
    ansible-playbook osinfo.yml -i inventory/hosts
    ```

## Upload known data to the database
- NIST NSRL Unique File Corpus data file
    - Reformat data file
        ```
        cd <cloned sys-check repository path>/upload_known_data/upload_nsrl_data/formatter
        ```
        ```
        python3 ensure_utf8.py <full path to data file> <full path to output file>
        ```
    - Upload data
        ```
        cd <cloned sys-check repository path>/upload_known_data/upload_nsrl_data
        ```
        ```
        ./upload_nsrl_data <full path to reformated data file>
        ```
- Verified data JSON file
    ```
    cd <cloned sys-check repository path>/upload_known_data/upload_verified_data
    ```
    ```
    ./upload_verified_data <full path to data file>
    ```
- Malicious data JSON file
    ```
    cd <cloned sys-check repository path>/upload_known_data/upload_malicious_data
    ```
    ```
    ./upload_malicious_data <full path to data file>
    ```

# Setup
- **NOTE: Setup only on Unix based OS, preferably Linux**
- **NOTE: This system was developed for Debian based Linux distributions**

## Setup backend server
1. Clone this repository
    ```
    git clone https://github.com/shmitzas/sys-check.git
    ```
2. Navigate to the cloned repository's analyzer_service directory
    ```
    cd <cloned sys-check repository path>/analyzer_service
    ```
3. Launch backend setup script
    ```
    ./setup_backend.sh <cloned sys-check repository's full path>
    ```
4. Go to environment configuration file location
    ```
    cd ~/.sys-check/.env/
    ```
5. Fill out data in environment files.
- Example of different type of data formats
    - IPv4 address: `DB_HOST=127.0.0.1`
    - Port: `DB_PORT=5432`
    - `String` type variables: `DB_NAME=sys_check`
    - File path: `REPORTS_DIR=/tmp/sys_check/reports`

## Setup Database server
1. Clone this repository
    ```
    git clone https://github.com/shmitzas/sys-check.git
    ```
2. Install dependencies
    ```
    sudo apt install -y curl gpg gnupg2 software-properties-common apt-transport-https lsb-release ca-certificates
    ```
3. Install PostgreSQL 13
    ```
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc|sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg
    ```
    ```
    echo "deb http://apt.postgresql.org/pub/repos/apt/ `lsb_release -cs`-pgdg main" |sudo tee  /etc/apt/sources.list.d/pgdg.list
    ```
    ```
    sudo apt update
    ```
    ```
    sudo apt install -y postgresql-13 postgresql-client-13
    ```
4. Start main cluster
    ```
    sudo systemctl start postgresql@13-main
    ```
5. Navigate to the cloned repository's database directory
    ```
    cd <cloned sys-check repository path>/database
    ```
6. Copy database creation scripts to /tmp/
    ```
    cp db_setup.sql.example /tmp/db_setup.sql 
    ```
    ```
    cp db_users.sql.example /tmp/db_users.sql 
    ```
7. Fill out `<placeholder text>` in `/tmp/db_setup.sql ` and `/tmp/db_users.sql` files with actual data

8. Change to postgres user
    ```
    sudo su postgres
    ```
9. Create database and users from setup files
    ```
    cd /tmp
    ```
    ```
    psql -U postgres -f db_setup.sql
    ```
    ```
    psql -U postgres -f db_users.sql
    ```
    ```
    exit
    ```
10. Configure database connection limit
- In `/etc/postgresql/13/main/postgresql.conf` file under `Connection Settings` change:
        - `max_connections = 100` from `100` to `200000` or any number between 1 and 200000
11. Add database user to configuration file
- In `/etc/postgresql/13/main/pg_hba.conf` file under `Database administrative login by Unix domain socket` add:
    ```
    local   <database name>     <database user>                                     md5
    ```
12. Configure database to be accessed from outside database server (if needed)
- In `/etc/postgresql/13/main/postgresql.conf` file under `Connection Settings` change:
    - `listen_addresses = 'localhost'` from `'localhost'` to `'*'` or a specific IPv4 address
- In `/etc/postgresql/13/main/pg_hba.conf` file under `Database administrative login by Unix domain socket` add:
    - If you do not need any other authorization besides database's user use `trust`, otherwise use `md5` or other
    ```
    host    <database name>     <database user>     <specific IPv4 address>/32      trust
    ```
- Allow database port through your firewall
13. Restart main cluster to refresh configuration
    ```
    sudo systemctl restart postgresql@13-main
    ```

## Setup environment for uploading know file data
1. Clone this repository
    ```
    git clone https://github.com/shmitzas/sys-check.git
    ```
2. Navigate to the cloned repository's upload_known_data directory
    ```
    cd <cloned sys-check repository path>/upload_known_data
    ```
3. Launch environment setup script
    ```
    ./setup_env.sh <cloned sys-check repository's full path>
    ```
4. Go to environment configuration file location
    ```
    cd ~/.sys-check/.env/
    ```
5. Fill out data in environment files.
- Example of different type of data formats
    - IPv4 address: `DB_HOST=127.0.0.1`
    - Port: `DB_PORT=5432`
    - `String` type variables: `DB_NAME=sys_check`

## Setup Application for scanning target computers
1. Clone this repository
    ```
    git clone https://github.com/shmitzas/sys-check.git
    ```
2. Navigate to the cloned repository's analyzer_service directory
    ```
    cd <cloned sys-check repository path>/scanner
    ```
3. Run environment setup script
    ```
    ./setup_env.sh
    ```
4. Create a copy of `hosts.example` and name it `hosts`
    ```
    cd inventory/
    ```
    ```
    cp hosts.example hosts
    ```
5. Fill out `hosts` file with necessary data for remote access to target computers
6. Go to tasks directory
    ```
    cd <cloned sys-check repository path>/scanner/tasks
    ```
7. Edit `file_scan_linux.yml` file
- To configure what directories to scan edit `directories` list variable by adding or removing directories
- To configure analyzer server's address edit `service_host` and `service_port` variables to match values defined at `~/.sys_check/.env/listener.env`

## How to rebuild .go files after modifying them
- To rebuild analyzer
    - Navigate to analyzer directory
        ```
        cd <cloned sys-check repository path>/analyzer_service/analyzer
        ```
    - Rebuild analyzer
        ```
        go build analyzer
        ```
- To rebuild listener
    - Navigate to listener directory
        ```
        cd <cloned sys-check repository path>/analyzer_service/listener
        ```
    - Rebuild listener
        ```
        go build listener
        ```
- To rebuild report finalizer
    - Navigate to report finalizer directory
        ```
        cd <cloned sys-check repository path>/analyzer_service/report_finalizer
        ```
    - Rebuild report_finalizer
        ```
        go build report_finalizer
        ```
- To rebuild known data uploading programs
    - Navigate to known data uploading program directory
        ```
        cd <cloned sys-check repository path>/known data uploading
        ```
    - Rebuild upload_verified_data
        ```
        go build upload_verified_data
        ```
    - Rebuild upload_nsrl_data
        ```
        go build upload_verified_data
        ```
    - Rebuild upload_verified_data
        ```
        go build upload_malicious_data
        ```