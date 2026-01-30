-- CLMS 任務參數：Scheduler/Fetcher 需 dataset_uid、download_info_id、output_format、output_gcs
ALTER TABLE source_job
  ADD COLUMN IF NOT EXISTS dataset_uid TEXT,
  ADD COLUMN IF NOT EXISTS download_info_id TEXT,
  ADD COLUMN IF NOT EXISTS output_format TEXT,
  ADD COLUMN IF NOT EXISTS output_gcs TEXT;
