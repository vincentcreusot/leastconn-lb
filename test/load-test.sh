#!/bin/bash

# Set number of parallel requests
NUM_REQUESTS=100

# Forwarder URL
URL=http://localhost:8888

# Array to store errors
ERRORS=()

# Launch requests in background
for i in $(seq 1 $NUM_REQUESTS); do
  response=$(curl -s -w "%{http_code}" $URL -o /dev/null&)
  http_code=$(tail -n1 <<< "$response")
  if [ $http_code -ne 200 ]; then
    ERRORS+=("Request $i failed with HTTP code $http_code")
  fi
done


# Print errors
if [ ${#ERRORS[@]} -gt 0 ]; then
  echo "Errors:"
  for err in "${ERRORS[@]}"; do
    echo $err
  done
fi