SELECT
  event_id,
  event_type,
  session_id,
  publisher_org_id,
  app_bundle_id,
  platform,
  campaign_id,
  brand_org_id,
  creative_id,
  country,
  duration_ms,
  revenue,
  epoch_ms(timestamp) AS event_time
FROM session_events
