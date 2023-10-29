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
        # Verify that the data received is a valid JSON
        json_data = json.loads(data_to_json)
    except Exception as e:
        logger.error(f'Invalid JSON data: {e}')
        print(data)
        return {'ERROR': 422, 'message': 'Invalid JSON data'}

    try:
        # Launch the compiled Go script and pass the JSON data to it
        subprocess.run(['/home/netsec/Desktop/praktika/analyzer_service/analyzer/analyzer'], input=json.dumps(json_data), capture_output=True, text=True)
    except Exception as e:
        logger.exception(f'Unexpected error occurred: {e}')
        return {'ERROR': 422, 'message': 'Anazlyzer crashed'}
    return 'OK'

if __name__ == '__main__':
    port = int(os.getenv('PORT', '8000'))
    host = os.getenv('HOST', '127.0.0.1')
    workers = int(os.getenv('WORKERS', '4'))  # Number of worker processes
    uvicorn.run("server:app", host=host, port=port, workers=workers)