import codecs
import io
from tqdm import tqdm
import os

def process_large_file(input_path, output_path, chunk_size=1024):
    total_size = os.path.getsize(input_path)
    processed_size = 0

    with codecs.open(input_path, 'r', encoding='utf-8', errors='replace') as input_file:
        with codecs.open(output_path, 'w', encoding='utf-8') as output_file:
            with tqdm(total=total_size, unit='B', unit_scale=True, desc="Processing") as pbar:
                while True:
                    chunk = input_file.read(chunk_size)
                    if not chunk:
                        break
                    output_file.write(chunk)
                    processed_size += len(chunk)
                    pbar.update(len(chunk))

    return processed_size

# Input and output file paths
input_file_path = '/home/netsec/Desktop/praktika/analyser/data/CorpIdMetadata.tab'
output_file_path = '/home/netsec/Desktop/praktika/analyser/data/SanitizedMetadata.tab'

processed_size = process_large_file(input_file_path, output_file_path)
print(f"Processing complete. Total processed size: {processed_size} bytes.")
