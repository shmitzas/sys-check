import codecs
from tqdm import tqdm
import os
import sys

def process_large_file(input_path, output_path, chunk_size=1000000):
    total_size = os.path.getsize(input_path)

    try:
        with codecs.open(input_path, 'r', encoding='utf-8', errors='replace') as input_file:
            with codecs.open(output_path, 'w', encoding='utf-8') as output_file:
                with tqdm(total=total_size, unit='B', unit_scale=True, desc="Processing") as pbar:
                    while True:
                        chunk = input_file.read(chunk_size)
                        if not chunk:
                            break
                        output_file.write(chunk)
                        pbar.update(len(chunk))
    except FileNotFoundError:
        print(f'Input file not found: {input_path}')
        print('Usage: python3 ensure_utf8.py <input file path> <output file path>')
        exit()

def main():
    if len(sys.argv) < 2:
        print('Usage: python3 ensure_utf8.py <input file path> <output file path>')
        exit()
        
    input_file_path = sys.argv[1]
    output_file_path = sys.argv[2]

    process_large_file(input_file_path, output_file_path)
    print('Processing complete.')

if __name__ == '__main__':
    main()