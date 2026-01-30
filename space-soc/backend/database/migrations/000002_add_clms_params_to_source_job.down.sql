ALTER TABLE source_job
  DROP COLUMN IF EXISTS dataset_uid,
  DROP COLUMN IF EXISTS download_info_id,
  DROP COLUMN IF EXISTS output_format,
  DROP COLUMN IF EXISTS output_gcs;
