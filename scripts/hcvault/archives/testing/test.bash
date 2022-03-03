#!/bin/bash
./bin/kubectl apply -f scripts/secret.yaml
for (( c=1; c<=100; c++ ))
do
   ./bin/kubectl get secret data-test  -o yaml > /dev/null
done

for (( c=1; c<=100; c++ ))
do
   ./bin/kubectl get secret data-test2  -o yaml > /dev/null
done
