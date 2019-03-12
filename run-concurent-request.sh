#!/bin/bash
for i in {1..50}
do
   curl -s http://localhost:9091/v1alpha1/committer?language=java | jq &
done
