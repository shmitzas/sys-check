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
        data_to_json = json.dumps(data)
        json_data = json.loads(data_to_json)
    except Exception as e:
        logger.error(f'Invalid JSON data: {e}')
        print(data)
        return {'ERROR': 422, 'message': 'Invalid JSON data'}
    if json_data['status'] == "processing":
        try:
            subprocess.run(['/home/netsec/Desktop/praktika/analyzer_service/analyzer/analyzer'], input=json.dumps(json_data), capture_output=True, text=True)
        except Exception as e:
            logger.exception(f'Unexpected error occurred: {e}')
            return {'ERROR': 422, 'message': 'Anazlyzer crashed'}
        return 'OK'
    if json_data['status'] == "final":
        try:
            hostname = json_data['metadata']['hostname']
            ipv4_address = json_data['metadata']['ipv4_address']
            subprocess.run(['/home/netsec/Desktop/praktika/analyzer_service/report_finalizer/report_finalizer', hostname, ipv4_address], capture_output=True, text=True)
        except Exception as e:
            logger.exception(f'Unexpected error occurred: {e}')
            return {'ERROR': 422, 'message': 'Report finalizer crashed'}
        return 'OK'
    


if __name__ == '__main__':
    port = int(os.getenv('PORT', '8000'))
    host = os.getenv('HOST', '127.0.0.1')
    workers = int(os.getenv('WORKERS', '10'))
    uvicorn.run("server:app", host=host, port=port, workers=workers)