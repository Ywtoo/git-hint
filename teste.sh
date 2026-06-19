#!/bin/bash

echo "commit"
echo "checkout"
echo "push"

sleep 2

for i in {1..3}; do
    printf '\033[1A'
    printf '\033[2K'
done