import json
import logging
import os
import sys
import pprint
from typing import Dict, List

from pyspark.sql import SparkSession
from pyspark.sql import functions as F

try:
    from awsglue.context import GlueContext
    from awsglue.job import Job
    from awsglue.utils import getResolvedOptions
except ImportError:
    pass


LOGGING_FORMAT = os.environ.get("LOGGING_FORMAT", logging.BASIC_FORMAT)
LOGGING_LEVEL = os.environ.get("LOGGING_LEVEL", logging.INFO)

logging.basicConfig(format=LOGGING_FORMAT, level=LOGGING_LEVEL)
logger = logging.getLogger()


def main():
    logging.info('Starting Glue Job')

    spark = SparkSession.builder \
        .config('spark.serializer', 'org.apache.spark.serializer.KryoSerializer') \
        .config('spark.sql.hive.convertMetastoreParquet', 'false') \
        .getOrCreate()

    sc = spark.sparkContext
    glue_context = GlueContext(sc)

    args = getResolvedOptions(sys.argv, [
                              'JOB_NAME', 'input_path', 'output_path', 'glue_database', 'glue_table', 'write_operation', 'additional_columns'])

    logging.debug(args)
    input_path = args['input_path']
    output_path = args['output_path']
    database_name = args['glue_database']
    table_name = args['glue_table']
    write_operation = args['write_operation']
    additional_columns = args['additional_columns']

    valid_write_operation = ['insert', 'upsert', 'bulk_insert', 'delete']
    if write_operation not in valid_write_operation:
        logger.error(f'write_operation must be one of options: {valid_write_operation}. But was: {write_operation}')
        exit(1)

    job = Job(glue_context)
    job.init(args['JOB_NAME'], args)

    partition_path = 'pair'
    key = 'transaction_id,close_time'
    precombine = 'transaction_id'

    logger.debug(f'Using Partition Path {partition_path}, Key {key}, Precombine: {precombine}')
    config = {
        # Hudi
        'hoodie.table.name': table_name,
        'hoodie.datasource.write.table.type': 'COPY_ON_WRITE',
        'hoodie.datasource.write.recordkey.field': key,
        'hoodie.datasource.write.partitionpath.field': partition_path,
        'hoodie.datasource.write.precombine.field': precombine,
        'hoodie.datasource.write.keygenerator.class': 'org.apache.hudi.keygen.ComplexKeyGenerator',
        'hoodie.datasource.write.hive_style_partitioning': "true",
        # Parquet
        'hoodie.parquet.compression.codec': "snappy",
        # Hive
        'hoodie.datasource.hive_sync.enable': "true",
        'hoodie.datasource.hive_sync.use_jdbc': "false",
        'hoodie.datasource.hive_sync.partition_fields': partition_path,
        'hoodie.datasource.hive_sync.assume_date_partitioning': "false",
        'hoodie.datasource.hive_sync.partition_extractor_class': "org.apache.hudi.hive.MultiPartKeysValueExtractor",
        'hoodie.datasource.hive_sync.database': database_name,
        'hoodie.datasource.hive_sync.table': table_name
    }

    # Read data from the JSON files
    logger.info(f'Reading from {input_path}')
    frame = spark.read.json(input_path)

    frame.printSchema()
    frame.show()

    logger.debug('Formatting Columns')
    frame = frame.withColumn('close_time', F.from_unixtime(F.col('close_time'), 'yyyy-MM-dd HH:mm:ss.SS').cast('timestamp'))
    frame = frame.withColumn('open_time', F.from_unixtime(F.col('open_time'), 'yyyy-MM-dd HH:mm:ss.SS').cast('timestamp'))
    frame = frame.withColumn('fee', frame['fee'].cast('double'))
    frame = frame.withColumn('price', frame['price'].cast('double'))
    frame = frame.withColumn('volume', frame['volume'].cast('double'))

    # Columns which are not in the source data file
    # but need to be applied to output frame
    if additional_columns and additional_columns != "none":
        logger.debug('Adding Partition Columns')

        loaded_additional_columns: List[Dict[str, str]] = json.loads(additional_columns)
        logger.debug(f'Loaded Additional Columns {loaded_additional_columns}')

        if loaded_additional_columns:
            for column_name, value in loaded_additional_columns.items():
                logger.info(f'Adding Column {column_name}, Value: {value}')
                frame = frame.withColumn(column_name, F.lit(value))

    frame.printSchema()
    frame.show()

    # Write to Output
    hudi_output_path = os.path.join(output_path, table_name)
    logger.info('Hudi Configuration: \n ' + pprint.pformat(config))
    logger.info(f'Writing with operation {write_operation} to path: {hudi_output_path}')

    frame.write \
        .format('hudi') \
        .option('hoodie.datasource.write.operation', write_operation) \
        .options(**config) \
        .mode('append') \
        .save(hudi_output_path)

    print('DONE')


main()
