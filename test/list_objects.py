#! /usr/bin/env python3

import os
import boto3

s3 = boto3.resource('s3',
    endpoint_url="http://127.0.0.1:8080",
    aws_access_key_id="irods_user", 
    aws_secret_access_key="irods_password")

bucket = s3.Bucket('iychoi')
for my_bucket_object in bucket.objects.all():
    print(my_bucket_object.key)