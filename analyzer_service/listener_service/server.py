import json
import os
from dotenv import load_dotenv
import subprocess
import logging
from logging.handlers import RotatingFileHandler
from fastapi import FastAPI

import uvicorn

load_dotenv()

app = FastAPI()

# Configure logging
log_formatter = logging.Formatter('[%(asctime)s] %(levelname)s %(message)s', '%Y-%m-%d %H:%M:%S')
log_handler = RotatingFileHandler('service.log', maxBytes=1024 * 1024, backupCount=5)
log_handler.setFormatter(log_formatter)
logger = logging.getLogger('service_logger')
logger.setLevel(logging.INFO)
logger.addHandler(log_handler)

@app.post('/')
async def process_request(data: dict):
    try:
        analyzer_bin = os.getenv('ANALYZER_BIN')
        report_finalizer_bin = os.getenv('REPORT_FINALIZER_BIN')
    except Exception as e:
        logger.error(f'Invalid Analyzer/Report_finalizer paths: {e}')
        return {'ERROR': 422, 'message': 'nvalid Analyzer/report_finalizer paths'}
    try:
        with open("request_data.json", "w") as json_file:
            json.dump(data, json_file)

    except Exception as e:
        logger.error(f'Invalid JSON data: {e}')
        return {'ERROR': 422, 'message': 'Invalid JSON data'}
    if data['status'] == "processing":
        try:
            subprocess.run([analyzer_bin], input=json.dumps(data), capture_output=True, text=True)
        except Exception as e:
            logger.exception(f'Unexpected error occurred: {e}')
            return {'ERROR': 422, 'message': 'Anazlyzer crashed'}
        return 'OK'
    if data['status'] == "final":
        try:
            hostname = data['metadata']['hostname']
            ipv4_address = data['metadata']['ipv4_address']
            subprocess.run([report_finalizer_bin, hostname, ipv4_address], capture_output=True, text=True)
        except Exception as e:
            logger.exception(f'Unexpected error occurred: {e}')
            return {'ERROR': 422, 'message': 'Report finalizer crashed'}
        return 'OK'
    


if __name__ == '__main__':
    port = int(os.getenv('PORT', '8000'))
    host = os.getenv('HOST', '127.0.0.1')
    workers = int(os.getenv('WORKERS', '10'))
    uvicorn.run("server:app", host=host, port=port, workers=workers)