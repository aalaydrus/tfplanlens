package analyze

// builtinStateful lists resource types that hold data that is hard or
// impossible to recreate (SPEC 5.2).
var builtinStateful = []string{
	"aws_db_instance", "aws_rds_cluster", "aws_rds_cluster_instance",
	"aws_docdb_cluster", "aws_neptune_cluster", "aws_redshift_cluster",
	"aws_s3_bucket", "aws_ebs_volume", "aws_kms_key", "aws_route53_zone",
	"aws_efs_file_system", "aws_dynamodb_table",
	"aws_elasticache_cluster", "aws_elasticache_replication_group",
	"aws_opensearch_domain", "aws_elasticsearch_domain",
}
