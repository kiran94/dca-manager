#!/bin/bash

# This is just a test script to quickly bring up an interactive
# shell with all the required JARS & configuration
# AWS credentials are derived from environment.

SPARK_HADOOP_AWS=org.apache.hadoop:hadoop-aws:2.8.5
SPARK_AWS_SDK=com.amazonaws:aws-java-sdk:1.11.659
SPARK_HADOOP_COMMON=org.apache.hadoop:hadoop-common:2.8.5

# https://hudi.apache.org/docs/quick-start-guide/
SPARK_HUDI=org.apache.hudi:hudi-spark3-bundle_2.12:0.10.0
SPARK_AVRO=org.apache.spark:spark-avro_2.12:3.1.2

SPARK_PACKAGES=$SPARK_HADOOP_AWS,$SPARK_AWS_SDK,$SPARK_HADOOP_COMMON,$SPARK_HUDI,$SPARK_AVRO
SPARK_CONF='spark.serializer=org.apache.spark.serializer.KryoSerializer'

pyspark --packages $SPARK_PACKAGES --conf $SPARK_CONF

# EXAMPLES:
# > hudi_frame = spark.read.format("hudi").load("s3a://dca-manager/glue/hudi/transactions")
# > hudi_frame.printSchema()
# > hudi_frame.show(n=5)
