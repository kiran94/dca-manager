import os
import sys

from pyspark.sql import SparkSession
from pyspark.sql import functions as F

try:
    from awsglue.context import GlueContext
    from awsglue.job import Job
    from awsglue.utils import getResolvedOptions
except ImportError:
    pass


def main():
    print('Starting Glue Job')

    spark = SparkSession.builder \
        .config('spark.serializer', 'org.apache.spark.serializer.KryoSerializer') \
        .config('spark.sql.hive.convertMetastoreParquet', 'false') \
        .getOrCreate()

    sc = spark.sparkContext
    glue_context = GlueContext(sc)

    args = getResolvedOptions(sys.argv, [
                              'JOB_NAME', 'input_path', 'output_path', 'glue_database', 'glue_table', 'write_operation'])
    input_path = args['input_path']
    output_path = args['output_path']
    database_name = args['glue_database']
    table_name = args['glue_table']
    write_operation = args['write_operation']

    job = Job(glue_context)
    job.init(args['JOB_NAME'], args)

    partition_path = 'pair'
    key = 'transaction_id,close_time'
    precombine = 'transaction_id'

    config = {
        # Hudi
        'hoodie.datasource.write.table.type': 'COPY_ON_WRITE',
        'hoodie.datasource.write.recordkey.field': key,
        'hoodie.datasource.write.partitionpath.field': partition_path,
        'hoodie.datasource.write.precombine.field': precombine,
        'hoodie.datasource.write.keygenerator.class': 'org.apache.hudi.keygen.ComplexKeyGenerator',
        'hoodie.datasource.write.row.writer.enable': "true",
        'hoodie.datasource.write.hive_style_partitioning': "true",
        'hoodie.cleaner.commits.retained': "1",
        'hoodie.keep.min.commits': "2",
        'hoodie.keep.max.commits': "3",
        'hoodie.clean.automatic': "true",
        'hoodie.parquet.max.file.size': "120",
        'hoodie.bulkinsert.shuffle.parallelism': "100",
        # Parquet
        'hoodie.parquet.compression.codec': "snappy",
        'hoodie.parquet.page.size': "102400",
        'hoodie.parquet.block.size': "268435456",
        # Hive
        'hoodie.datasource.hive_sync.enable': "true",
        'hoodie.datasource.hive_sync.use_jdbc': "false",
        'hoodie.datasource.hive_sync.partition_fields': partition_path,
        'hoodie.datasource.hive_sync.assume_date_partitioning': "false",
        'hoodie.datasource.hive_sync.partition_extractor_class': "org.apache.hudi.hive.MultiPartKeysValueExtractor",
        'hoodie.datasource.hive_sync.database': database_name,
        'hoodie.datasource.hive_sync.table': table_name,
        'hoodie.table.name': table_name,
    }

    # Read data from the JSON files
    print('Reading from ', input_path)
    frame = spark.read.json(input_path)

    frame.printSchema()
    frame.show()

    print('Formatting Columns')
    frame = frame.withColumn('close_time', F.from_unixtime(F.col('close_time'), 'yyyy-MM-dd HH:mm:ss.SS').cast('timestamp'))
    frame = frame.withColumn('open_time', F.from_unixtime(F.col('open_time'), 'yyyy-MM-dd HH:mm:ss.SS').cast('timestamp'))
    frame = frame.withColumn('fee', frame['fee'].cast('double'))
    frame = frame.withColumn('price', frame['price'].cast('double'))
    frame = frame.withColumn('volume', frame['volume'].cast('double'))

    frame.printSchema()
    frame.show()

    # Write to Output
    hudi_output_path = os.path.join(output_path, table_name)
    print(f'Using Config = {config}')
    print(f'Writing to Output {hudi_output_path}')

    frame.write \
        .format('hudi') \
        .option('hoodie.datasource.write.operation', write_operation) \
        .options(**config) \
        .mode('append') \
        .save(hudi_output_path)

    print('DONE')


main()
