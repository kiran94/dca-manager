
spark-submit \
  --packages org.apache.hudi:hudi-spark3-bundle_2.12:0.8.0,org.apache.spark:spark-avro_2.12:3.0.1 \
  --conf 'spark.serializer=org.apache.spark.serializer.KryoSerializer' ./scripts/load_transactions.py



  # --py-files ../../aws-glue-libs/
# hudi-spark-bundle_2.11-0.8.0.jar
# spark-avro_2.11-2.4.7.jar
