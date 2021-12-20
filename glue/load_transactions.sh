#!/bin/bash
# NOTE: this is just a quick testing script
# https://hudi.apache.org/docs/quick-start-guide/

spark-submit \
    --packages org.apache.hudi:hudi-spark3-bundle_2.12:0.10.0,org.apache.spark:spark-avro_2.12:3.1.2 \
    --conf 'spark.serializer=org.apache.spark.serializer.KryoSerializer' \
    ./scripts/load_transactions.py
