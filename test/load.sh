#!/bin/bash
for j in $(seq 1 10)
    do bash load-test.sh $j&
done

wait