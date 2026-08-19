import logging
import sys

from checktools import installed, pack_version

# Configure logging to match Go's default behavior
logging.basicConfig(
    format='%(asctime)s - %(message)s',
    level=logging.INFO,
    datefmt='%H:%M:%S'
)

def main():
    logging.info("Checking tools")
    try:
        installed()
    except Exception as e:
        logging.critical(f"Error: {e}")
        sys.exit(1)

    logging.info("Checking pack version")
    try:
        pack_version()
    except Exception as e:
        logging.critical(f"Error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
